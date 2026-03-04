package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"lcp.io/lcp/app/lcp-server/handler"
	"lcp.io/lcp/lib/buildinfo"
	"lcp.io/lcp/lib/httpserver"
	"lcp.io/lcp/lib/lflag"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/profile"
	"lcp.io/lcp/lib/utils/procutil"

	"lcp.io/lcp/pkg/apis"
	"lcp.io/lcp/pkg/db"
)

var (
	httpListenAddrs  = lflag.NewArrayString("httpListenerAddr", "The address to listen on for HTTP requests")
	useProxyProtocol = lflag.NewArrayBool("httpListenerAddr.useProxyProtocol", "Whether to use proxy protocol for connections accepted at the corresponding -httpListenAddr")
)

const (
	LCPAPIServer = "lcp-server"
)

func main() {
	defer profile.Profile().Stop()

	// 1. Initialize
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = usage
	lflag.Parse()
	buildinfo.Init()
	logger.Init()

	// Signal handling: returns cancellable context
	ctx := procutil.SetupSignalContext()

	// Database
	dbCfg := db.Config{
		Host:     envOrDefault("DB_HOST", "localhost"),
		Port:     envOrDefaultInt("DB_PORT", 5432),
		User:     envOrDefault("DB_USER", "lcp"),
		Password: envOrDefault("DB_PASSWORD", "lcp"),
		DBName:   envOrDefault("DB_NAME", "lcp"),
		SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
		MaxConns: int32(envOrDefaultInt("DB_MAX_CONNS", 10)),
	}
	database, err := db.NewDB(ctx, dbCfg)
	if err != nil {
		logger.Fatalf("cannot create database: %v", err)
	}
	defer database.Close()

	// API module registration
	groups := apis.NewAPIGroupInfos(database)

	// 2. Start http server
	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":8428"}
	}

	logger.Infof("starting lcp-server at %q...", listenAddrs)

	startTime := time.Now()

	apiHandler, err := handler.NewAPIServerHandler(LCPAPIServer, groups...)
	if err != nil {
		logger.Fatalf("cannot create API server handler: %v", err)
	}

	go httpserver.Serve(listenAddrs, apiHandler.RequestHandler, httpserver.ServerOptions{
		UseProxyProtocol: useProxyProtocol,
	})
	logger.Infof("starting lcp-server in %.3f seconds", time.Since(startTime).Seconds())

	// 3. Wait for shutdown signal
	<-ctx.Done()

	logger.Infof("gracefully shutting down lcp-server at %q", listenAddrs)
	startTime = time.Now()
	if err := httpserver.Stop(listenAddrs); err != nil {
		logger.Fatalf("cannot stop the lcp-server: %s", err)
	}
	logger.Infof("successfully shut down lcp-server in %.3f seconds", time.Since(startTime).Seconds())
}

func requestHandler(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" {
		_, _ = fmt.Fprintf(w, "ok")
	}
	return true
}

func usage() {
	const s = `
lcp-server is a PaaS management solution.

See the docs at https://docs.lcp.io/lcp/
`
	lflag.Usage(s)
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
