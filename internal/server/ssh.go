package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"crypto/x509"

	"golang.org/x/crypto/ssh"
)

// SSHConn wraps an SSH channel to provide an io.ReadWriteCloser
// compatible with the BBS terminal system.
type SSHConn struct {
	channel ssh.Channel
	mu      sync.Mutex

	// Terminal properties from SSH
	Width       int
	Height      int
	ANSICapable bool
	TermType    string
}

// NewSSHConn wraps an SSH channel.
func NewSSHConn(channel ssh.Channel, width, height int, termType string) *SSHConn {
	return &SSHConn{
		channel:     channel,
		Width:       width,
		Height:      height,
		ANSICapable: true, // SSH clients are typically ANSI-capable
		TermType:    termType,
	}
}

// Read implements io.Reader.
func (sc *SSHConn) Read(p []byte) (int, error) {
	return sc.channel.Read(p)
}

// Write implements io.Writer.
func (sc *SSHConn) Write(p []byte) (int, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.channel.Write(p)
}

// Close implements io.Closer.
func (sc *SSHConn) Close() error {
	return sc.channel.Close()
}

// RemoteAddr returns a placeholder address (SSH channels don't expose this directly).
func (sc *SSHConn) RemoteAddr() string {
	return "ssh"
}

// SetEcho is a no-op for SSH (echo is handled by the client).
func (sc *SSHConn) SetEcho(on bool) error {
	return nil
}

// SSHListener accepts incoming SSH connections.
type SSHListener struct {
	addr     string
	config   *ssh.ServerConfig
	handler  func(conn *SSHConn, remoteAddr string)
	hostKeyPath string
}

// NewSSHListener creates a new SSH listener.
func NewSSHListener(port int, hostKeyPath string, handler func(conn *SSHConn, remoteAddr string)) (*SSHListener, error) {
	config := &ssh.ServerConfig{
		// Password auth callback - we let the BBS handle actual auth
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Accept all passwords at SSH level - the BBS login menu handles real auth
			return nil, nil
		},
		NoClientAuth: true, // Allow connections without auth at SSH level
	}

	l := &SSHListener{
		addr:        fmt.Sprintf(":%d", port),
		config:      config,
		handler:     handler,
		hostKeyPath: hostKeyPath,
	}

	// Load or generate host key
	if err := l.loadOrGenerateHostKey(); err != nil {
		return nil, fmt.Errorf("host key: %w", err)
	}

	return l, nil
}

// loadOrGenerateHostKey loads an existing host key or generates a new one.
func (l *SSHListener) loadOrGenerateHostKey() error {
	// Try to load existing key
	if data, err := os.ReadFile(l.hostKeyPath); err == nil {
		key, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return fmt.Errorf("parse host key: %w", err)
		}
		l.config.AddHostKey(key)
		log.Printf("SSH: loaded host key from %s", l.hostKeyPath)
		return nil
	}

	// Generate new ED25519 key
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	// Marshal to PKCS8 and PEM-encode
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	pemData := pem.EncodeToMemory(pemBlock)

	// Save to file
	dir := filepath.Dir(l.hostKeyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}
	if err := os.WriteFile(l.hostKeyPath, pemData, 0600); err != nil {
		return fmt.Errorf("write host key: %w", err)
	}

	key, err := ssh.ParsePrivateKey(pemData)
	if err != nil {
		return fmt.Errorf("parse new key: %w", err)
	}
	l.config.AddHostKey(key)

	log.Printf("SSH: generated new host key at %s", l.hostKeyPath)
	return nil
}

// ListenAndServe starts accepting SSH connections.
func (l *SSHListener) ListenAndServe() error {
	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", l.addr, err)
	}
	defer ln.Close()

	log.Printf("SSH server listening on %s", l.addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("SSH accept error: %v", err)
			continue
		}

		go l.handleConnection(conn)
	}
}

// handleConnection processes a single SSH connection.
func (l *SSHListener) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, l.config)
	if err != nil {
		log.Printf("SSH handshake failed from %s: %v", remoteAddr, err)
		conn.Close()
		return
	}
	defer sshConn.Close()

	log.Printf("SSH connection from %s (user: %s)", remoteAddr, sshConn.User())

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Accept channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("SSH channel accept error: %v", err)
			continue
		}

		// Handle session requests (pty-req, shell, window-change)
		width := 80
		height := 24
		termType := "xterm"

		go func() {
			for req := range requests {
				switch req.Type {
				case "pty-req":
					// Parse PTY request
					if len(req.Payload) >= 4 {
						termLen := int(req.Payload[3])
						if len(req.Payload) >= 4+termLen+8 {
							termType = string(req.Payload[4 : 4+termLen])
							offset := 4 + termLen
							width = int(req.Payload[offset])<<24 | int(req.Payload[offset+1])<<16 |
								int(req.Payload[offset+2])<<8 | int(req.Payload[offset+3])
							height = int(req.Payload[offset+4])<<24 | int(req.Payload[offset+5])<<16 |
								int(req.Payload[offset+6])<<8 | int(req.Payload[offset+7])
						}
					}
					if req.WantReply {
						req.Reply(true, nil)
					}

				case "shell":
					if req.WantReply {
						req.Reply(true, nil)
					}
					// Create SSHConn and hand off to BBS
					sc := NewSSHConn(channel, width, height, termType)
					l.handler(sc, remoteAddr)
					channel.Close()
					return

				case "window-change":
					if len(req.Payload) >= 8 {
						width = int(req.Payload[0])<<24 | int(req.Payload[1])<<16 |
							int(req.Payload[2])<<8 | int(req.Payload[3])
						height = int(req.Payload[4])<<24 | int(req.Payload[5])<<16 |
							int(req.Payload[6])<<8 | int(req.Payload[7])
					}

				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}()
	}
}

// Ensure SSHConn implements io.ReadWriteCloser.
var _ io.ReadWriteCloser = (*SSHConn)(nil)
