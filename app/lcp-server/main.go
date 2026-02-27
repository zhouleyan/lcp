package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"lcp.io/lcp/app/lcp-server/handler"
	"lcp.io/lcp/lib/buildinfo"
	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/httpserver"
	"lcp.io/lcp/lib/lflag"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/profile"
	"lcp.io/lcp/lib/service"
	storepg "lcp.io/lcp/lib/store/pg"
	"lcp.io/lcp/lib/utils/procutil"
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

	// TODO: Load config file
	// TODO: Load env var

	// 1. Initialize
	// Write flags and help message to stdout, since it is easier to grep or pipe
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = usage
	lflag.Parse()
	buildinfo.Init()
	logger.Init()

	// 1.5 Initialize database, store, and service
	ctx := context.Background()
	dbCfg := db.Config{
		Host:     envOrDefault("DB_HOST", "localhost"),
		Port:     envOrDefaultInt("DB_PORT", 5432),
		User:     envOrDefault("DB_USER", "lcp"),
		Password: envOrDefault("DB_PASSWORD", "lcp"),
		DBName:   envOrDefault("DB_NAME", "lcp"),
		SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
		MaxConns: int32(envOrDefaultInt("DB_MAX_CONNS", 10)),
	}
	pool, err := db.NewPool(ctx, dbCfg)
	if err != nil {
		logger.Fatalf("cannot create database pool: %v", err)
	}
	defer pool.Close()

	s := storepg.New(pool)
	defer s.Close()

	svc := service.New(s)

	// 2. Start http server
	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":8428"}
	}

	logger.Infof("starting lcp-server at %q...", listenAddrs)

	startTime := time.Now()

	apiHandler, err := handler.NewAPIServerHandler(LCPAPIServer, svc)
	if err != nil {
		logger.Fatalf("cannot create API server handler: %v", err)
	}

	go httpserver.Serve(listenAddrs, apiHandler.RequestHandler, httpserver.ServerOptions{
		UseProxyProtocol: useProxyProtocol,
	})
	logger.Infof("starting lcp-server in %.3f seconds", time.Since(startTime).Seconds())

	// 3. Wait for signal to stop server
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal: %v", sig)

	logger.Infof("gracefully shutting down lcp-server at %q", listenAddrs)
	startTime = time.Now()
	if err := httpserver.Stop(listenAddrs); err != nil {
		logger.Fatalf("cannot stop the lcp-server: %s", err)
	}
	logger.Infof("successfully shut down lcp-server in %.3f seconds", time.Since(startTime).Seconds())

	logger.Infof("the lcp-server has been stopped in %.3f seconds", time.Since(startTime).Seconds())

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
