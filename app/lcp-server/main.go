package main

import (
	"flag"
	"io/fs"
	"os"
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
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/ui"
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

	ctx := procutil.SetupSignalContext()
	cfg := loadConfig()

	// Database
	database, err := db.NewDB(ctx, dbConfigFrom(cfg))
	if err != nil {
		logger.Fatalf("cannot create database: %v", err)
	}

	// Hot-reload
	config.RegisterReloadCallback(func(c *config.Config) {
		logger.Reload(c.Logger.Level, c.Logger.Format)
		if err := database.Reload(ctx, dbConfigFrom(c)); err != nil {
			logger.Errorf("failed to reload database config: %v", err)
		}
	})
	go watchSIGHUP()

	// OIDC provider
	oidcProvider := apis.NewOIDCProvider(database, &cfg.OIDC)

	// API modules (permission sync, role seeding)
	apisResult := apis.NewAPIGroupInfos(ctx, database)

	// RBAC authorizer
	authorizer := apis.NewAuthorizer(database, apisResult.Groups)

	// 2. Start HTTP server
	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":8428"}
	}

	startTime := time.Now()

	apiHandler, err := handler.NewAPIServerHandler(handler.APIServerConfig{
		Name:         LCPAPIServer,
		OIDCProvider: oidcProvider,
		Authorizer:   authorizer,
	}, apisResult.Groups...)
	if err != nil {
		logger.Fatalf("cannot create API server handler: %v", err)
	}

	distFS, err := fs.Sub(ui.DistFS, "dist")
	if err != nil {
		logger.Fatalf("cannot load embedded frontend: %v", err)
	}

	rootHandler := handler.NewRootHandler(handler.RootHandlerConfig{
		APIHandler:  apiHandler,
		OIDCMux:     apis.NewOIDCMux(oidcProvider),
		OpenAPISpec: docs.OpenAPISpec,
		FrontendFS:  distFS,
	})

	go httpserver.Serve(listenAddrs, rootHandler, httpserver.ServerOptions{
		UseProxyProtocol: useProxyProtocol,
	})
	logger.Infof("lcp-server started at %q in %.3f seconds", listenAddrs, time.Since(startTime).Seconds())

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

// loadConfig loads configuration: file → defaults → env overrides → CLI overrides.
func loadConfig() *config.Config {
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

var cliFlags map[string]string

func initCLIFlags() {
	cliFlags = make(map[string]string)
	flag.Visit(func(f *flag.Flag) {
		cliFlags[f.Name] = f.Value.String()
	})
}

func applyCLIOverrides(cfg *config.Config) {
	for name, val := range cliFlags {
		switch name {
		case "loggerLevel":
			cfg.Logger.Level = val
		case "loggerFormat":
			cfg.Logger.Format = val
		}
	}
}

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

// watchSIGHUP listens for SIGHUP and reloads configuration.
func watchSIGHUP() {
	for range procutil.NewSighupChan() {
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
}

func usage() {
	const s = `
lcp-server is a PaaS management solution.

See the docs at https://docs.lcp.io/lcp/
`
	lflag.Usage(s)
}
