package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

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

// EnterBinaryMode returns the raw SSH channel for binary transfers.
// SSH channels are already binary-safe so no special handling is needed.
func (sc *SSHConn) EnterBinaryMode() (io.ReadWriter, func(), bool) {
	rw := struct {
		io.Reader
		io.Writer
	}{sc.channel, sc.channel}
	return rw, func() {}, false
}

// SSHListener accepts incoming SSH connections.
type SSHListener struct {
	addr        string
	config      *ssh.ServerConfig
	handler     func(conn *SSHConn, remoteAddr string)
	hostKeyPath string

	attemptMu sync.Mutex
	attempts  map[string]*sshAttempt
}

// NewSSHListener creates a new SSH listener.
func NewSSHListener(port int, hostKeyPath string, handler func(conn *SSHConn, remoteAddr string)) (*SSHListener, error) {
	config := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: []string{
				// Modern first
				"curve25519-sha256",
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
				"diffie-hellman-group-exchange-sha256",
				"diffie-hellman-group14-sha256",
				"diffie-hellman-group16-sha512",
				// Legacy for older SSH clients (SyncTerm cryptlib/libssh2)
				"diffie-hellman-group-exchange-sha1",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group1-sha1",
			},
			Ciphers: []string{
				// Modern first
				"chacha20-poly1305@openssh.com",
				"aes128-gcm@openssh.com",
				"aes256-gcm@openssh.com",
				"aes128-ctr",
				"aes192-ctr",
				"aes256-ctr",
				// Legacy CBC modes for older SSH clients
				"aes128-cbc",
				"3des-cbc",
			},
			MACs: []string{
				// Modern first
				"hmac-sha2-256-etm@openssh.com",
				"hmac-sha2-512-etm@openssh.com",
				"hmac-sha2-256",
				"hmac-sha2-512",
				// Legacy
				"hmac-sha1",
			},
		},
		ServerVersion: "SSH-2.0-TwilightBBS",
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
		attempts:    make(map[string]*sshAttempt),
	}

	// Load or generate host key
	if err := l.loadOrGenerateHostKey(); err != nil {
		return nil, fmt.Errorf("host key: %w", err)
	}

	return l, nil
}

// loadOrGenerateHostKey loads an existing host key or generates a new one.
func (l *SSHListener) loadOrGenerateHostKey() error {
	loadKey := func(path string) (ssh.Signer, bool, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, false, fmt.Errorf("parse host key %s: %w", path, err)
		}
		return signer, true, nil
	}

	wrapRSA := func(signer ssh.Signer) ssh.Signer {
		// Newer Go x/crypto only advertises rsa-sha2-256/512 for RSA keys.
		// SyncTerm (libssh2) only understands "ssh-rsa", so we must include it.
		if signer.PublicKey().Type() == ssh.KeyAlgoRSA {
			if as, ok := signer.(ssh.AlgorithmSigner); ok {
				wrapped, err := ssh.NewSignerWithAlgorithms(as, []string{
					ssh.KeyAlgoRSASHA512,
					ssh.KeyAlgoRSASHA256,
					ssh.KeyAlgoRSA, // legacy ssh-rsa for SyncTerm/libssh2
				})
				if err == nil {
					return wrapped
				}
				log.Printf("SSH: warning: could not wrap RSA signer: %v", err)
			}
		}
		return signer
	}

	addKeyFromPath := func(path string) (bool, error) {
		signer, ok, err := loadKey(path)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		l.config.AddHostKey(wrapRSA(signer))
		log.Printf("SSH: loaded host key from %s (%s)", path, signer.PublicKey().Type())
		return true, nil
	}

	ensureDir := func() error {
		dir := filepath.Dir(l.hostKeyPath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create key dir: %w", err)
		}
		return nil
	}

	// 1) Primary host key (existing behavior): ED25519 in l.hostKeyPath
	if ok, err := addKeyFromPath(l.hostKeyPath); err != nil {
		return err
	} else if !ok {
		// Generate new ED25519 key
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("generate ed25519 key: %w", err)
		}

		// Marshal to PKCS8 and PEM-encode
		privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return fmt.Errorf("marshal ed25519 key: %w", err)
		}

		pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

		if err := ensureDir(); err != nil {
			return err
		}
		if err := os.WriteFile(l.hostKeyPath, pemData, 0600); err != nil {
			return fmt.Errorf("write host key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(pemData)
		if err != nil {
			return fmt.Errorf("parse new ed25519 key: %w", err)
		}
		l.config.AddHostKey(signer)
		log.Printf("SSH: generated new host key at %s (%s)", l.hostKeyPath, signer.PublicKey().Type())
	}

	// 2) Additional RSA host key for legacy SSH clients (e.g., SyncTerm/libssh2)
	rsaKeyPath := l.hostKeyPath + "_rsa"
	if ok, err := addKeyFromPath(rsaKeyPath); err != nil {
		return err
	} else if !ok {
		priv, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return fmt.Errorf("generate rsa key: %w", err)
		}

		privBytes := x509.MarshalPKCS1PrivateKey(priv)
		pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})

		if err := ensureDir(); err != nil {
			return err
		}
		if err := os.WriteFile(rsaKeyPath, pemData, 0600); err != nil {
			return fmt.Errorf("write rsa host key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(pemData)
		if err != nil {
			return fmt.Errorf("parse new rsa key: %w", err)
		}
		l.config.AddHostKey(wrapRSA(signer))
		log.Printf("SSH: generated new host key at %s (%s)", rsaKeyPath, signer.PublicKey().Type())
	}

	return nil
}

type sshAttempt struct {
	last  time.Time
	count int
}

func (l *SSHListener) allowConnection(host string) (time.Duration, bool) {
	// Simple per-host backoff to reduce anonymous connection abuse.
	// Not perfect, but it meaningfully raises the cost of flooding.
	const (
		window     = 10 * time.Second
		resetAfter = 30 * time.Second
		maxCount   = 30
		step       = 250 * time.Millisecond
		maxDelay   = 5 * time.Second
	)

	now := time.Now()

	l.attemptMu.Lock()
	defer l.attemptMu.Unlock()

	a := l.attempts[host]
	if a == nil {
		a = &sshAttempt{last: now}
		l.attempts[host] = a
	}

	if now.Sub(a.last) > resetAfter {
		a.count = 0
	}
	if now.Sub(a.last) <= window {
		a.count++
	} else {
		a.count = 1
	}
	a.last = now

	if a.count > maxCount {
		return 0, false
	}

	if a.count <= 3 {
		return 0, true
	}
	d := time.Duration(a.count-3) * step
	if d > maxDelay {
		d = maxDelay
	}
	return d, true
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
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	if delay, ok := l.allowConnection(host); !ok {
		conn.Close()
		return
	} else if delay > 0 {
		time.Sleep(delay)
	}

	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	defer conn.SetDeadline(time.Time{})

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, l.config)
	if err != nil {
		log.Printf("SSH handshake failed from %s: %v", remoteAddr, err)
		conn.Close()
		return
	}
	defer sshConn.Close()
	_ = conn.SetDeadline(time.Time{})

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
