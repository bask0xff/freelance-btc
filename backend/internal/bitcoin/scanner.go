package bitcoin

import (
	"database/sql"
	"log"
	"time"
)

// Scanner — периодически опрашивает bitcoind (как ваш while True + time.sleep(10*60)),
// но вместо CSV/глобальной таблицы обновляет статус конкретных заказов по их BTC-адресам.
type Scanner struct {
	rpc      *Client
	db       *sql.DB
	interval time.Duration
	// MinConfirmations — сколько подтверждений нужно, чтобы считать заказ оплаченным.
	MinConfirmations int64
}

func NewScanner(rpc *Client, db *sql.DB, interval time.Duration, minConf int64) *Scanner {
	return &Scanner{rpc: rpc, db: db, interval: interval, MinConfirmations: minConf}
}

func (s *Scanner) Run() {
	for {
		if err := s.scanOnce(); err != nil {
			log.Println("scanner error:", err)
		}
		time.Sleep(s.interval)
	}
}

func (s *Scanner) scanOnce() error {
	txs, err := s.rpc.ListTransactions(1000) // как в вашем скрипте: "*", 1000
	if err != nil {
		return err
	}

	for _, tx := range txs {
		if tx.Category != "receive" {
			continue // нас интересуют только входящие платежи на адреса заказов
		}

		status := statusFor(tx.Confirmations, s.MinConfirmations)

		// Дедуп по уникальному ограничению uniq_tx (address, txid, direction) — см. миграцию 000001.
		_, err := s.db.Exec(`
			INSERT INTO btc_transactions (order_id, address, txid, amount_btc, satoshi, confirmations, direction, status, created_at, updated_at)
			SELECT o.id, $1, $2, $3, $4, $5, 'in', $6, NOW(), NOW()
			FROM orders o
			JOIN btc_addresses a ON a.order_id = o.id
			WHERE a.address = $1
			ON CONFLICT ON CONSTRAINT uniq_tx DO UPDATE SET
				confirmations = EXCLUDED.confirmations,
				status = EXCLUDED.status,
				updated_at = NOW()
		`,
			tx.Address, tx.TxID, tx.Amount, int64(tx.Amount*1e8), tx.Confirmations, status,
		)
		if err != nil {
			log.Println("insert tx error:", err)
			continue
		}

		// Если платёж подтверждён достаточным числом confirmations — переводим заказ в статус funded.
		if tx.Confirmations >= s.MinConfirmations {
			_, err := s.db.Exec(`
				UPDATE orders
				SET status = 'funded', updated_at = NOW()
				FROM btc_addresses a
				WHERE a.order_id = orders.id AND a.address = $1 AND orders.status = 'awaiting_payment'
			`, tx.Address)
			if err != nil {
				log.Println("update order status error:", err)
			}
		}
	}
	return nil
}

func statusFor(confirmations, minConf int64) string {
	if confirmations >= minConf {
		return "confirmed"
	}
	return "pending"
}
