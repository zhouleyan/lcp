package store

import "strconv"

func filterStr(filters map[string]any, key string) *string {
	if v, ok := filters[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

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

func filterBool(filters map[string]any, key string) *bool {
	if v, ok := filters[key]; ok {
		switch val := v.(type) {
		case bool:
			return &val
		case string:
			if b, err := strconv.ParseBool(val); err == nil {
				return &b
			}
		}
	}
	return nil
}
