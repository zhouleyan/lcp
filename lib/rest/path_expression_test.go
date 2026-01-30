package rest

import (
	"reflect"
	"testing"
)

func TestTemplateToRegExp(t *testing.T) {

	cases := []struct {
		name          string
		template      string
		expExpression string
		expLiteral    int
		expVarNames   []string
		expVarCount   int
		expTokens     []string
	}{
		{
			name:          "Simple path with literals",
			template:      "/users/profile",
			expExpression: "^/users/profile(/.*)?$",
			expLiteral:    12,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"users", "profile"},
		},
		{
			name:          "Path with single variable",
			template:      "/users/{id}",
			expExpression: "^/users/([^/]+?)(/.*)?$",
			expLiteral:    5,
			expVarNames:   []string{"id"},
			expVarCount:   1,
			expTokens:     []string{"users", "{id}"},
		},
		{
			name:          "Path with multiple variables",
			template:      "/users/{userId}/posts/{postId}",
			expExpression: "^/users/([^/]+?)/posts/([^/]+?)(/.*)?$",
			expLiteral:    10,
			expVarNames:   []string{"userId", "postId"},
			expVarCount:   2,
			expTokens:     []string{"users", "{userId}", "posts", "{postId}"},
		},
		{
			name:          "Path with variable and regex pattern",
			template:      "/users/{id:[0-9]+}",
			expExpression: "^/users/([0-9]+)(/.*)?$",
			expLiteral:    5,
			expVarNames:   []string{"id"},
			expVarCount:   1,
			expTokens:     []string{"users", "{id:[0-9]+}"},
		},
		{
			name:          "Path with wildcard variable",
			template:      "/files/{path:*}",
			expExpression: "^/files/(.*)(/.*)?$",
			expLiteral:    5,
			expVarNames:   []string{"path"},
			expVarCount:   1,
			expTokens:     []string{"files", "{path:*}"},
		},
		//{
		//	name:          "Path with special characters requiring URI encode",
		//	template:      "/search/hello world",
		//	expExpression: "^/search/hello%20world(/.*)?$",
		//	expLiteral:    17,
		//	expVarNames:   []string{},
		//	expVarCount:   0,
		//	expTokens:     []string{"search", "hello world"},
		//},
		//{
		//	name:          "Path with multiple special characters",
		//	template:      "api/v1/user@example.com",
		//	expExpression: "^api/v1/user%40example.com(/.*)?$",
		//	expLiteral:    21,
		//	expVarNames:   []string{},
		//	expVarCount:   0,
		//	expTokens:     []string{"api", "v1", "user@example.com"},
		//},
		{
			name:          "Path with slash in literal (should be encoded)",
			template:      "/path/with/slash",
			expExpression: "^/path/with/slash(/.*)?$",
			expLiteral:    13,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"path", "with", "slash"},
		},
		{
			name:          "Path with regex special characters in literal",
			template:      "/files/test.file",
			expExpression: "^/files/test\\.file(/.*)?$",
			expLiteral:    14,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"files", "test.file"},
		},
		{
			name:          "Path with plus sign",
			template:      "/search/hello+world",
			expExpression: "^/search/hello\\+world(/.*)?$",
			expLiteral:    17,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"search", "hello+world"},
		},
		{
			name:          "Path with ampersand",
			template:      "/api/a&b",
			expExpression: "^/api/a&b(/.*)?$",
			expLiteral:    6,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"api", "a&b"},
		},
		{
			name:          "Path with equals sign",
			template:      "/api/key=value",
			expExpression: "^/api/key=value(/.*)?$",
			expLiteral:    12,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"api", "key=value"},
		},
		{
			name:          "Path with question mark",
			template:      "/api/what?",
			expExpression: "^/api/what\\?(/.*)?$",
			expLiteral:    8,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"api", "what?"},
		},
		{
			name:          "Path with hash",
			template:      "/api/section#1",
			expExpression: "^/api/section#1(/.*)?$",
			expLiteral:    12,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"api", "section#1"},
		},
		{
			name:          "Mixed path with variables and special characters",
			template:      "/users/{name}/files/{filename:.*\\.txt}",
			expExpression: "^/users/([^/]+?)/files/(.*\\.txt)(/.*)?$",
			expLiteral:    10,
			expVarNames:   []string{"name", "filename"},
			expVarCount:   2,
			expTokens:     []string{"users", "{name}", "files", "{filename:.*\\.txt}"},
		},
		{
			name:          "Empty template",
			template:      "",
			expExpression: "^(/.*)?$",
			expLiteral:    0,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{""},
		},
		{
			name:          "Single literal",
			template:      "/api",
			expExpression: "^/api(/.*)?$",
			expLiteral:    3,
			expVarNames:   []string{},
			expVarCount:   0,
			expTokens:     []string{"api"},
		},
		{
			name:          "Single variable",
			template:      "/{id}",
			expExpression: "^/([^/]+?)(/.*)?$",
			expLiteral:    0,
			expVarNames:   []string{"id"},
			expVarCount:   1,
			expTokens:     []string{"{id}"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expression, literalCount, varNames, varCount, tokens := templateToRegExp(tc.template)

			if expression != tc.expExpression {
				t.Errorf("case %s no pass\ninput: %q\nexpected expression: %q\ngot: %q\n", tc.name, tc.template, tc.expExpression, expression)
			}

			if literalCount != tc.expLiteral {
				t.Errorf("case %s no pass\ninput: %q\nexpected literalCount: %d\ngot: %d\n", tc.name, tc.template, tc.expLiteral, literalCount)
			}

			if !reflect.DeepEqual(varNames, tc.expVarNames) {
				t.Errorf("case %s no pass\ninput: %q\nexpected varNames: %#v\ngot: %#v\n", tc.name, tc.template, tc.expVarNames, varNames)
			}

			if varCount != tc.expVarCount {
				t.Errorf("case %s no pass\ninput: %q\nexpected varCount: %d\ngot: %d\n", tc.name, tc.template, tc.expVarCount, varCount)
			}

			if !reflect.DeepEqual(tokens, tc.expTokens) {
				t.Errorf("case %s no pass\ninput: %q\nexpected tokens: %#v\ngot: %#v\n", tc.name, tc.template, tc.expTokens, tokens)
			}
		})
	}
}
