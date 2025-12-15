package client

import (
	"crypto/tls"
	"time"
)

// ClientOptions configures the SyndrDB client behavior.
type ClientOptions struct {
	// DefaultTimeoutMs is the default timeout in milliseconds for operations.
	// Default: 10000 (10 seconds)
	DefaultTimeoutMs int

	// DebugMode enables verbose error serialization with full cause chains.
	// When true, errors include complete stack of wrapped errors.
	// When false, errors are flattened to single message.
	// Default: false
	DebugMode bool

	// MaxRetries is the maximum number of connection retry attempts.
	// Uses exponential backoff: 100ms, 200ms, 400ms, etc.
	// Default: 3
	MaxRetries int

	// PoolMinSize is the minimum number of idle connections to maintain.
	// Default: 1 (single connection mode)
	PoolMinSize int

	// PoolMaxSize is the maximum number of open connections.
	// Default: 1 (single connection mode)
	PoolMaxSize int

	// PoolIdleTimeout is the duration after which idle connections are closed.
	// Default: 30s
	PoolIdleTimeout time.Duration

	// HealthCheckInterval is how often to ping idle connections.
	// Default: 30s
	HealthCheckInterval time.Duration

	// MaxReconnectAttempts is the maximum number of automatic reconnection attempts.
	// Default: 10
	MaxReconnectAttempts int

	// TLSConfig provides custom TLS configuration.
	// If nil, TLS is disabled unless TLSEnabled is true.
	TLSConfig *tls.Config

	// TLSEnabled enables TLS with default configuration.
	// Default: false
	TLSEnabled bool

	// TLSInsecureSkipVerify skips certificate validation (for development only).
	// Default: false
	TLSInsecureSkipVerify bool

	// TLSCAFile is the path to a custom CA certificate file.
	TLSCAFile string

	// TLSCertFile is the path to the client certificate file.
	TLSCertFile string

	// TLSKeyFile is the path to the client private key file.
	TLSKeyFile string

	// Logger is the logger implementation to use.
	// If nil, a default logger is used.
	Logger Logger

	// LogLevel sets the minimum log level (DEBUG, INFO, WARN, ERROR).
	// Default: "INFO"
	LogLevel string

	// OnConnected is called when a connection is successfully established.
	OnConnected func(StateTransition)

	// OnDisconnected is called when a connection is lost.
	OnDisconnected func(StateTransition)

	// OnReconnecting is called when automatic reconnection is attempted.
	OnReconnecting func(StateTransition)

	// PreparedStatementCacheSize is the maximum number of prepared statements to cache.
	// Default: 100
	PreparedStatementCacheSize int

	// TransactionTimeout is the maximum duration a transaction can remain active.
	// Transactions exceeding this timeout are automatically rolled back.
	// Default: 5 minutes
	TransactionTimeout time.Duration

	// SchemaCacheTTL is the duration for which schema information is cached.
	// After this period, schema is refreshed from the server on next validation.
	// Default: 5 minutes
	SchemaCacheTTL time.Duration

	// PreloadSchema enables eager schema loading during connection initialization.
	// When true, schema is fetched immediately after connecting.
	// Default: false
	PreloadSchema bool
}

// DefaultOptions returns ClientOptions with default values.
func DefaultOptions() ClientOptions {
	return ClientOptions{
		DefaultTimeoutMs:           10000,
		DebugMode:                  false,
		MaxRetries:                 3,
		PoolMinSize:                1,
		PoolMaxSize:                1,
		PoolIdleTimeout:            30 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		MaxReconnectAttempts:       10,
		TLSEnabled:                 false,
		TLSInsecureSkipVerify:      false,
		LogLevel:                   "INFO",
		PreparedStatementCacheSize: 100,
		TransactionTimeout:         5 * time.Minute,
		SchemaCacheTTL:             5 * time.Minute,
		PreloadSchema:              false,
	}
}
