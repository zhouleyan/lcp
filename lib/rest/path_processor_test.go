package rest

import (
	"reflect"
	"testing"
)

func TestExtractParameters(t *testing.T) {

	cases := []struct {
		name      string
		routePath string
		urlPath   string
		expected  map[string]string
	}{
		{
			name:      "Single path param",
			routePath: "/users/{id}",
			urlPath:   "/users/123",
			expected:  map[string]string{"id": "123"},
		},
		{
			name:      "Multiple path parameters",
			routePath: "/namespaces/{namespaceId}/users/{userId}",
			urlPath:   "/namespaces/ns1/users/123",
			expected:  map[string]string{"namespaceId": "ns1", "userId": "123"},
		},
		{
			name:      "Multiple parts single path parameters",
			routePath: "/api/v1/users/{userId}",
			urlPath:   "/api/v1/users/999/profile",
			expected:  map[string]string{"userId": "999"},
		},
		{
			name:      "Empty path parameters",
			routePath: "/users/{userId}",
			urlPath:   "/users/",
			expected:  map[string]string{"userId": ""},
		},
		{
			name:      "No path parameters",
			routePath: "/api/v1/users",
			urlPath:   "/api/v1/users",
			expected:  map[string]string{},
		},
		{
			name:      "numeric regex parameter",
			routePath: "/users/{id:[0-9]+}",
			urlPath:   "/users/12345",
			expected:  map[string]string{"id": "12345"},
		},
		{
			name:      "custom verb",
			routePath: "/users/{id}:get",
			urlPath:   "/users/123:get",
			expected:  map[string]string{"id": "123"},
		},
	}

	p := defaultPathProcessor{}
	for _, c := range cases {
		route := &Route{
			Path:          c.routePath,
			pathParts:     tokenizePath(c.routePath),
			hasCustomVerb: hasCustomVerb(c.routePath),
		}
		t.Run(c.name, func(t *testing.T) {
			result := p.ExtractParameters(route, nil, c.urlPath)
			if !reflect.DeepEqual(result, c.expected) {
				t.Errorf("ExtractParameters() = %v, expected %v", result, c.expected)
			}
		})
	}
}
