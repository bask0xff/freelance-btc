package config

import (
	"os"
	"strconv"
)

type Config struct {
	// bitcoind RPC
	BtcRPCHost       string
	BtcRPCPort       int
	BtcRPCUser       string
	BtcRPCPassword   string
	WalletPassphrase string
	MinConfirmations int64

	// PostgreSQL
	DBHost    string
	DBPort    int
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string

	HTTPPort string

	JWTSecret string
	JWTTTLMin int
}

func Load() Config {
	return Config{
		BtcRPCHost:       getEnv("BTC_RPC_HOST", "127.0.0.1"),
		BtcRPCPort:       getEnvInt("BTC_RPC_PORT", 8332),
		BtcRPCUser:       getEnv("BTC_RPC_USER", "btcuser"),
		BtcRPCPassword:   getEnv("BTC_RPC_PASSWORD", ""),
		WalletPassphrase: getEnv("BTC_WALLET_PASSPHRASE", ""),
		MinConfirmations: int64(getEnvInt("BTC_MIN_CONFIRMATIONS", 2)),

		DBHost:    getEnv("DB_HOST", "postgres"),
		DBPort:    getEnvInt("DB_PORT", 5432),
		DBUser:    getEnv("DB_USER", "postgres"),
		DBPass:    getEnv("DB_PASSWORD", "password"),
		DBName:    getEnv("DB_NAME", "freelance_btc"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"), // локально без TLS; на проде поставьте "require"

		HTTPPort: getEnv("HTTP_PORT", "8080"),

		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTTTLMin: getEnvInt("JWT_TTL_MINUTES", 60*24*7), // неделя по умолчанию
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
