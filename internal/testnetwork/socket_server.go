package testnetwork

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

	return listener
}

func WaitForConnectionToSocketServer(t *testing.T, listener net.Listener) net.Conn {
	t.Helper()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("failed to accept client connection: %v", err)
	}

	return conn
}
