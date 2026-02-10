package rest

import (
	"fmt"
	"net/http"
	"testing"
)

func mockRouteFunction(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s matched\n", r.URL.Path)
}

func TestSelectRoutes(t *testing.T) {
	container := NewContainer()
	ws := new(WebService)
	ws.
		Path("/api/v1").
		Produces(MIME_JSON)

	ws.Route(ws.GET("/users").To(mockRouteFunction))
	ws.Route(ws.GET("/users/{id}").To(mockRouteFunction))
	ws.Route(ws.POST("/users").To(mockRouteFunction))
	ws.Route(ws.GET("/users/{id}/orders").To(mockRouteFunction))
	ws.Route(ws.GET("/products/{category}/{id}").To(mockRouteFunction))
	container.Add(ws)

	services := container.RegisteredWebServices()
	if len(services) < 1 {
		t.Fatal("no services registered")
	}

	cases := []struct {
		name           string
		path           string
		expectedRoutes int
	}{
		{
			name:           "/api/v1/users",
			path:           "/api/v1/users",
			expectedRoutes: 2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requestTokens := tokenizePath(c.path)

			router := CurlyRouter{}
			candidates := router.selectRoutes(ws, requestTokens)
			if len(candidates) != c.expectedRoutes {
				t.Errorf("expected %d candidate routes, got %d", c.expectedRoutes, len(candidates))
			}
			for i, candidate := range candidates {
				t.Logf("candidate[%d]: path=%s, paramCount=%d, staticCount=%d",
					i, candidate.Path, candidate.paramCount, candidate.staticCount)
			}
		})
	}
}
