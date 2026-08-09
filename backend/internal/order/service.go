package order

import (
	"database/sql"
	"errors"
	"fmt"

	"freelance-btc/internal/bitcoin"
)

type Service struct {
	db  *sql.DB
	rpc *bitcoin.Client
}

func NewService(db *sql.DB, rpc *bitcoin.Client) *Service {
	return &Service{db: db, rpc: rpc}
}

type Order struct {
	ID           int64
	ClientID     int64
	FreelancerID sql.NullInt64
	Title        string
	AmountBTC    float64
	Status       string
	Address      string
}

// CreateOrder создаёт заказ в статусе "open" — фрилансеры могут откликаться.
// BTC-адрес НЕ генерируется здесь: финальная сумма может отличаться от заявленной (фрилансер
// вправе предложить свою цену в отклике), поэтому адрес выдаётся только в момент найма (HireFreelancer).
func (s *Service) CreateOrder(clientID int64, title, description string, amountBTC float64) (*Order, error) {
	var orderID int64
	err := s.db.QueryRow(
		`INSERT INTO orders (client_id, title, description, amount_btc, status) VALUES ($1, $2, $3, $4, 'open') RETURNING id`,
		clientID, title, description, amountBTC,
	).Scan(&orderID)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	return &Order{ID: orderID, ClientID: clientID, Title: title, AmountBTC: amountBTC, Status: "open"}, nil
}

// HireFreelancer — клиент нанимает фрилансера по одному из его откликов.
// Проставляет freelancer_id, при необходимости обновляет amount_btc (если фрилансер предложил свою цену),
// генерирует депозитный BTC-адрес под заказ и переводит заказ в awaiting_payment.
// Остальные pending-отклики на этот заказ автоматически отклоняются.
func (s *Service) HireFreelancer(orderID, clientID, applicationID int64) (*Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		orderStatus     string
		orderClientID   int64
		amountBTC       float64
		freelancerID    int64
		proposedAmount  sql.NullFloat64
		applicationStat string
	)
	err = tx.QueryRow(`SELECT status, client_id, amount_btc FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&orderStatus, &orderClientID, &amountBTC)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	if orderClientID != clientID {
		return nil, errors.New("only the order's client can hire")
	}
	if orderStatus != "open" {
		return nil, fmt.Errorf("order is not open (status: %s)", orderStatus)
	}

	err = tx.QueryRow(`SELECT freelancer_id, proposed_amount_btc, status FROM applications WHERE id = $1 AND order_id = $2`, applicationID, orderID).
		Scan(&freelancerID, &proposedAmount, &applicationStat)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("application not found for this order")
		}
		return nil, err
	}
	if applicationStat != "pending" {
		return nil, fmt.Errorf("application is not pending (status: %s)", applicationStat)
	}
	if proposedAmount.Valid {
		amountBTC = proposedAmount.Float64
	}

	if _, err := tx.Exec(`UPDATE applications SET status = 'accepted', updated_at = NOW() WHERE id = $1`, applicationID); err != nil {
		return nil, fmt.Errorf("accept application: %w", err)
	}
	if _, err := tx.Exec(`UPDATE applications SET status = 'rejected', updated_at = NOW() WHERE order_id = $1 AND id != $2 AND status = 'pending'`, orderID, applicationID); err != nil {
		return nil, fmt.Errorf("reject other applications: %w", err)
	}

	label := fmt.Sprintf("order_%d", orderID)
	address, err := s.rpc.GetNewAddress(label, "bech32")
	if err != nil {
		return nil, fmt.Errorf("getnewaddress: %w", err)
	}

	if _, err := tx.Exec(
		`UPDATE orders SET freelancer_id = $1, amount_btc = $2, status = 'awaiting_payment' WHERE id = $3`,
		freelancerID, amountBTC, orderID,
	); err != nil {
		return nil, fmt.Errorf("update order: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO btc_addresses (order_id, address, address_type, label) VALUES ($1, $2, 'bech32', $3)`,
		orderID, address, label,
	); err != nil {
		return nil, fmt.Errorf("insert address: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Order{ID: orderID, ClientID: clientID, FreelancerID: sql.NullInt64{Int64: freelancerID, Valid: true}, AmountBTC: amountBTC, Status: "awaiting_payment", Address: address}, nil
}

// ListOpen — открытые заказы, на которые фрилансеры ещё могут откликнуться.
func (s *Service) ListOpen() ([]Order, error) {
	rows, err := s.db.Query(`SELECT id, client_id, title, amount_btc, status FROM orders WHERE status = 'open' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.ClientID, &o.Title, &o.AmountBTC, &o.Status); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ReleaseEscrow — вызывается, когда клиент подтвердил сдачу работы (или разрешён спор в пользу фрилансера).
// Отправляет средства с общего хот-кошелька на payout_address фрилансера.
// ВАЖНО: это кастодиальная модель (см. обсуждение) — деньги физически лежат в общем кошельке ноды,
// привязка к заказу — только на уровне вашей БД (btc_addresses/btc_transactions).
func (s *Service) ReleaseEscrow(orderID int64) (string, error) {
	var amountBTC float64
	var payoutAddress sql.NullString
	var status string
	err := s.db.QueryRow(`
		SELECT o.amount_btc, u.payout_address, o.status
		FROM orders o
		JOIN users u ON u.id = o.freelancer_id
		WHERE o.id = $1
	`, orderID).Scan(&amountBTC, &payoutAddress, &status)
	if err != nil {
		return "", fmt.Errorf("lookup order: %w", err)
	}
	if status != "delivered" {
		return "", fmt.Errorf("order status %q is not eligible for release (need 'delivered')", status)
	}
	if !payoutAddress.Valid || payoutAddress.String == "" {
		return "", errors.New("freelancer has no payout_address set")
	}

	txid, err := s.rpc.SendWithUnlock(payoutAddress.String, amountBTC, fmt.Sprintf("escrow release order_%d", orderID))
	if err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	if _, err := s.db.Exec(`
		INSERT INTO btc_transactions (order_id, address, txid, amount_btc, satoshi, confirmations, direction, status)
		VALUES ($1, $2, $3, $4, $5, 0, 'out', 'pending')
	`, orderID, payoutAddress.String, txid, amountBTC, int64(amountBTC*1e8)); err != nil {
		return "", fmt.Errorf("log outgoing tx: %w", err)
	}

	if _, err := s.db.Exec(`UPDATE orders SET status = 'completed' WHERE id = $1`, orderID); err != nil {
		return "", err
	}

	return txid, nil
}
