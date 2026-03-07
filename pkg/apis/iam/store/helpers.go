package store

import "strconv"

// toNullString converts an empty string to nil, otherwise returns a pointer to the string.
func toNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// filterStr extracts a string filter value from a query filter map.
func filterStr(filters map[string]any, key string) *string {
	if v, ok := filters[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

// filterInt64 extracts an int64 filter value from a query filter map.
// Accepts both int64 and string (parsed) values.
func filterInt64(filters map[string]any, key string) *int64 {
	if v, ok := filters[key]; ok {
		switch val := v.(type) {
		case int64:
			return &val
		case string:
			if i, err := strconv.ParseInt(val, 10, 64); err == nil {
				return &i
			}
		}
	}
	return nil
}

// toNullInt64 converts a zero int64 to nil, otherwise returns a pointer to the value.
func toNullInt64(n int64) *int64 {
	if n == 0 {
		return nil
	}
	return &n
}

// toNullInt32 converts a zero int32 to nil, otherwise returns a pointer to the value.
func toNullInt32(n int32) *int32 {
	if n == 0 {
		return nil
	}
	return &n
}
