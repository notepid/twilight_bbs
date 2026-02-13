package server

import (
	"fmt"
	"log"
	"net"
)

// ConnectionHandler is called for each new telnet connection.
// The handler is responsible for running the connection and closing it.
type ConnectionHandler func(tc *TelnetConn)

// Listener accepts incoming telnet connections.
type Listener struct {
	addr    string
	handler ConnectionHandler
}

// NewListener creates a new TCP listener for telnet connections.
func NewListener(port int, handler ConnectionHandler) *Listener {
	return &Listener{
		addr:    fmt.Sprintf(":%d", port),
		handler: handler,
	}
}

// ListenAndServe starts accepting connections. Blocks until the listener
// is closed or a fatal error occurs.
func (l *Listener) ListenAndServe() error {
	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", l.addr, err)
	}
	defer ln.Close()

	log.Printf("Telnet server listening on %s", l.addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		tc := NewTelnetConn(conn)
		go l.handler(tc)
	}
}
