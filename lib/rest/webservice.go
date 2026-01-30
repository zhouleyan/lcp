package rest

import "lcp.io/lcp/lib/logger"

// WebService holds a collection of Route values that bind a HTTP Method + URL Path to a function
type WebService struct {
	rootPath   string
	pathExpr   *pathExpression // cached compilation of rootPath as RegExp
	routes     []Route
	apiVersion string
}

// RootPath returns the RootPath associated with this WebService. Default "/"
func (w *WebService) RootPath() string {
	return w.rootPath
}

// Path specifies the root URL template path of the WebService
// All Routes will be relative to this path
func (w *WebService) Path(root string) *WebService {
	w.rootPath = root
	if len(w.rootPath) == 0 {
		w.rootPath = "/"
	}
	w.compilePathExpression()
	return w
}

// compilePathExpression ensures that the path is compiled into a RegEx for those Routes that need it
func (w *WebService) compilePathExpression() {
	compiled, err := newPathExpression(w.rootPath)
	if err != nil {
		logger.Fatalf("invalid path: %s, %v", w.rootPath, err)
	}
	w.pathExpr = compiled
}
