package httpserver

import (
	"crypto/tls"
	"flag"
	"net"
)

var enableTCP6 = flag.Bool("enableTCP6", false, "Whether to enable IPv6 for listening and dialing. By default, only IPv4 TCP and UDP are used")

func NewTCPListener(name, addr string, tlsConfig *tls.Config) (net.Listener, error) {
	network := GetTCPNetwork()
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	tln := &TCPListener{
		Listener:  ln,
		tlsConfig: tlsConfig,
	}
	return tln, err
}

// TCPListener listens for the addr passed to NewTCPListener
type TCPListener struct {
	net.Listener

	tlsConfig *tls.Config
}

// Accept accepts connections from the addr passed to NewTCPListener
//func (ln *TCPListener) Accept() (net.Conn, error) {
//	for {
//		conn, err := ln.Listener.Accept()
//	}
//}

// GetTCPNetwork returns current tcp network.
func GetTCPNetwork() string {
	if *enableTCP6 {
		// Enable both tcp4 and tcp6
		return "tcp"
	}
	return "tcp4"
}
