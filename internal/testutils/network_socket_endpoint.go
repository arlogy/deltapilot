package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arlogy/deltapilot/internal/filesystem"
)

type SocketEndpoint struct {
	Network string
	Address string
	Release func(t *testing.T) // releases any resource that may have been acquired for this endpoint
}

var socketTestFilePath = filepath.Join(os.TempDir(), "deltapilot_socket_test.sock")

var SocketTCPIPAny = SocketEndpoint{
	Network: "tcp",
	Address: "127.0.0.1:0", // see dynamic_port below
	Release: func(t *testing.T) {},
}

var SocketTCPIPv4 = SocketEndpoint{
	Network: "tcp4",
	Address: "127.0.0.1:0", // see dynamic_port below
	Release: func(t *testing.T) {},
}

var SocketTCPIPv6 = SocketEndpoint{
	Network: "tcp6",
	Address: "[::1]:0", // see dynamic_port below
	Release: func(t *testing.T) {},
}

var SocketUnixStream = SocketEndpoint{
	Network: "unix",
	Address: socketTestFilePath,
	Release: func(t *testing.T) {
		if filesystem.ExistsFile(socketTestFilePath) {
			err := os.Remove(socketTestFilePath)
			if err != nil {
				t.Fatalf("failed to remove socket file path %q: %v", socketTestFilePath, err)
			}
		}
	},
}

var SocketUnixPacket = SocketEndpoint{
	Network: "unixpacket",
	Address: socketTestFilePath,
	Release: func(t *testing.T) {
		if filesystem.ExistsFile(socketTestFilePath) {
			err := os.Remove(socketTestFilePath)
			if err != nil {
				t.Fatalf("failed to remove socket file path %q: %v", socketTestFilePath, err)
			}
		}
	},
}

// (dynamic_port)
//     port 0 lets the OS choose any available port during server setup: listener, err := net.Listen()
//     listener.Addr().String() returns the listener address, which can be used to connect to the server
