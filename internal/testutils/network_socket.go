package testutils

import (
	"net"
	"os"
	"path/filepath"
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

type SocketEndpoint struct {
	Network string
	Address string
	Release func(t *testing.T) // releases any resource this endpoint may have acquired
}

var socketTestFilePath = filepath.Join(os.TempDir(), "deltapilot_socket_test.sock")

var SocketTCPIPAny = SocketEndpoint{
	Network: "tcp",
	Address: "127.0.0.1:0", // port 0 lets the OS choose any available port
	Release: func(t *testing.T) {},
}

var SocketTCPIPv4 = SocketEndpoint{
	Network: "tcp4",
	Address: "127.0.0.1:0", // port 0 lets the OS choose any available port
	Release: func(t *testing.T) {},
}

var SocketTCPIPv6 = SocketEndpoint{
	Network: "tcp6",
	Address: "[::1]:0", // port 0 lets the OS choose any available port
	Release: func(t *testing.T) {},
}

var SocketUnixStream = SocketEndpoint{
	Network: "unix",
	Address: socketTestFilePath,
	Release: func(t *testing.T) {
		err := os.Remove(socketTestFilePath)
		if err != nil {
			t.Fatalf("failed to remove socket file path %q: %v", socketTestFilePath, err)
		}
	},
}

var SocketUnixPacket = SocketEndpoint{
	Network: "unixpacket",
	Address: socketTestFilePath,
	Release: func(t *testing.T) {
		err := os.Remove(socketTestFilePath)
		if err != nil {
			t.Fatalf("failed to remove socket file path %q: %v", socketTestFilePath, err)
		}
	},
}
