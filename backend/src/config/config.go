package config

import "os"

var (
	KafkaBroker = GetEnv("KAFKA_BROKER", "kafka:9092")
	Topic       = GetEnv("KAFKA_TOPIC", "chat-messages")
	DexIssuer   = GetEnv("DEX_ISSUER_URL", "http://dex:5556/dex")
	ClientID    = GetEnv("DEX_CLIENT_ID", "backend")
	Audience    = GetEnv("DEX_AUDIENCE", "backend")
	ApiPort     = GetEnv("API_PORT", "8080")
)

// GetEnv returns the value of the environment variable or a default value
func GetEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
