package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

// parseTLSOptions extracts TLS parameters from connection string query parameters.
// Supports: ?tls=true, ?tlsCAFile=/path, ?tlsCert=/path, ?tlsKey=/path, ?tlsInsecureSkipVerify=true
func parseTLSOptions(connStr string) map[string]string {
	options := make(map[string]string)

	// Find query string after ?
	if idx := strings.Index(connStr, "?"); idx >= 0 {
		queryStr := connStr[idx+1:]
		pairs := strings.Split(queryStr, "&")

		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				options[key] = value
			}
		}
	}

	return options
}

// buildTLSConfig creates a TLS configuration from ClientOptions.
// TODO: TLS performance metrics (handshake duration, cipher suite) could be exposed for monitoring
func buildTLSConfig(opts ClientOptions, serverName string) (*tls.Config, error) {
	if opts.TLSConfig != nil {
		return opts.TLSConfig, nil
	}

	if !opts.TLSEnabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: opts.TLSInsecureSkipVerify,
	}

	// Load custom CA certificate if provided
	if opts.TLSCAFile != "" {
		caCert, err := os.ReadFile(opts.TLSCAFile)
		if err != nil {
			return nil, &ConnectionError{
				Code:    "TLS_CA_LOAD_FAILED",
				Type:    "CONNECTION_ERROR",
				Message: fmt.Sprintf("failed to load CA certificate from %s", opts.TLSCAFile),
				Details: map[string]interface{}{
					"caFile": opts.TLSCAFile,
				},
				Cause: err,
			}
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, &ConnectionError{
				Code:    "TLS_CA_INVALID",
				Type:    "CONNECTION_ERROR",
				Message: "failed to parse CA certificate",
				Details: map[string]interface{}{
					"caFile": opts.TLSCAFile,
				},
			}
		}

		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key if provided
	if opts.TLSCertFile != "" && opts.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
		if err != nil {
			return nil, &ConnectionError{
				Code:    "TLS_CLIENT_CERT_FAILED",
				Type:    "CONNECTION_ERROR",
				Message: "failed to load client certificate and key",
				Details: map[string]interface{}{
					"certFile": opts.TLSCertFile,
					"keyFile":  opts.TLSKeyFile,
				},
				Cause: err,
			}
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// parseTLSError provides clear error messages for common TLS failures.
func parseTLSError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "certificate has expired"):
		return &ConnectionError{
			Code:    "TLS_CERT_EXPIRED",
			Type:    "CONNECTION_ERROR",
			Message: "server certificate has expired",
			Cause:   err,
		}
	case strings.Contains(errStr, "certificate is not trusted"):
		return &ConnectionError{
			Code:    "TLS_CERT_UNTRUSTED",
			Type:    "CONNECTION_ERROR",
			Message: "server certificate is not trusted (try setting a custom CA or tlsInsecureSkipVerify for testing)",
			Cause:   err,
		}
	case strings.Contains(errStr, "doesn't match"):
		return &ConnectionError{
			Code:    "TLS_HOSTNAME_MISMATCH",
			Type:    "CONNECTION_ERROR",
			Message: "server certificate hostname doesn't match connection address",
			Cause:   err,
		}
	case strings.Contains(errStr, "unknown authority"):
		return &ConnectionError{
			Code:    "TLS_UNKNOWN_CA",
			Type:    "CONNECTION_ERROR",
			Message: "server certificate signed by unknown authority (try setting a custom CA)",
			Cause:   err,
		}
	default:
		return &ConnectionError{
			Code:    "TLS_HANDSHAKE_FAILED",
			Type:    "CONNECTION_ERROR",
			Message: "TLS handshake failed",
			Cause:   err,
		}
	}
}
