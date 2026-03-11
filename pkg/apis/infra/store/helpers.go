package store

import (
	"encoding/json"
	"strconv"
)

func toNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

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

func labelsToJSON(labels map[string]string) json.RawMessage {
	if labels == nil {
		return json.RawMessage("{}")
	}
	b, err := json.Marshal(labels)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}
