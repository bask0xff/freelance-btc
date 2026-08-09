package application

import (
	"database/sql"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestApply_OrderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM orders WHERE id = $1`)).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	_, err = svc.Apply(99, 1, "hello", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestApply_OrderNotOpen(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM orders WHERE id = $1`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("awaiting_payment"))

	_, err = svc.Apply(10, 1, "hello", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not open")
}

func TestApply_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM orders WHERE id = $1`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("open"))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO applications`)).
		WithArgs(int64(10), int64(2), "буду рад помочь", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	a, err := svc.Apply(10, 2, "буду рад помочь", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), a.ID)
	require.Equal(t, "pending", a.Status)
}

func TestApply_WithProposedAmount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)
	amount := 0.02

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM orders WHERE id = $1`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("open"))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO applications`)).
		WithArgs(int64(10), int64(2), "готов за меньшую цену", amount).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

	a, err := svc.Apply(10, 2, "готов за меньшую цену", &amount)
	require.NoError(t, err)
	require.NotNil(t, a.ProposedAmount)
	require.Equal(t, amount, *a.ProposedAmount)
}

func TestWithdraw_NotFoundOrNotYours(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE applications SET status = 'withdrawn'`)).
		WithArgs(int64(1), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = svc.Withdraw(1, 2)
	require.Error(t, err)
}

func TestWithdraw_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE applications SET status = 'withdrawn'`)).
		WithArgs(int64(1), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = svc.Withdraw(1, 2)
	require.NoError(t, err)
}
