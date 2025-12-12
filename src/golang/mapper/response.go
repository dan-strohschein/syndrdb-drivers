package mapper

import (
	"fmt"
	"strconv"
	"time"
)

// ResponseMapper handles type coercion for database responses.
// Matches the Node.js implementation in ResponseMapper.ts.
type ResponseMapper struct{}

// NewResponseMapper creates a new response mapper.
func NewResponseMapper() *ResponseMapper {
	return &ResponseMapper{}
}

// MapResponse maps a raw response to the expected type structure.
func (m *ResponseMapper) MapResponse(response interface{}, targetType string) (interface{}, error) {
	if response == nil {
		return nil, nil
	}

	switch targetType {
	case "string":
		return m.ToString(response), nil
	case "int":
		return m.ToInt(response)
	case "float":
		return m.ToFloat(response)
	case "boolean":
		return m.ToBool(response)
	case "datetime":
		return m.ToDateTime(response)
	case "object", "json":
		return response, nil
	default:
		return response, nil
	}
}

// ToString converts any value to a string.
func (m *ResponseMapper) ToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToInt converts a value to an integer.
func (m *ResponseMapper) ToInt(value interface{}) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("cannot convert nil to int")
	}

	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to int: %w", v, err)
		}
		return i, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ToFloat converts a value to a float.
func (m *ResponseMapper) ToFloat(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("cannot convert nil to float")
	}

	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to float: %w", v, err)
		}
		return f, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float", value)
	}
}

// ToBool converts a value to a boolean.
func (m *ResponseMapper) ToBool(value interface{}) (bool, error) {
	if value == nil {
		return false, nil
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case int, int32, int64:
		return v != 0, nil
	case float32, float64:
		return v != 0, nil
	case string:
		// Handle common boolean strings
		switch v {
		case "true", "1", "yes", "y", "on":
			return true, nil
		case "false", "0", "no", "n", "off", "":
			return false, nil
		default:
			return false, fmt.Errorf("cannot convert '%s' to boolean", v)
		}
	default:
		return false, fmt.Errorf("cannot convert %T to boolean", value)
	}
}

// ToDateTime converts a value to a time.Time.
func (m *ResponseMapper) ToDateTime(value interface{}) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("cannot convert nil to datetime")
	}

	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		// Try multiple datetime formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}

		return time.Time{}, fmt.Errorf("cannot parse '%s' as datetime", v)
	case int, int32, int64:
		// Assume Unix timestamp
		var timestamp int64
		switch tv := v.(type) {
		case int:
			timestamp = int64(tv)
		case int32:
			timestamp = int64(tv)
		case int64:
			timestamp = tv
		}
		return time.Unix(timestamp, 0), nil
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to datetime", value)
	}
}

// MapArray maps an array of responses to the expected type.
func (m *ResponseMapper) MapArray(responses []interface{}, targetType string) ([]interface{}, error) {
	if responses == nil {
		return nil, nil
	}

	result := make([]interface{}, len(responses))
	for i, response := range responses {
		mapped, err := m.MapResponse(response, targetType)
		if err != nil {
			return nil, fmt.Errorf("error mapping array element %d: %w", i, err)
		}
		result[i] = mapped
	}

	return result, nil
}

// MapObject maps object fields to their expected types.
func (m *ResponseMapper) MapObject(obj map[string]interface{}, fieldTypes map[string]string) (map[string]interface{}, error) {
	if obj == nil {
		return nil, nil
	}

	result := make(map[string]interface{})

	for key, value := range obj {
		if targetType, hasType := fieldTypes[key]; hasType {
			mapped, err := m.MapResponse(value, targetType)
			if err != nil {
				return nil, fmt.Errorf("error mapping field '%s': %w", key, err)
			}
			result[key] = mapped
		} else {
			// No type specified, keep original value
			result[key] = value
		}
	}

	return result, nil
}
