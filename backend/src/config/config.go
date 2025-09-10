package config

import "os"

var (
	KafkaBroker = GetEnv("KAFKA_BROKER", "kafka:9092")
	Topic       = GetEnv("KAFKA_TOPIC", "chat-messages")
	DLQTopic    = GetEnv("KAFKA_DLQ_TOPIC", "chat-messages-dlq")
	DexIssuer   = GetEnv("DEX_ISSUER_URL", "http://dex:5556/dex")
	// DexIssuerDialOverride allows dialing a different host:port while preserving the issuer Host header.
	// Example: ingress-nginx-controller.ingress-nginx.svc.cluster.local:80
	DexIssuerDialOverride = GetEnv("DEX_ISSUER_DIAL_ADDRESS", "")
	// Internal dial target used automatically if issuer host resolves only to loopback and no explicit override is set.
	DexIssuerInternalDial = GetEnv("DEX_ISSUER_INTERNAL_DIAL", "ingress-nginx-controller.ingress-nginx.svc.cluster.local:80")
	// Internal fallback issuer (cluster-internal service) used if primary issuer discovery fails.
	InternalDexIssuer = GetEnv("DEX_INTERNAL_ISSUER_URL", "http://dex:5556/dex")
	// Enable / disable automatic fallback to InternalDexIssuer when primary issuer fails.
	DexOIDCFallbackEnabled = GetEnv("DEX_OIDC_FALLBACK_ENABLED", "true")
	// Max attempts for primary issuer before considering failure.
	DexOIDCMaxAttempts = GetEnv("DEX_OIDC_MAX_ATTEMPTS", "8")
	// Enable extra debug logging for OIDC discovery.
	DexOIDCDebug = GetEnv("DEX_OIDC_DEBUG", "false")
	// Optional path to a PEM encoded CA certificate bundle used to trust the Dex issuer (for custom/self-signed CA).
	DexCACertFile = GetEnv("DEX_CA_CERT_FILE", "")
	ClientID      = GetEnv("DEX_CLIENT_ID", "backend")
	Audience      = GetEnv("DEX_AUDIENCE", "backend")
	ApiPort       = GetEnv("API_PORT", "8080")
	MongoURI      = GetEnv("MONGO_URI", "mongodb://mongodb:27017")
)

// GetEnv returns the value of the environment variable or a default value
func GetEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
