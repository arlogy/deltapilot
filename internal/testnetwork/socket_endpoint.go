package testnetwork

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arlogy/deltapilot/internal/filesystem"
	"github.com/arlogy/deltapilot/internal/testutils"
)

type SocketEndpoint struct {
	Network string
	Address string
	Release func() // releases any resource that may have been acquired for this endpoint
}

var SocketTCPIPAny = SocketEndpoint{
	Network: "tcp",
	Address: "127.0.0.1:0", // see dynamic_port below
	Release: func() {},
}

var SocketTCPIPv4 = SocketEndpoint{
	Network: "tcp4",
	Address: "127.0.0.1:0", // see dynamic_port below
	Release: func() {},
}

var SocketTCPIPv6 = SocketEndpoint{
	Network: "tcp6",
	Address: "[::1]:0", // see dynamic_port below
	Release: func() {},
}

func SocketUnixStream(t *testing.T) SocketEndpoint {
	return socketUnixEndpoint(t, "unix")
}

func SocketUnixPacket(t *testing.T) SocketEndpoint {
	return socketUnixEndpoint(t, "unixpacket")
}

func socketUnixEndpoint(t *testing.T, network string) SocketEndpoint {
	socketFileName := "deltapilot_test_socket_" + testutils.GenerateAlphanumeric(t) + ".sock"
	socketFilePath := filepath.Join(os.TempDir(), socketFileName)

	return SocketEndpoint{
		Network: network,
		Address: socketFilePath,
		Release: func() {
			// on Windows, the socket file is removed automatically
			// remove it explicitly only if it still exists
			if filesystem.ExistsFile(socketFilePath) {
				err := os.Remove(socketFilePath)
				if err != nil {
					t.Fatalf("failed to remove socket file path %q: %v", socketFilePath, err)
				}
			}
		},
	}
}

// (dynamic_port)
//     port 0 lets the OS choose any available port during server setup: listener, err := net.Listen()
//     listener.Addr().String() returns the listener address, which can be used to connect to the server
