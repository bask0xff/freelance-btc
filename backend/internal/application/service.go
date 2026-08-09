package application

import (
	"database/sql"
	"errors"
	"fmt"
)

type Application struct {
	ID                int64           `json:"id"`
	OrderID           int64           `json:"order_id"`
	FreelancerID      int64           `json:"freelancer_id"`
	FreelancerName    string          `json:"freelancer_name,omitempty"`
	Message           string          `json:"message"`
	ProposedAmountBTC sql.NullFloat64 `json:"-"`
	ProposedAmount    *float64        `json:"proposed_amount_btc,omitempty"`
	Status            string          `json:"status"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Apply — фрилансер откликается на открытый заказ. Один активный отклик на заказ
// (см. CONSTRAINT uniq_application в схеме) — повторный отклик обновляет предыдущий и снова
// переводит его в pending (например, если отклик был отклонён при найме другого фрилансера,
// а заказ почему-то снова открылся).
func (s *Service) Apply(orderID, freelancerID int64, message string, proposedAmount *float64) (*Application, error) {
	var status string
	if err := s.db.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	if status != "open" {
		return nil, fmt.Errorf("order is not open for applications (status: %s)", status)
	}

	var id int64
	err := s.db.QueryRow(
		`INSERT INTO applications (order_id, freelancer_id, message, proposed_amount_btc, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 ON CONFLICT ON CONSTRAINT uniq_application DO UPDATE SET
		     message = EXCLUDED.message,
		     proposed_amount_btc = EXCLUDED.proposed_amount_btc,
		     status = 'pending',
		     updated_at = NOW()
		 RETURNING id`,
		orderID, freelancerID, message, nullableFloat(proposedAmount),
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert application: %w", err)
	}

	return &Application{ID: id, OrderID: orderID, FreelancerID: freelancerID, Message: message, ProposedAmount: proposedAmount, Status: "pending"}, nil
}

// ListForOrder — клиент смотрит все отклики на свой заказ.
func (s *Service) ListForOrder(orderID int64) ([]Application, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.order_id, a.freelancer_id, u.display_name, a.message, a.proposed_amount_btc, a.status
		FROM applications a
		JOIN users u ON u.id = a.freelancer_id
		WHERE a.order_id = $1
		ORDER BY a.created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.OrderID, &a.FreelancerID, &a.FreelancerName, &a.Message, &a.ProposedAmountBTC, &a.Status); err != nil {
			return nil, err
		}
		if a.ProposedAmountBTC.Valid {
			v := a.ProposedAmountBTC.Float64
			a.ProposedAmount = &v
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Withdraw — фрилансер отзывает свой отклик.
func (s *Service) Withdraw(applicationID, freelancerID int64) error {
	res, err := s.db.Exec(
		`UPDATE applications SET status = 'withdrawn' WHERE id = $1 AND freelancer_id = $2 AND status = 'pending'`,
		applicationID, freelancerID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("application not found, not yours, or not pending")
	}
	return nil
}

func nullableFloat(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}
