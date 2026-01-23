package main

import (
	"errors"
	"fmt"
	"lcp/lib/flag"
	"lcp/lib/httpserver"
	"lcp/lib/logger"
	"lcp/lib/profile"
	"lcp/lib/utils/procutil"
	"net/http"
	"os"
	"time"
)

func main() {
	defer profile.Profile().Stop()

	// TODO: Load config file
	// TODO: Load env var

	flag.Parse()
	logger.Init()

	startTime := time.Now()

	addrs := []string{
		":8421",
		":8422",
		":8423",
	}
	go httpserver.Serve(addrs, requestHandler)

	logger.Infof("starting lcp-server in %.3f seconds", time.Since(startTime).Seconds())

	sig := procutil.WaitForSigterm()
	logger.Infof("received signal: %v", sig)

	logger.Infof("gracefully shutting down lcp-server at %q", addrs)
	startTime = time.Now()
	if err := httpserver.Stop(addrs); err != nil {
		logger.Fatalf("cannot stop the lcp-server: %s", err)
	}
	logger.Infof("successfully shut down lcp-server in %.3f seconds", time.Since(startTime).Seconds())

	// TODO: stop others
	logger.Infof("the lcp-server has been stopped in %.3f seconds", time.Since(startTime).Seconds())

}

func requestHandler(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" {
		_, _ = fmt.Fprintf(w, "ok")
	}
	return true
}

func Execute() {
	logger.Infof("info test")
	if err := NewServerCmd(); err != nil {
		os.Exit(1)
	}
}

func NewServerCmd() error {
	return errors.New("not implemented")
}
