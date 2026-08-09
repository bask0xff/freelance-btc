package order

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"freelance-btc/internal/bitcoin"
)

// newTestBitcoinClient поднимает локальный httptest-сервер, отвечающий на JSON-RPC как bitcoind,
// чтобы протестировать HireFreelancer (вызывает GetNewAddress) без реальной ноды.
func newTestBitcoinClient(t *testing.T, fixedAddress string) (*bitcoin.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		resp := map[string]interface{}{"id": req.ID, "error": nil, "result": nil}
		if req.Method == "getnewaddress" {
			resp["result"] = fixedAddress
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	client := bitcoin.NewClient(u.Hostname(), port, "user", "pass", "passphrase")
	return client, srv.Close
}

func TestCreateOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db, nil) // rpc не используется в CreateOrder — адрес теперь выдаётся только при найме

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO orders`)).
		WithArgs(int64(1), "Landing page", "Собрать лендинг", 0.05).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	o, err := svc.CreateOrder(1, "Landing page", "Собрать лендинг", 0.05)
	require.NoError(t, err)
	require.Equal(t, int64(10), o.ID)
	require.Equal(t, "open", o.Status)
	require.Empty(t, o.Address, "адрес не должен генерироваться при создании заказа — только при найме")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHireFreelancer_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rpc, closeSrv := newTestBitcoinClient(t, "bc1qtestaddressxxxxxxxxxxxxxxxxxxxxxxxxxx")
	defer closeSrv()

	svc := NewService(db, rpc)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status, client_id, amount_btc FROM orders WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "client_id", "amount_btc"}).AddRow("open", int64(1), 0.05))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT freelancer_id, proposed_amount_btc, status FROM applications WHERE id = $1 AND order_id = $2`)).
		WithArgs(int64(5), int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"freelancer_id", "proposed_amount_btc", "status"}).AddRow(int64(2), nil, "pending"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE applications SET status = 'accepted'`)).
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE applications SET status = 'rejected'`)).
		WithArgs(int64(10), int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE orders SET freelancer_id = $1, amount_btc = $2, status = 'awaiting_payment' WHERE id = $3`)).
		WithArgs(int64(2), 0.05, int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO btc_addresses`)).
		WithArgs(int64(10), "bc1qtestaddressxxxxxxxxxxxxxxxxxxxxxxxxxx", "order_10").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	o, err := svc.HireFreelancer(10, 1, 5)
	require.NoError(t, err)
	require.Equal(t, "awaiting_payment", o.Status)
	require.Equal(t, "bc1qtestaddressxxxxxxxxxxxxxxxxxxxxxxxxxx", o.Address)
	require.True(t, o.FreelancerID.Valid)
	require.Equal(t, int64(2), o.FreelancerID.Int64)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHireFreelancer_NotOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db, nil)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status, client_id, amount_btc FROM orders WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "client_id", "amount_btc"}).AddRow("open", int64(999), 0.05))
	mock.ExpectRollback()

	_, err = svc.HireFreelancer(10, 1, 5) // clientID=1, но реальный владелец заказа — 999
	require.Error(t, err)
	require.Contains(t, err.Error(), "only the order's client")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHireFreelancer_OrderNotOpen(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db, nil)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status, client_id, amount_btc FROM orders WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "client_id", "amount_btc"}).AddRow("awaiting_payment", int64(1), 0.05))
	mock.ExpectRollback()

	_, err = svc.HireFreelancer(10, 1, 5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not open")

	require.NoError(t, mock.ExpectationsWereMet())
}
