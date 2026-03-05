package main

import (
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"lcp.io/lcp/app/lcp-server/handler"
	"lcp.io/lcp/lib/buildinfo"
	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/httpserver"
	"lcp.io/lcp/lib/lflag"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/profile"
	"lcp.io/lcp/lib/utils/procutil"

	"lcp.io/lcp/docs"
	"lcp.io/lcp/pkg/apis"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
)

var (
	httpListenAddrs  = lflag.NewArrayString("httpListenerAddr", "The address to listen on for HTTP requests")
	useProxyProtocol = lflag.NewArrayBool("httpListenerAddr.useProxyProtocol", "Whether to use proxy protocol for connections accepted at the corresponding -httpListenAddr")
	configPath       = flag.String("config", "/etc/lcp/config.yaml", "Path to the YAML configuration file")
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
	initCLIFlags()
	buildinfo.Init()
	logger.Init()

	// Signal handling: returns cancellable context for SIGTERM/SIGINT
	ctx := procutil.SetupSignalContext()

	cfg := loadConfig()

	// Database
	dbCfg := dbConfigFrom(cfg)
	database, err := db.NewDB(ctx, dbCfg)
	if err != nil {
		logger.Fatalf("cannot create database: %v", err)
	}

	// Register reload callbacks
	config.RegisterReloadCallback(func(c *config.Config) {
		logger.Reload(c.Logger.Level, c.Logger.Format)
		if err := database.Reload(ctx, dbConfigFrom(c)); err != nil {
			logger.Errorf("failed to reload database config: %v", err)
		}
	})

	// Start SIGHUP listener for hot-reload
	sighupCh := procutil.NewSighupChan()
	go func() {
		for range sighupCh {
			logger.Infof("received SIGHUP, reloading configuration from %q", *configPath)
			newCfg, err := config.LoadFromFile(*configPath)
			if err != nil {
				logger.Errorf("failed to reload config: %v", err)
				continue
			}
			config.ApplyEnvOverrides(newCfg)
			applyCLIOverrides(newCfg)
			config.Set(newCfg)
			logger.Infof("configuration reloaded successfully")
		}
	}()

	// API module registration (includes OIDC provider setup)
	groups, oidcProvider := apis.NewAPIGroupInfos(database, cfg)

	// 2. Start http server
	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":8428"}
	}

	logger.Infof("starting lcp-server at %q...", listenAddrs)

	startTime := time.Now()

	apiHandler, err := handler.NewAPIServerHandler(LCPAPIServer, oidcProvider, groups...)
	if err != nil {
		logger.Fatalf("cannot create API server handler: %v", err)
	}

	// Build request handler: OIDC public endpoints + authenticated API
	var oidcMux http.Handler
	if oidcProvider != nil {
		oidcMux = iam.NewOIDCMux(oidcProvider)
	}

	requestHandler := func(w http.ResponseWriter, r *http.Request) bool {
		path := r.URL.Path
		// Route OIDC endpoints to public mux (no auth middleware)
		if oidcMux != nil && (strings.HasPrefix(path, "/.well-known/") || strings.HasPrefix(path, "/oidc/")) {
			oidcMux.ServeHTTP(w, r)
			return true
		}
		// Serve OpenAPI spec (no auth)
		if path == "/docs/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(docs.OpenAPISpec)
			return true
		}
		// All other requests go through the API handler (with auth middleware)
		return apiHandler.RequestHandler(w, r)
	}

	go httpserver.Serve(listenAddrs, requestHandler, httpserver.ServerOptions{
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
	database.Close()
	logger.Infof("successfully shut down lcp-server in %.3f seconds", time.Since(startTime).Seconds())
}

// loadConfig
func loadConfig() *config.Config {
	// Load configuration: file → defaults → env overrides → CLI overrides
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		logger.Fatalf("cannot load config from %q: %v", *configPath, err)
	}
	config.ApplyEnvOverrides(cfg)
	applyCLIOverrides(cfg)
	config.Set(cfg)
	logger.Infof("configuration loaded from %q", *configPath)
	return cfg
}

// cliFlags tracks which flags the user explicitly set on the command line.
// Populated once at startup (after flag parsing) and reused on every SIGHUP
// reload so that CLI values always take the highest priority.
var cliFlags map[string]string

func initCLIFlags() {
	cliFlags = make(map[string]string)
	flag.Visit(func(f *flag.Flag) {
		cliFlags[f.Name] = f.Value.String()
	})
}

// applyCLIOverrides re-applies command-line flag values that were explicitly
// set by the user, ensuring they always win over file and env values.
func applyCLIOverrides(cfg *config.Config) {
	for name, val := range cliFlags {
		switch name {
		case "loggerLevel":
			cfg.Logger.Level = val
		case "loggerFormat":
			cfg.Logger.Format = val
		}
		// Database-related CLI flags could be added here in the future.
	}
}

// dbConfigFrom converts a config.Config into a db.Config.
func dbConfigFrom(cfg *config.Config) db.Config {
	return db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		MaxConns: cfg.Database.MaxConns,
	}
}

func usage() {
	const s = `
lcp-server is a PaaS management solution.

See the docs at https://docs.lcp.io/lcp/
`
	lflag.Usage(s)
}
