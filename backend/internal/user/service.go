package user

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type User struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	DisplayName   string `json:"display_name"`
	PayoutAddress string `json:"payout_address,omitempty"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Register(email, password, displayName, role string) (*User, error) {
	if !emailRe.MatchString(email) {
		return nil, errors.New("invalid email")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}
	if role != "client" && role != "freelancer" {
		return nil, errors.New("role must be 'client' or 'freelancer'")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var id int64
	err = s.db.QueryRow(
		`INSERT INTO users (email, password_hash, role, display_name) VALUES ($1, $2, $3, $4) RETURNING id`,
		email, string(hash), role, displayName,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errors.New("email already registered")
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &User{ID: id, Email: email, Role: role, DisplayName: displayName}, nil
}

// Authenticate проверяет пароль и возвращает пользователя, если всё ок.
func (s *Service) Authenticate(email, password string) (*User, error) {
	var (
		u    User
		hash string
	)
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, role, display_name, COALESCE(payout_address, '') FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &hash, &u.Role, &u.DisplayName, &u.PayoutAddress)
	if err == sql.ErrNoRows {
		return nil, errors.New("invalid email or password")
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return &u, nil
}

func (s *Service) GetByID(id int64) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, email, role, display_name, COALESCE(payout_address, '') FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Role, &u.DisplayName, &u.PayoutAddress)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SetPayoutAddress — фрилансер указывает свой BTC-адрес для получения выплат из эскроу.
func (s *Service) SetPayoutAddress(userID int64, address string) error {
	_, err := s.db.Exec(`UPDATE users SET payout_address = $1 WHERE id = $2`, address, userID)
	return err
}

// isUniqueViolation — код 23505 в PostgreSQL означает нарушение UNIQUE/PRIMARY KEY.
// См. https://www.postgresql.org/docs/current/errcodes-appendix.html
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
