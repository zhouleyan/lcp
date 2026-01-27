package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"lcp.io/lcp/lib/buildinfo"
	"lcp.io/lcp/lib/httpserver"
	"lcp.io/lcp/lib/lflag"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/profile"
	"lcp.io/lcp/lib/utils/procutil"
)

var (
	httpListenAddrs  = lflag.NewArrayString("httpListenerAddr", "The address to listen on for HTTP requests")
	useProxyProtocol = lflag.NewArrayBool("httpListenerAddr.useProxyProtocol", "Whether to use proxy protocol for connections accepted at the corresponding -httpListenAddr")
)

func main() {
	defer profile.Profile().Stop()

	// TODO: Load config file
	// TODO: Load env var

	// Write flags and help message to stdout, since it is easier to grep or pipe.
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = usage
	lflag.Parse()
	buildinfo.Init()
	logger.Init()

	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":8428"}
	}

	logger.Infof("starting lcp-server at %q...", listenAddrs)

	startTime := time.Now()

	go httpserver.Serve(listenAddrs, requestHandler, httpserver.ServerOptions{
		UseProxyProtocol: useProxyProtocol,
	})
	logger.Infof("starting lcp-server in %.3f seconds", time.Since(startTime).Seconds())

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
