// Package bitcoin — обёртка над JSON-RPC bitcoind.
// Прямой аналог AuthServiceProxy из вашего btc_check.py, но типизированный.
package bitcoin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	url      string
	user     string
	pass     string
	http     *http.Client
	reqID    int
	mu       sync.Mutex
	Passphrase string // паспарола кошелька — используется ТОЛЬКО для временной разблокировки
}

func NewClient(host string, port int, user, pass, walletPassphrase string) *Client {
	return &Client{
		url:        fmt.Sprintf("http://%s:%d", host, port),
		user:       user,
		pass:       pass,
		http:       &http.Client{Timeout: 60 * time.Second},
		Passphrase: walletPassphrase,
	}
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
	ID     int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("bitcoind rpc error %d: %s", e.Code, e.Message)
}

// call — низкоуровневый вызов, аналог rpc_connection.<method>(...) в питоне.
func (c *Client) call(method string, params []interface{}, result interface{}) error {
	c.mu.Lock()
	c.reqID++
	id := c.reqID
	c.mu.Unlock()

	body, err := json.Marshal(rpcRequest{JSONRPC: "1.0", ID: id, Method: method, Params: params})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rr rpcResponse
	if err := json.Unmarshal(raw, &rr); err != nil {
		return fmt.Errorf("bad rpc response: %s", string(raw))
	}
	if rr.Error != nil {
		return rr.Error
	}
	if result != nil {
		return json.Unmarshal(rr.Result, result)
	}
	return nil
}

// ---- Высокоуровневые методы, аналогичные вызовам в вашем скрипте ----

func (c *Client) GetBestBlockHash() (string, error) {
	var out string
	err := c.call("getbestblockhash", nil, &out)
	return out, err
}

func (c *Client) GetBlockCount() (int64, error) {
	var out int64
	err := c.call("getblockcount", nil, &out)
	return out, err
}

func (c *Client) GetBalance() (float64, error) {
	var out float64
	err := c.call("getbalance", nil, &out)
	return out, err
}

// GetNewAddress — аналог rpc_connection.getnewaddress("label", "bech32").
// Здесь label — ID заказа (строкой), чтобы позже фильтровать транзакции по заказу.
func (c *Client) GetNewAddress(label, addressType string) (string, error) {
	var out string
	err := c.call("getnewaddress", []interface{}{label, addressType}, &out)
	return out, err
}

// KeypoolRefill — аналог rpc_connection.keypoolrefill()
func (c *Client) KeypoolRefill(size ...int) error {
	params := []interface{}{}
	if len(size) > 0 {
		params = append(params, size[0])
	}
	return c.call("keypoolrefill", params, nil)
}

type WalletInfo struct {
	KeypoolSize int64 `json:"keypoolsize"`
	Unlocked    bool  `json:"unlocked_until"`
}

func (c *Client) GetWalletInfo() (map[string]interface{}, error) {
	var out map[string]interface{}
	err := c.call("getwalletinfo", nil, &out)
	return out, err
}

// Transaction — одна запись из listtransactions, поля как в вашем скрипте.
type Transaction struct {
	Address       string  `json:"address"`
	Category      string  `json:"category"` // "send" | "receive"
	Amount        float64 `json:"amount"`
	Confirmations int64   `json:"confirmations"`
	TxID          string  `json:"txid"`
	Time          int64   `json:"time"`
	Label         string  `json:"label"`
}

// ListTransactions — аналог rpc_connection.listtransactions("*", 1000)
func (c *Client) ListTransactions(count int) ([]Transaction, error) {
	var out []Transaction
	err := c.call("listtransactions", []interface{}{"*", count}, &out)
	return out, err
}

// ListTransactionsByLabel — фильтр по лейблу (=order_id), bitcoind поддерживает label напрямую в getnewaddress,
// но listtransactions по лейблу нужно фильтровать на своей стороне (RPC label-фильтра нет в старых версиях).
func (c *Client) ListTransactionsByLabel(label string, count int) ([]Transaction, error) {
	all, err := c.ListTransactions(count)
	if err != nil {
		return nil, err
	}
	var filtered []Transaction
	for _, t := range all {
		if t.Label == label {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// UnlockWallet — аналог rpc_connection.walletpassphrase(pass, timeoutSec).
// ВАЖНО: использовать только непосредственно перед sendtoaddress, затем сразу LockWallet.
func (c *Client) UnlockWallet(timeoutSec int) error {
	return c.call("walletpassphrase", []interface{}{c.Passphrase, timeoutSec}, nil)
}

func (c *Client) LockWallet() error {
	return c.call("walletlock", nil, nil)
}

// SendToAddress — используется для release эскроу фрилансеру.
func (c *Client) SendToAddress(address string, amountBTC float64, comment string) (string, error) {
	var txid string
	err := c.call("sendtoaddress", []interface{}{address, amountBTC, comment}, &txid)
	return txid, err
}

// SendWithUnlock — безопасная обёртка: разблокировать -> отправить -> заблокировать, что бы ни случилось.
func (c *Client) SendWithUnlock(address string, amountBTC float64, comment string) (string, error) {
	if err := c.UnlockWallet(30); err != nil {
		return "", fmt.Errorf("unlock wallet: %w", err)
	}
	defer c.LockWallet() //nolint:errcheck

	txid, err := c.SendToAddress(address, amountBTC, comment)
	if err != nil {
		return "", fmt.Errorf("send: %w", err)
	}
	return txid, nil
}
