package rest

import (
	"reflect"
	"testing"
)

func TestTokenizePath(t *testing.T) {

	cases := []struct {
		name string
		path string
		exp  []string
	}{
		{
			name: "Normal path - No trailing slash",
			path: "/apps/v1/namespaces/default/deployments/my-deployment",
			exp:  []string{"apps", "v1", "namespaces", "default", "deployments", "my-deployment"},
		},
		{
			name: "Normal path - With trailing slash",
			path: "/apps/v1/namespaces/default/deployments/my-deployment/",
			exp:  []string{"apps", "v1", "namespaces", "default", "deployments", "my-deployment"},
		},
		{
			name: "Root path only",
			path: "/",
			exp:  nil,
		},
		{
			name: "Empty path",
			path: "",
			exp:  []string{""},
		},
		{
			name: "Multiple leading slashes",
			path: "///user//info",
			exp:  []string{"user", "", "info"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tokenizePath(tc.path)
			if !reflect.DeepEqual(actual, tc.exp) {
				t.Errorf("case %s no pass\ninput: %q\nexpected: %#v\ngot: %#v\n", tc.name, tc.path, tc.exp, actual)
			}
		})
	}

}
