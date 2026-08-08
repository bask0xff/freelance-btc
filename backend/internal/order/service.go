package order

import (
	"database/sql"
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

// CreateOrder создаёт заказ и сразу выдаёт под него уникальный BTC-адрес
// (аналог getnewaddress("label", "bech32") из вашего скрипта, но с order_id как label).
func (s *Service) CreateOrder(clientID int64, title, description string, amountBTC float64) (*Order, error) {
	res, err := s.db.Exec(
		`INSERT INTO orders (client_id, title, description, amount_btc, status) VALUES (?, ?, ?, ?, 'created')`,
		clientID, title, description, amountBTC,
	)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	label := fmt.Sprintf("order_%d", orderID)
	address, err := s.rpc.GetNewAddress(label, "bech32")
	if err != nil {
		return nil, fmt.Errorf("getnewaddress: %w", err)
	}

	if _, err := s.db.Exec(
		`INSERT INTO btc_addresses (order_id, address, address_type, label) VALUES (?, ?, 'bech32', ?)`,
		orderID, address, label,
	); err != nil {
		return nil, fmt.Errorf("insert address: %w", err)
	}

	if _, err := s.db.Exec(`UPDATE orders SET status = 'awaiting_payment' WHERE id = ?`, orderID); err != nil {
		return nil, err
	}

	return &Order{ID: orderID, ClientID: clientID, Title: title, AmountBTC: amountBTC, Status: "awaiting_payment", Address: address}, nil
}

// ReleaseEscrow — вызывается, когда клиент подтвердил сдачу работы (или разрешён спор в пользу фрилансера).
// Отправляет средства с общего хот-кошелька на payout_address фрилансера.
// ВАЖНО: это кастодиальная модель (см. обсуждение) — деньги физически лежат в общем кошельке ноды,
// привязка к заказу — только на уровне вашей БД (btc_addresses/btc_transactions).
func (s *Service) ReleaseEscrow(orderID int64) (string, error) {
	var amountBTC float64
	var payoutAddress string
	var status string
	err := s.db.QueryRow(`
		SELECT o.amount_btc, u.payout_address, o.status
		FROM orders o
		JOIN users u ON u.id = o.freelancer_id
		WHERE o.id = ?
	`, orderID).Scan(&amountBTC, &payoutAddress, &status)
	if err != nil {
		return "", fmt.Errorf("lookup order: %w", err)
	}
	if status != "delivered" {
		return "", fmt.Errorf("order status %q is not eligible for release (need 'delivered')", status)
	}
	if payoutAddress == "" {
		return "", fmt.Errorf("freelancer has no payout_address set")
	}

	txid, err := s.rpc.SendWithUnlock(payoutAddress, amountBTC, fmt.Sprintf("escrow release order_%d", orderID))
	if err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	if _, err := s.db.Exec(`
		INSERT INTO btc_transactions (order_id, address, txid, amount_btc, satoshi, confirmations, direction, status)
		VALUES (?, ?, ?, ?, ?, 0, 'out', 'pending')
	`, orderID, payoutAddress, txid, amountBTC, int64(amountBTC*1e8)); err != nil {
		return "", fmt.Errorf("log outgoing tx: %w", err)
	}

	if _, err := s.db.Exec(`UPDATE orders SET status = 'completed' WHERE id = ?`, orderID); err != nil {
		return "", err
	}

	return txid, nil
}
