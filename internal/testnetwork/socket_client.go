package testnetwork

import (
	"net"
	"testing"
)

func ConnectToSocketServer(t *testing.T, network string, address string) net.Conn {
	t.Helper()

	conn, err := net.Dial(network, address)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}

	return conn
}
