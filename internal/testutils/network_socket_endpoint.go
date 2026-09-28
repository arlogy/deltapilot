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
