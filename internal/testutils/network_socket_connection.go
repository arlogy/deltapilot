package testutils

import (
	"net"
	"testing"
)

func SetupSocketServer(t *testing.T, network string, address string) net.Listener {
	t.Helper()

	listener, err := net.Listen(network, address)
	if err != nil {
		t.Fatalf("failed to listen on %q (%q): %v", address, network, err)
	}
	t.Cleanup(func() {
		listener.Close()
	})

	return listener
}

func WaitForConnectionToSocketServer(t *testing.T, listener net.Listener) net.Conn {
	t.Helper()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("failed to accept client connection: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

func ConnectToSocketServer(t *testing.T, network string, address string) net.Conn {
	t.Helper()

	conn, err := net.Dial(network, address)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}
