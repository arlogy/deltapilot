package transport_test

import (
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/netkit/transport"
)

func TestReadFromConnection(t *testing.T) {
	t.Run("fails if maxMsgBytes < 0", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			maxBytes := []int{-25}
			timeouts := []time.Duration{-time.Millisecond, 0, time.Millisecond}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					clientConn := testutils.ConnectToSocketServer(t, network, address)
					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, "max bytes per message must be non-negative").
						Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					testutils.ConnectToSocketServer(t, network, address)

					serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
					msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, "max bytes per message must be non-negative").
						Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails if maxMsgBytes > MaxSupportedMsgBytes", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			maxBytes := []int{transport.MaxSupportedMsgBytes + 1}
			timeouts := []time.Duration{-time.Millisecond, 0, time.Millisecond}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					clientConn := testutils.ConnectToSocketServer(t, network, address)
					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"max bytes per message must be lower than %d", transport.MaxSupportedMsgBytes,
					)).Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					testutils.ConnectToSocketServer(t, network, address)

					serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
					msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"max bytes per message must be lower than %d", transport.MaxSupportedMsgBytes,
					)).Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails if readTimeout < 0 (and maxMsgBytes is valid)", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			maxBytes := []int{0, 25}
			timeouts := []time.Duration{-time.Millisecond}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					clientConn := testutils.ConnectToSocketServer(t, network, address)
					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, "read timeout must be non-negative").Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					testutils.ConnectToSocketServer(t, network, address)

					serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
					msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, "read timeout must be non-negative").Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails when too much data is read (within the 2-byte detection range)", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			func() {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte("a"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, 0, time.Millisecond)
				testutils.AssertErrorMessage(
					t, err, "received message exceeds the allowed 0-byte size limit",
				).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				err = transport.WriteToConnection(serverConn, []byte("a"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, 0, time.Millisecond)
				testutils.AssertErrorMessage(
					t, err, "received message exceeds the allowed 0-byte size limit",
				).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}()

			func() {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte("ab"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, 1, time.Millisecond)
				testutils.AssertErrorMessage(
					t, err, "received message exceeds the allowed 1-byte size limit",
				).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				err = transport.WriteToConnection(serverConn, []byte("ab"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, 1, time.Millisecond)
				testutils.AssertErrorMessage(
					t, err, "received message exceeds the allowed 1-byte size limit",
				).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}()

			func() {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte("😀"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, 3, time.Millisecond)
				testutils.AssertErrorMessage(
					t, err, "received message exceeds the allowed 3-byte size limit",
				).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				err = transport.WriteToConnection(serverConn, []byte("😀"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, 3, time.Millisecond)
				testutils.AssertErrorMessage(
					t, err, "received message exceeds the allowed 3-byte size limit",
				).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}()
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails when too much data is read (outside the 2-byte detection range)", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, inputStr := range []string{testutils.GenerateAlphanumeric(t), "😀"} {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, 0, time.Millisecond)
				testutils.AssertErrorMessage(t, err, "connection closed without end marker").Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				err = transport.WriteToConnection(serverConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, 0, time.Millisecond)
				testutils.AssertErrorMessage(t, err, "connection closed without end marker").Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}

			for _, inputStr := range []string{testutils.GenerateAlphanumeric(t), "😀a"} {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, 3, time.Millisecond)
				testutils.AssertErrorMessage(t, err, "connection closed without end marker").Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				err = transport.WriteToConnection(serverConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, 3, time.Millisecond)
				testutils.AssertErrorMessage(t, err, "connection closed without end marker").Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails with a timeout when read timeout is 0", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, maxMsgBytes := range []int{0, 1, 10} {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte("a"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, 0)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				err = transport.WriteToConnection(serverConn, []byte("a"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, maxMsgBytes, 0)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, clientConn.LocalAddr(), address,
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails with a timeout when no data is received in time", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, maxMsgBytes := range []int{0, 1, 10} {
				clientConn := testutils.ConnectToSocketServer(t, network, address)

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, maxMsgBytes, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, clientConn.LocalAddr(), address,
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("fails with a timeout when MessageEndByte is not read in time", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			// note: transport.WriteToConnection() would have written transport.MessageEndByte too, contrary
			//       to *.Write()
			for _, maxMsgBytes := range []int{0, 1, 10} {
				clientConn := testutils.ConnectToSocketServer(t, network, address)

				_, err := clientConn.Write([]byte("a"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				_, err = serverConn.Write([]byte("a"))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, maxMsgBytes, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, clientConn.LocalAddr(), address,
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("reads available data when below the maximum allowed size", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, inputStr := range []string{"", testutils.GenerateAlphanumeric(t), "😀"} {
				inputLen := len(inputStr)

				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, inputLen+1, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				testutils.AssertEqual(t, msgData, []byte(inputStr)).Critical()

				err = transport.WriteToConnection(serverConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, inputLen+1, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				testutils.AssertEqual(t, msgData, []byte(inputStr)).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("reads available data when at the maximum allowed size", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, inputStr := range []string{"", testutils.GenerateAlphanumeric(t), "😀"} {
				inputLen := len(inputStr)

				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				msgData, err := transport.ReadFromConnection(serverConn, inputLen, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				testutils.AssertEqual(t, msgData, []byte(inputStr)).Critical()

				err = transport.WriteToConnection(serverConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err = transport.ReadFromConnection(clientConn, inputLen, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				testutils.AssertEqual(t, msgData, []byte(inputStr)).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})
}

func TestWriteFromConnection(t *testing.T) {
	t.Run("fails if the connection is not writable", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, inputStr := range []string{"", testutils.GenerateAlphanumeric(t), "😀"} {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				clientConn.Close() // test one failure scenario among several
				err := transport.WriteToConnection(clientConn, []byte(inputStr))
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"write %s %s->%s: use of closed network connection",
					network, clientConn.LocalAddr(), address,
				)).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				serverConn.SetReadDeadline(time.Now().Add(time.Millisecond))
				serverConn.Close() // test one failure scenario among several
				err = transport.WriteToConnection(serverConn, []byte(inputStr))
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"write %s %s->%s: use of closed network connection",
					network, address, serverConn.RemoteAddr(),
				)).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})

	t.Run("writes input data followed by MessageEndByte", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			for _, inputStr := range []string{"", testutils.GenerateAlphanumeric(t), "😀"} {
				inputLen := len(inputStr)

				clientConn := testutils.ConnectToSocketServer(t, network, address)
				err := transport.WriteToConnection(clientConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				serverConn.SetReadDeadline(time.Now().Add(time.Millisecond))
				func() {
					buf := make([]byte, inputLen+1)
					n, err := io.ReadFull(serverConn, buf)
					msgData := append([]byte(inputStr), transport.MessageEndByte)
					testutils.AssertEqual(t, err, nil).Critical()
					testutils.AssertEqual(t, n, inputLen+1).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()
				func() {
					buf := make([]byte, 1)
					n, err := io.ReadFull(serverConn, buf)
					msgData := make([]byte, 1)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
					)).Critical()
					testutils.AssertEqual(t, n, 0).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()

				err = transport.WriteToConnection(serverConn, []byte(inputStr))
				testutils.AssertErrorIs(t, err, nil).Critical()

				clientConn.SetReadDeadline(time.Now().Add(time.Millisecond))
				func() {
					buf := make([]byte, inputLen+1)
					n, err := io.ReadFull(clientConn, buf)
					msgData := append([]byte(inputStr), transport.MessageEndByte)
					testutils.AssertEqual(t, err, nil).Critical()
					testutils.AssertEqual(t, n, inputLen+1).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()
				func() {
					buf := make([]byte, 1)
					n, err := io.ReadFull(clientConn, buf)
					msgData := make([]byte, 1)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"read %s %s->%s: i/o timeout", network, clientConn.LocalAddr(), address,
					)).Critical()
					testutils.AssertEqual(t, n, 0).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testutils.SocketTCPIPAny)
		check(testutils.SocketTCPIPv4)
		check(testutils.SocketTCPIPv6)
		check(testutils.SocketUnixStream)
		if runtime.GOOS != "windows" {
			check(testutils.SocketUnixPacket) // not supported on excluded platforms
		}
	})
}
