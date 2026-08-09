package user

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WithArgs("alice@example.com", sqlmock.AnyArg(), "client", "Alice").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	u, err := svc.Register("alice@example.com", "supersecret", "Alice", "client")
	require.NoError(t, err)
	assert.Equal(t, int64(1), u.ID)
	assert.Equal(t, "client", u.Role)
	assert.Equal(t, "alice@example.com", u.Email)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_InvalidEmail(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)
	_, err = svc.Register("not-an-email", "supersecret", "Alice", "client")
	require.Error(t, err)
}

func TestRegister_ShortPassword(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)
	_, err = svc.Register("alice@example.com", "short", "Alice", "client")
	require.Error(t, err)
}

func TestRegister_InvalidRole(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)
	_, err = svc.Register("alice@example.com", "supersecret", "Alice", "admin")
	require.Error(t, err, "регистрация с ролью admin должна быть запрещена — эта роль не выставляется через публичный API")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	// Код 23505 — реальный код PostgreSQL для нарушения UNIQUE/PRIMARY KEY,
	// именно на него смотрит isUniqueViolation().
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WithArgs("alice@example.com", sqlmock.AnyArg(), "client", "Alice").
		WillReturnError(&pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"})

	_, err = svc.Register("alice@example.com", "supersecret", "Alice", "client")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, role, display_name`)).
		WithArgs("alice@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "role", "display_name", "payout_address"}).
			AddRow(1, "alice@example.com", string(hash), "client", "Alice", ""))

	_, err = svc.Authenticate("alice@example.com", "wrong-password")
	require.Error(t, err)
}

func TestAuthenticate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewService(db)

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, role, display_name`)).
		WithArgs("alice@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "role", "display_name", "payout_address"}).
			AddRow(1, "alice@example.com", string(hash), "client", "Alice", ""))

	u, err := svc.Authenticate("alice@example.com", "correct-password")
	require.NoError(t, err)
	assert.Equal(t, int64(1), u.ID)
}
