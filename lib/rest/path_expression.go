package rest

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// pathExpression holds a compiled path expression (RegExp) needed match against
// HTTP request paths and to extract path parameter values
type pathExpression struct {
	LiteralCount int      // the number of literal characters (means those not resulting from template variable substitution)
	VarNames     []string // the names of parameters (enclosed by {}) in the path
	VarCount     int      // the number of named parameters (enclosed by {}) in the path
	Matcher      *regexp.Regexp
	Source       string // Path as defined by the RouteBuilder
	tokens       []string
}

func newPathExpression(path string) (*pathExpression, error) {
	expression, literalCount, varNames, varCount, tokens := templateToRegExp(path)
	compiled, err := regexp.Compile(expression)
	if err != nil {
		return nil, err
	}
	return &pathExpression{literalCount, varNames, varCount, compiled, expression, tokens}, nil
}

func templateToRegExp(template string) (expression string, literalCount int, varNames []string, varCount int, tokens []string) {
	var buf bytes.Buffer
	varNames = []string{}
	buf.WriteString("^")
	tokens = TokenizePath(template)
	for _, each := range tokens {
		if each == "" {
			continue
		}
		buf.WriteString("/")
		if strings.HasPrefix(each, "{") {
			// check for RegExp in variable
			colon := strings.Index(each, ":")
			var varName string
			if colon != -1 {
				// extract expression
				varName = strings.TrimSpace(each[1:colon])
				paramExpr := strings.TrimSpace(each[colon+1 : len(each)-1])
				if paramExpr == "*" {
					// special case
					buf.WriteString("(.*)")
				} else {
					buf.WriteString(fmt.Sprintf("(%s)", paramExpr))
				}
			} else {
				// plain var
				varName = strings.TrimSpace(each[1 : len(each)-1])
				buf.WriteString("([^/]+?)")
			}
			varNames = append(varNames, varName)
			varCount += 1
		} else {
			literalCount += len(each)
			//encoded := url.PathEscape(each)
			encoded := each
			buf.WriteString(regexp.QuoteMeta(encoded))
		}
	}
	return strings.TrimRight(buf.String(), "/") + "(/.*)?$", literalCount, varNames, varCount, tokens
}
