package terminal

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestPauseWithTimeout_DoesNotConsumeKeyAfterTimeout(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Drain output so net.Pipe writes don't block.
	go io.Copy(io.Discard, client)

	term := New(server, 80, 24, false)

	done := make(chan error, 1)
	go func() {
		done <- term.PauseWithTimeout(1)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("PauseWithTimeout returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("PauseWithTimeout did not return after timeout")
	}

	// Now send a key; it should be consumed by the next GetKey(), not by PauseWithTimeout.
	if _, err := client.Write([]byte{'A'}); err != nil {
		t.Fatalf("client write: %v", err)
	}

	keyCh := make(chan byte, 1)
	errCh := make(chan error, 1)
	go func() {
		b, err := term.GetKey()
		if err != nil {
			errCh <- err
			return
		}
		keyCh <- b
	}()

	select {
	case b := <-keyCh:
		if b != 'A' {
			t.Fatalf("expected key 'A', got %q", b)
		}
	case err := <-errCh:
		t.Fatalf("GetKey returned error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatalf("GetKey did not return")
	}
}

func TestPauseWithTimeout_ConsumesKeyBeforeTimeout(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go io.Copy(io.Discard, client)

	term := New(server, 80, 24, false)

	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- term.PauseWithTimeout(5)
	}()

	// Press a key shortly after starting.
	time.Sleep(150 * time.Millisecond)
	if _, err := client.Write([]byte{'X'}); err != nil {
		t.Fatalf("client write: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("PauseWithTimeout returned error: %v", err)
		}
		if time.Since(start) > 2*time.Second {
			t.Fatalf("PauseWithTimeout did not return promptly after keypress")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("PauseWithTimeout did not return after keypress")
	}

	// The key should have been consumed by PauseWithTimeout, so the next GetKey
	// should block until we send another key.
	getDone := make(chan struct{})
	go func() {
		_, _ = term.GetKey()
		close(getDone)
	}()

	select {
	case <-getDone:
		t.Fatalf("expected GetKey to block (key should have been consumed)")
	case <-time.After(250 * time.Millisecond):
		// ok
	}

	if _, err := client.Write([]byte{'Y'}); err != nil {
		t.Fatalf("client write: %v", err)
	}

	select {
	case <-getDone:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatalf("GetKey did not unblock after sending second key")
	}
}
