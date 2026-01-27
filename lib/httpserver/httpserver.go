package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"lcp.io/lcp/lib/lflag"
	"lcp.io/lcp/lib/logger"
)

var (
	tlsEnable   = lflag.NewArrayBool("tls", "Whether to enable TLS for the server, -tlsCertFile and -tlsKeyFile must be set if -tls is set")
	tlsCertFile = lflag.NewArrayString("tlsCertFile", "Path to file with TLS certificate for the corresponding -httpListenAddr if -tls is set."+
		"Prefer ECDSA certs instead of RSA certs as RSA certs are slower")
	tlsKeyFile                  = lflag.NewArrayString("tlsKeyFile", "Path to file with TLS key for the corresponding -httpListenAddr if -tls is set")
	maxGracefulShutdownDuration = flag.Duration("http.maxGracefulShutdownDuration", 7*time.Second, `The maximum duration for a graceful shutdown of the HTTP server. A highly loaded server may require increased value for a graceful shutdown`)
	shutdownDelay               = flag.Duration("http.shutdownDelay", 0, `Optional delay before http server shutdown. During this delay, the server returns non-OK responses from /health page, so load balancers can route new requests to other servers`)
	idleConnTimeout             = flag.Duration("http.idleConnTimeout", time.Minute, "Timeout for incoming idle http connections")
	connTimeout                 = flag.Duration("http.connTimeout", 2*time.Minute, "Incoming connections to -httpListenAddr are closed after the configured timeout. "+
		"This may help evenly spreading load among a cluster of services behind TCP-level load balancer. Zero value disables closing of incoming connections")
	headerHSTS         = flag.String("http.header.hsts", "max-age=31536000; includeSubDomains", "Value for 'Strict-Transport-Security' header, recommended: 'max-age=31536000; includeSubDomains'")
	headerFrameOptions = flag.String("http.header.frameOptions", "SAMEORIGIN", "Value for 'X-Frame-Options' header")
	headerCSP          = flag.String("http.header.csp", "default-src 'self'", `Value for 'Content-Security-Policy' header, recommended: "default-src 'self'"`)
)

var (
	servers     = make(map[string]*server)
	serversLock sync.Mutex
)

var hostname = func() string {
	h, err := os.Hostname()
	if err != nil {
		// Cannot use logger.Errorf, since it isn't initialized yet.
		// So use log.Printf instead.
		log.Printf("ERROR: cannot determine hostname: %s", err)
		return "unknown"
	}
	return h
}()

type server struct {
	s                     *http.Server
	shutdownDelayDeadline atomic.Int64
}

// RequestHandler must serve the given request r and write response to w
//
// RequestHandler must return true if the request has been served (successfully or not)
//
// RequestHandler must return false if it cannot serve the given request
type RequestHandler func(w http.ResponseWriter, r *http.Request) bool

type ServerOptions struct {
	// UseProxyProtocol if is set to true for the corresponding addr, then the incoming connections are accepted via proxy protocol
	UseProxyProtocol *lflag.ArrayBool
}

// Serve starts an http server on the given addresses with the given optional request handler
func Serve(addrs []string, rh RequestHandler, opts ServerOptions) {
	if rh == nil {
		rh = func(_ http.ResponseWriter, _ *http.Request) bool { return false }
	}
	for idx, addr := range addrs {
		if addr == "" {
			continue
		}
		logger.Infof("starting http server on %s", addr)
		go serve(addr, rh, idx, opts)
	}
}

func serve(addr string, rh RequestHandler, idx int, opts ServerOptions) {
	scheme := "http"
	if tlsEnable.GetOptionalArg(idx) {
		scheme = "https"
	}
	useProxyProto := false
	if opts.UseProxyProtocol != nil {
		useProxyProto = opts.UseProxyProtocol.GetOptionalArg(idx)
	}

	var tlsConfig *tls.Config
	if tlsEnable.GetOptionalArg(idx) {
		certFile := tlsCertFile.GetOptionalArg(idx)
		keyFile := tlsKeyFile.GetOptionalArg(idx)
		tc, err := GetServerTLSConfig(certFile, keyFile)
		if err != nil {
			logger.Fatalf("cannot load TLS cert from -tlsCertFile=%q, -tlsKeyFile=%q: %s", certFile, keyFile, err)
		}
		tlsConfig = tc
	}
	// create a TCP listener
	ln, err := NewTCPListener(scheme, addr, useProxyProto, tlsConfig)
	if err != nil {
		logger.Fatalf("cannot start http server on %s: %v", addr, err)
	}
	logger.Infof("started http server on %s://%s/", scheme, ln.Addr())

	serveWithListener(addr, ln, rh)
}

func serveWithListener(addr string, ln net.Listener, rh RequestHandler) {
	var s server

	s.s = &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          logger.StdErrorLogger(),
	}

	s.s.SetKeepAlivesEnabled(true)

	// Set handler
	rhw := rh
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerWrapper(w, r, rhw)
	})
	s.s.Handler = h

	serversLock.Lock()
	servers[addr] = &s
	serversLock.Unlock()
	if err := s.s.Serve(ln); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		logger.Panicf("FATAL: cannot serve http server on %s: %v", addr, err)
	}
}

// Stop stops the http server on the given addrs, which has been started via Serve func
func Stop(addrs []string) error {
	var errGlobalLock sync.Mutex
	var errGlobal error

	var wg sync.WaitGroup
	for _, addr := range addrs {
		if addr == "" {
			continue
		}
		wg.Add(1)
		go func(addr string) {
			if err := stop(addr); err != nil {
				errGlobalLock.Lock()
				errGlobal = err
				errGlobalLock.Unlock()
			}
			wg.Done()
		}(addr)
	}
	wg.Wait()

	return errGlobal
}

func stop(addr string) error {
	serversLock.Lock()
	s := servers[addr]
	delete(servers, addr)
	serversLock.Unlock()
	if s == nil {
		err := fmt.Errorf("BUG: there is no server at %q", addr)
		logger.Panicf("%s", err)
		return err
	}

	deadline := time.Now().Add(*shutdownDelay).UnixNano()
	s.shutdownDelayDeadline.Store(deadline)
	if *shutdownDelay > 0 {
		// Sleep for a while until load balancer in front of the server
		// notifies that "/health" endpoint returns non-OK responses
		logger.Infof("Waiting for %.3fs before shutdown of http server %q, so load balancers could re-route requests to other servers", shutdownDelay.Seconds(), addr)
		time.Sleep(*shutdownDelay)
		logger.Infof("Starting shutdown for http server %q", addr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *maxGracefulShutdownDuration)
	defer cancel()
	if err := s.s.Shutdown(ctx); err != nil {
		return fmt.Errorf("cannot gracefully shutdown http server at %q in %.3fs; "+
			"probably, `-http.maxGracefulShutdownDuration` command-line flag value must be increased; error: %s", addr, maxGracefulShutdownDuration.Seconds(), err)
	}
	return nil
}

func handlerWrapper(w http.ResponseWriter, r *http.Request, rh RequestHandler) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, false)
			_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n\n%s", err, buf[:n])
			os.Exit(1)
		}
	}()

	h := w.Header()
	if *headerHSTS != "" {
		h.Add("Strict-Transport-Security", *headerHSTS)
	}
	if *headerFrameOptions != "" {
		h.Add("X-Frame-Options", *headerFrameOptions)
	}
	if *headerCSP != "" {
		h.Add("Content-Security-Policy", *headerCSP)
	}
	h.Add("X-Server-Hostname", hostname)

	if rh(w, r) {
		return
	}
}
