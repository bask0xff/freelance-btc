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

	// MySQL
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string

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

		DBHost: getEnv("DB_HOST", "mysql"),
		DBPort: getEnvInt("DB_PORT", 3306),
		DBUser: getEnv("DB_USER", "root"),
		DBPass: getEnv("DB_PASSWORD", "password"),
		DBName: getEnv("DB_NAME", "freelance_btc"),

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
