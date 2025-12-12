package client

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to a LogLevel.
func ParseLogLevel(s string) LogLevel {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Field represents a structured log field.
type Field struct {
	Key   string
	Value interface{}
}

// Helper functions for creating fields
func String(key, val string) Field       { return Field{Key: key, Value: val} }
func Int(key string, val int) Field      { return Field{Key: key, Value: val} }
func Int64(key string, val int64) Field  { return Field{Key: key, Value: val} }
func Float64(key string, val float64) Field { return Field{Key: key, Value: val} }
func Bool(key string, val bool) Field    { return Field{Key: key, Value: val} }
func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Value: val.String()}
}
func Error(key string, err error) Field {
	if err == nil {
		return Field{Key: key, Value: nil}
	}
	return Field{Key: key, Value: err.Error()}
}

// Logger is the interface for structured logging.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
}

// defaultLogger implements Logger using standard library log package.
type defaultLogger struct {
	logger    *log.Logger
	minLevel  LogLevel
	baseFields []Field
}

// NewLogger creates a new default logger with the specified level and output.
func NewLogger(level string, output io.Writer) Logger {
	if output == nil {
		output = os.Stdout
	}

	return &defaultLogger{
		logger:   log.New(output, "", 0),
		minLevel: ParseLogLevel(level),
		baseFields: []Field{},
	}
}

// NewDefaultLogger creates a logger with INFO level writing to stdout.
func NewDefaultLogger() Logger {
	return NewLogger("INFO", os.Stdout)
}

func (l *defaultLogger) Debug(msg string, fields ...Field) {
	if l.minLevel <= DEBUG {
		l.log(DEBUG, msg, fields...)
	}
}

func (l *defaultLogger) Info(msg string, fields ...Field) {
	if l.minLevel <= INFO {
		l.log(INFO, msg, fields...)
	}
}

func (l *defaultLogger) Warn(msg string, fields ...Field) {
	if l.minLevel <= WARN {
		l.log(WARN, msg, fields...)
	}
}

func (l *defaultLogger) Error(msg string, fields ...Field) {
	if l.minLevel <= ERROR {
		l.log(ERROR, msg, fields...)
	}
}

func (l *defaultLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, len(l.baseFields)+len(fields))
	copy(newFields, l.baseFields)
	copy(newFields[len(l.baseFields):], fields)

	return &defaultLogger{
		logger:     l.logger,
		minLevel:   l.minLevel,
		baseFields: newFields,
	}
}

func (l *defaultLogger) log(level LogLevel, msg string, fields ...Field) {
	// Combine base fields and log fields
	allFields := make([]Field, 0, len(l.baseFields)+len(fields)+3)
	allFields = append(allFields, Field{Key: "timestamp", Value: time.Now().Format(time.RFC3339Nano)})
	allFields = append(allFields, Field{Key: "level", Value: level.String()})
	allFields = append(allFields, Field{Key: "message", Value: msg})
	allFields = append(allFields, l.baseFields...)
	allFields = append(allFields, fields...)

	// Redact sensitive fields
	allFields = redactSensitiveFields(allFields)

	// Format as JSON
	logMap := make(map[string]interface{}, len(allFields))
	for _, field := range allFields {
		logMap[field.Key] = field.Value
	}

	jsonBytes, err := json.Marshal(logMap)
	if err != nil {
		l.logger.Printf(`{"level":"ERROR","message":"failed to marshal log","error":"%s"}`, err.Error())
		return
	}

	l.logger.Println(string(jsonBytes))
}

// redactSensitiveFields masks values for sensitive keys.
func redactSensitiveFields(fields []Field) []Field {
	sensitiveKeys := map[string]bool{
		"password":      true,
		"token":         true,
		"secret":        true,
		"authorization": true,
		"api_key":       true,
		"apikey":        true,
		"auth":          true,
	}

	result := make([]Field, len(fields))
	for i, field := range fields {
		key := strings.ToLower(field.Key)
		if sensitiveKeys[key] {
			result[i] = Field{Key: field.Key, Value: "[REDACTED]"}
		} else {
			result[i] = field
		}
	}

	return result
}

// noopLogger implements Logger but does nothing.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...Field) {}
func (n *noopLogger) Info(msg string, fields ...Field)  {}
func (n *noopLogger) Warn(msg string, fields ...Field)  {}
func (n *noopLogger) Error(msg string, fields ...Field) {}
func (n *noopLogger) WithFields(fields ...Field) Logger { return n }

// NewNoopLogger creates a logger that discards all output.
func NewNoopLogger() Logger {
	return &noopLogger{}
}

// requestIDKey is the context key for request IDs.
type contextKey string

const requestIDKey contextKey = "requestID"

// RequestIDField extracts request ID from context and returns it as a Field.
func RequestIDField(ctx interface{}) Field {
	// TODO: implement context value extraction when request ID tracking is added
	return Field{Key: "requestID", Value: "unknown"}
}
