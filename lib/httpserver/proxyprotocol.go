package httpserver

import (
	"net"
	"sync"
)

type proxyProtocolConn struct {
	net.Conn
	once       sync.Once
	remoteAddr net.Addr
	readErr    error
}

func newProxyProtocolConn(c net.Conn) net.Conn {
	return &proxyProtocolConn{
		Conn: c,
	}
}
