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

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			maxBytes := []int{-25}
			timeouts := []time.Duration{-time.Millisecond, 0, time.Millisecond}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, "max bytes per message must be non-negative").
						Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
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

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			maxBytes := []int{transport.MaxSupportedMsgBytes + 1}
			timeouts := []time.Duration{-time.Millisecond, 0, time.Millisecond}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"max bytes per message must be lower than %d", transport.MaxSupportedMsgBytes,
					)).Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
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

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			maxBytes := []int{0, 25}
			timeouts := []time.Duration{-time.Millisecond}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, readTimeout)
					testutils.AssertErrorMessage(t, err, "read timeout must be non-negative").Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, maxMsgBytes := range maxBytes {
				for _, readTimeout := range timeouts {
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

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := []string{"a", "ab", testutils.GenerateAlphanumeric(t), "😀"}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(clientConn, []byte(inputVal))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(serverConn, inputLen-1, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"received message exceeds the allowed %d-byte size limit", inputLen-1,
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(serverConn, []byte(inputVal))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(clientConn, inputLen-1, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"received message exceeds the allowed %d-byte size limit", inputLen-1,
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

	t.Run("fails when too much data is read (outside the 2-byte detection range)", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := []string{"ab", testutils.GenerateAlphanumeric(t), "😀"}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(clientConn, []byte(inputVal))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(serverConn, inputLen-2, time.Millisecond)
				testutils.AssertErrorMessage(t, err, "connection closed without end marker").Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				restData, err := io.ReadAll(serverConn) // read remaining bytes
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
				)).Critical()
				testutils.AssertEqual(t, restData, []byte{transport.MessageEndByte}).Critical()
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(serverConn, []byte(inputVal))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(clientConn, inputLen-2, time.Millisecond)
				testutils.AssertErrorMessage(t, err, "connection closed without end marker").Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()

				restData, err := io.ReadAll(clientConn) // read remaining bytes
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, clientConn.LocalAddr(), address,
				)).Critical()
				testutils.AssertEqual(t, restData, []byte{transport.MessageEndByte}).Critical()
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

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := [][]byte{
				nil, []byte(""), []byte(testutils.GenerateAlphanumeric(t)), []byte("😀"),
			}
			maxBytes := []int{0, 10}

			for _, inputVal := range inputEntries {
				for _, maxMsgBytes := range maxBytes {
					err := transport.WriteToConnection(clientConn, inputVal)
					testutils.AssertErrorIs(t, err, nil).Critical()

					msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, 0)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
					)).Critical()
					testutils.AssertEqual(t, msgData == nil, true).Critical()
				}
			}

			for _, inputVal := range inputEntries {
				for _, maxMsgBytes := range maxBytes {
					err := transport.WriteToConnection(serverConn, inputVal)
					testutils.AssertErrorIs(t, err, nil).Critical()

					msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, 0)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"read %s %s->%s: i/o timeout", network, clientConn.LocalAddr(), address,
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

	t.Run("fails with a timeout when no data is received in time", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			maxBytes := []int{0, 10}

			for _, maxMsgBytes := range maxBytes {
				msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}

			for _, maxMsgBytes := range maxBytes {
				msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, time.Millisecond)
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

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := [][]byte{
				nil, []byte(""), []byte(testutils.GenerateAlphanumeric(t)), []byte("😀"),
			}
			maxMsgBytes := transport.MaxSupportedMsgBytes

			// note: *.Write() does not automatically write transport.MessageEndByte, unlike
			//       transport.WriteToConnection()

			for _, inputVal := range inputEntries {
				_, err := clientConn.Write(inputVal)
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(serverConn, maxMsgBytes, time.Millisecond)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
				)).Critical()
				testutils.AssertEqual(t, msgData == nil, true).Critical()
			}

			for _, inputVal := range inputEntries {
				_, err := serverConn.Write(inputVal)
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(clientConn, maxMsgBytes, time.Millisecond)
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

	t.Run("reads available data when size is below the maximum expected", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := [][]byte{
				nil, []byte(""), []byte(testutils.GenerateAlphanumeric(t)), []byte("😀"),
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(clientConn, inputVal)
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(serverConn, inputLen+1, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				if inputVal == nil {
					testutils.AssertEqual(t, msgData, []byte{}).Critical()
				} else {
					testutils.AssertEqual(t, msgData, inputVal).Critical()
				}
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(serverConn, inputVal)
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(clientConn, inputLen+1, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				if inputVal == nil {
					testutils.AssertEqual(t, msgData, []byte{}).Critical()
				} else {
					testutils.AssertEqual(t, msgData, inputVal).Critical()
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

	t.Run("reads available data when size is at the maximum expected", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := [][]byte{
				nil, []byte(""), []byte(testutils.GenerateAlphanumeric(t)), []byte("😀"),
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(clientConn, []byte(inputVal))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(serverConn, inputLen, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				if inputVal == nil {
					testutils.AssertEqual(t, msgData, []byte{}).Critical()
				} else {
					testutils.AssertEqual(t, msgData, inputVal).Critical()
				}
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(serverConn, []byte(inputVal))
				testutils.AssertErrorIs(t, err, nil).Critical()

				msgData, err := transport.ReadFromConnection(clientConn, inputLen, time.Millisecond)
				testutils.AssertEqual(t, err, nil).Critical()
				if inputVal == nil {
					testutils.AssertEqual(t, msgData, []byte{}).Critical()
				} else {
					testutils.AssertEqual(t, msgData, inputVal).Critical()
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
}

func TestWriteFromConnection(t *testing.T) {
	t.Run("fails if the connection is not writable", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			inputEntries := [][]byte{
				nil, []byte(""), []byte(testutils.GenerateAlphanumeric(t)), []byte("😀"),
			}

			for _, inputVal := range inputEntries {
				clientConn := testutils.ConnectToSocketServer(t, network, address)
				clientConn.Close() // test one failure scenario among several
				err := transport.WriteToConnection(clientConn, inputVal)
				testutils.AssertErrorMessage(t, err, fmt.Sprintf(
					"write %s %s->%s: use of closed network connection",
					network, clientConn.LocalAddr(), address,
				)).Critical()

				serverConn := testutils.WaitForConnectionToSocketServer(t, listener)
				serverConn.SetReadDeadline(time.Now().Add(time.Millisecond))
				serverConn.Close() // test one failure scenario among several
				err = transport.WriteToConnection(serverConn, inputVal)
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

	t.Run("writes input data followed by MessageEndByte otherwise", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			clientConn := testutils.ConnectToSocketServer(t, network, address)
			serverConn := testutils.WaitForConnectionToSocketServer(t, listener)

			inputEntries := [][]byte{
				nil, []byte(""), []byte(testutils.GenerateAlphanumeric(t)), []byte("😀"),
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(clientConn, inputVal)
				testutils.AssertErrorIs(t, err, nil).Critical()

				serverConn.SetReadDeadline(time.Now().Add(time.Millisecond))
				func() {
					// read expected bytes
					buf := make([]byte, inputLen+1)
					n, err := io.ReadFull(serverConn, buf)
					msgData := append(inputVal, transport.MessageEndByte)
					testutils.AssertEqual(t, err, nil).Critical()
					testutils.AssertEqual(t, n, inputLen+1).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()
				func() {
					// ensure no more bytes are available
					buf := make([]byte, 1)
					n, err := io.ReadFull(serverConn, buf)
					msgData := make([]byte, 1)
					testutils.AssertErrorMessage(t, err, fmt.Sprintf(
						"read %s %s->%s: i/o timeout", network, address, serverConn.RemoteAddr(),
					)).Critical()
					testutils.AssertEqual(t, n, 0).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()
			}

			for _, inputVal := range inputEntries {
				inputLen := len(inputVal)

				err := transport.WriteToConnection(serverConn, inputVal)
				testutils.AssertErrorIs(t, err, nil).Critical()

				clientConn.SetReadDeadline(time.Now().Add(time.Millisecond))
				func() {
					// read expected bytes
					buf := make([]byte, inputLen+1)
					n, err := io.ReadFull(clientConn, buf)
					msgData := append(inputVal, transport.MessageEndByte)
					testutils.AssertEqual(t, err, nil).Critical()
					testutils.AssertEqual(t, n, inputLen+1).Critical()
					testutils.AssertEqual(t, buf, msgData).Critical()
				}()
				func() {
					// ensure no more bytes are available
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
