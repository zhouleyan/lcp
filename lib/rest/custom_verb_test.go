package rest

import (
	"fmt"
	"testing"
)

func TestIsMatchCustomVerb(t *testing.T) {
	cases := []struct {
		routeToken string
		pathToken  string
		expected   bool
	}{
		{
			"user:show", "user:show", true,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_%s", c.routeToken, c.pathToken), func(t *testing.T) {
			result := isMatchCustomVerb(c.routeToken, c.pathToken)
			if result != c.expected {
				t.Errorf("isMatchCustomVerb(%q, %q) = %v; want %v", c.routeToken, c.pathToken, result, c.expected)
			}
		})
	}
}

func TestRemoveCustomVerb(t *testing.T) {
	cases := []struct {
		routeToken string
		expected   string
	}{
		{
			"user", "user",
		},
		{
			":show", "",
		},
		{
			"user:show", "user",
		},
	}

	for _, c := range cases {
		t.Run(c.routeToken, func(t *testing.T) {
			result := removeCustomVerb(c.routeToken)
			if result != c.expected {
				t.Errorf("removeCustomVerb(%q) = %q; want %q", c.routeToken, result, c.expected)
			}
		})
	}
}
