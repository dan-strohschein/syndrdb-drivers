//go:build milestone2
// +build milestone2

package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Statement represents a prepared statement with parameter placeholders.
// Follows the server's PREPARE/EXECUTE/DEALLOCATE protocol from parameterized_queries.md.
type Statement struct {
	name       string
	query      string
	paramCount int
	conn       ConnectionInterface
	closed     bool
	createdAt  time.Time
	mu         sync.Mutex
}

// QueryParams is a type-safe wrapper for query parameters.
type QueryParams []interface{}

// NewQueryParams creates a new QueryParams instance.
func NewQueryParams(values ...interface{}) QueryParams {
	return QueryParams(values)
}

// Execute runs the prepared statement with the provided parameters.
// Parameters are passed using the delimiter-based protocol: EXECUTE name\x05param1\x05param2
func (s *Statement) Execute(params ...interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("statement %s is already closed", s.name)
	}

	if len(params) != s.paramCount {
		return nil, ErrInvalidParameterCount(s.paramCount, len(params))
	}

	// Build EXECUTE command with delimiter-separated parameters
	command := buildExecuteCommand(s.name, params)

	// Send command and receive response
	ctx := context.Background() // TODO: Accept context parameter in next iteration
	if err := s.conn.SendCommand(ctx, command); err != nil {
		return nil, &QueryError{
			Code:    "E_EXECUTE_FAILED",
			Type:    "QueryError",
			Message: fmt.Sprintf("failed to execute statement %s", s.name),
			Details: map[string]interface{}{
				"statement_name": s.name,
				"param_count":    len(params),
			},
			Query:  s.query,
			Params: params,
			Cause:  err,
		}
	}

	result, err := s.conn.ReceiveResponse(ctx)
	if err != nil {
		return nil, &QueryError{
			Code:    "E_EXECUTE_RESPONSE_FAILED",
			Type:    "QueryError",
			Message: fmt.Sprintf("failed to receive response for statement %s", s.name),
			Details: map[string]interface{}{
				"statement_name": s.name,
			},
			Query:  s.query,
			Params: params,
			Cause:  err,
		}
	}

	return result, nil
}

// Close deallocates the prepared statement on the server.
// Sends DEALLOCATE command per server protocol.
func (s *Statement) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil // Already closed, no-op
	}

	command := fmt.Sprintf("DEALLOCATE %s", s.name)
	ctx := context.Background()

	if err := s.conn.SendCommand(ctx, command); err != nil {
		return &StatementError{
			QueryError: QueryError{
				Code:    "E_DEALLOCATE_FAILED",
				Type:    "StatementError",
				Message: fmt.Sprintf("failed to deallocate statement %s", s.name),
				Details: map[string]interface{}{
					"statement_name": s.name,
				},
				Cause: err,
			},
			StatementName: s.name,
		}
	}

	s.closed = true
	return nil
}

// Name returns the statement name.
func (s *Statement) Name() string {
	return s.name
}

// Query returns the original query text.
func (s *Statement) Query() string {
	return s.query
}

// ParamCount returns the number of parameters expected.
func (s *Statement) ParamCount() int {
	return s.paramCount
}

// escapeParameterValue escapes special control characters in parameter values.
// Per server protocol: \x04 (EOT) -> \x04\x04, \x05 (ENQ) -> \x05\x05
func escapeParameterValue(value string) string {
	value = strings.ReplaceAll(value, "\x04", "\x04\x04") // Escape EOT
	value = strings.ReplaceAll(value, "\x05", "\x05\x05") // Escape ENQ
	return value
}

// buildExecuteCommand formats an EXECUTE command with delimiter-separated parameters.
// Format: EXECUTE statement_name\x05param1\x05param2\x05param3
func buildExecuteCommand(stmtName string, params []interface{}) string {
	var sb strings.Builder
	sb.WriteString("EXECUTE ")
	sb.WriteString(stmtName)

	for _, param := range params {
		sb.WriteString("\x05") // ENQ delimiter
		paramStr := convertToString(param)
		sb.WriteString(escapeParameterValue(paramStr))
	}

	return sb.String()
}

// convertToString converts a parameter value to its string representation.
func convertToString(value interface{}) string {
	if value == nil {
		return "" // NULL value represented as empty string
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%f", v)
	case float32:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// validateStatementName checks if the statement name is valid per server requirements.
// Statement names must be alphanumeric with underscores only, no special characters.
func validateStatementName(name string) error {
	if name == "" {
		return &StatementError{
			QueryError: QueryError{
				Code:    "E_INVALID_STMT_NAME",
				Type:    "StatementError",
				Message: "statement name cannot be empty",
				Details: map[string]interface{}{
					"allowed_pattern": "alphanumeric characters and underscores only",
				},
			},
			StatementName: name,
		}
	}

	// Check for valid characters: alphanumeric and underscores only
	validName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validName.MatchString(name) {
		return &StatementError{
			QueryError: QueryError{
				Code:    "E_INVALID_STMT_NAME",
				Type:    "StatementError",
				Message: fmt.Sprintf("invalid statement name: %s", name),
				Details: map[string]interface{}{
					"statement_name":  name,
					"allowed_pattern": "alphanumeric characters and underscores only",
					"rejected_chars":  "special characters like -, @, #, . are not allowed",
				},
			},
			StatementName: name,
		}
	}

	return nil
}

// countPlaceholders scans a query for $N placeholders and returns the maximum index.
// Returns the parameter count expected for the query.
func countPlaceholders(query string) int {
	placeholderRegex := regexp.MustCompile(`\$(\d+)`)
	matches := placeholderRegex.FindAllStringSubmatch(query, -1)

	maxIndex := 0
	for _, match := range matches {
		if len(match) > 1 {
			var index int
			fmt.Sscanf(match[1], "%d", &index)
			if index > maxIndex {
				maxIndex = index
			}
		}
	}

	return maxIndex
}
