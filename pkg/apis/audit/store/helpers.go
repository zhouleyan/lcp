package store

import (
	"strconv"
	"time"
)

func filterStr(filters map[string]any, key string) *string {
	if v, ok := filters[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

func filterInt32(filters map[string]any, key string) *int32 {
	if v, ok := filters[key]; ok {
		switch val := v.(type) {
		case int32:
			return &val
		case int64:
			i := int32(val)
			return &i
		case string:
			if i, err := strconv.ParseInt(val, 10, 32); err == nil {
				i32 := int32(i)
				return &i32
			}
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
			if val == "true" {
				b := true
				return &b
			} else if val == "false" {
				b := false
				return &b
			}
		}
	}
	return nil
}

func filterTime(filters map[string]any, key string) *time.Time {
	if v, ok := filters[key]; ok {
		if s, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return &t
			}
		}
	}
	return nil
}
