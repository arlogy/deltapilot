package protocol_test

import (
	"bytes"
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/testnetwork"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/netkit/protocol"
)

const readDataTimeout = 5 * time.Millisecond // empirically chosen to avoid flaky tests

// used to acquire available data; it always expires, so a short duration is preferable for faster tests
const waitForDataTimeout = 1 * time.Millisecond

type ioPeer struct {
	baseConn net.Conn
	isServer bool
}

type ioExchange struct {
	sender   *ioPeer
	receiver *ioPeer
}

func readMessagePass(
	t *testing.T,
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
	wantData []byte,
) {
	t.Helper()

	msgData, err := protocol.ReadFromConnection(conn, maxMsgBytes, readTimeout)
	testutils.AssertErrorIs(t, err, nil).Critical()
	testutils.AssertEqual(t, msgData, wantData).Critical()
}

func readMessageFail(
	t *testing.T,
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
	wantErr error,
	wantFailure string,
) {
	t.Helper()

	msgData, gotErr := protocol.ReadFromConnection(conn, maxMsgBytes, readTimeout)

	testutils.AssertErrorIs(t, gotErr, wantErr).Critical()
	testutils.AssertErrorMessage(t, gotErr, wantErr.Error()+": "+wantFailure).Critical()

	testutils.AssertEqual(t, msgData == nil, true).Critical()
}

func readMessageDoubleFail(
	t *testing.T,
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
	wantErrs [2]error,
	wantFailures [2]string,
) {
	t.Helper()

	msgData, gotErr := protocol.ReadFromConnection(conn, maxMsgBytes, readTimeout)

	for _, wantErr := range wantErrs {
		testutils.AssertErrorIs(t, gotErr, wantErr).Critical()
	}

	testutils.AssertErrorMessage(
		t, gotErr, wantErrs[0].Error()+": "+wantFailures[0]+"\n"+wantErrs[1].Error()+": "+wantFailures[1],
	).Critical()

	testutils.AssertEqual(t, msgData == nil, true).Critical()
}

func writeMessagePass(t *testing.T, conn net.Conn, data []byte) {
	t.Helper()

	err := protocol.WriteToConnection(conn, data)
	testutils.AssertErrorIs(t, err, nil).Critical()
}

func writeMessageFail(t *testing.T, conn net.Conn, data []byte, wantErr error, wantFailure string) {
	t.Helper()

	gotErr := protocol.WriteToConnection(conn, data)
	testutils.AssertErrorIs(t, gotErr, wantErr).Critical()
	testutils.AssertErrorMessage(t, gotErr, wantErr.Error()+" "+wantFailure).Critical()
}

func peerExchanges(clientConn net.Conn, serverConn net.Conn) []ioExchange {
	clientPeer := &ioPeer{baseConn: clientConn, isServer: false}
	serverPeer := &ioPeer{baseConn: serverConn, isServer: true}

	return []ioExchange{
		{sender: clientPeer, receiver: serverPeer},
		{sender: serverPeer, receiver: clientPeer},
	}
}

func TestReadFromConnection(t *testing.T) {
	t.Run("fails if maxMsgBytes < 0", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{-25}
			timeouts := []time.Duration{-time.Millisecond, 0, time.Millisecond}
			useBaseData := []bool{false, true}
			wantErr := protocol.ErrInvalidArgument
			wantFailure := "max bytes per message must be non-negative"

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, includeBaseData := range useBaseData {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if includeBaseData {
								wireData = testnetwork.GenerateNetProtocolMessage(t)
								testnetwork.ExpectBytesWritten(t, sendConn, wireData)
							}

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)

							if includeBaseData {
								testnetwork.ExpectBytesAvailable(
									t, recvConn, network, address, recvServer, waitForDataTimeout, wireData,
								)
							}
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if maxMsgBytes > MaxReadMessageBytes", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{protocol.MaxReadMessageBytes + 1}
			timeouts := []time.Duration{-time.Millisecond, 0, time.Millisecond}
			useBaseData := []bool{false, true}
			wantErr := protocol.ErrInvalidArgument
			wantFailure := fmt.Sprintf(
				"max bytes per message must be at most %d", protocol.MaxReadMessageBytes,
			)

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, includeBaseData := range useBaseData {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if includeBaseData {
								wireData = testnetwork.GenerateNetProtocolMessage(t)
								testnetwork.ExpectBytesWritten(t, sendConn, wireData)
							}

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)

							if includeBaseData {
								testnetwork.ExpectBytesAvailable(
									t, recvConn, network, address, recvServer, waitForDataTimeout, wireData,
								)
							}
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if readTimeout < 0 (and maxMsgBytes is valid)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25, protocol.MaxReadMessageBytes}
			timeouts := []time.Duration{-time.Millisecond}
			useBaseData := []bool{false, true}
			wantErr := protocol.ErrInvalidArgument
			wantFailure := "read timeout must be non-negative"

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, includeBaseData := range useBaseData {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if includeBaseData {
								wireData = testnetwork.GenerateNetProtocolMessage(t)
								testnetwork.ExpectBytesWritten(t, sendConn, wireData)
							}

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)

							if includeBaseData {
								testnetwork.ExpectBytesAvailable(
									t, recvConn, network, address, recvServer, waitForDataTimeout, wireData,
								)
							}
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if a valid readTimeout cannot be applied", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{0, readDataTimeout}
			useBaseData := []bool{false, true}
			wantErr := protocol.ErrReadFailed
			wantFailure := func(targetErr error) string {
				return fmt.Sprintf("failed to set read deadline: %v", targetErr)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, includeBaseData := range useBaseData {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if includeBaseData {
								wireData = testnetwork.GenerateNetProtocolMessage(t)
								testnetwork.ExpectBytesWritten(t, sendConn, wireData)
							}

							randErr := testutils.GenerateError(t)
							recvStub := testnetwork.NewConnStubFailingToApplyReadDeadline(
								t, recvConn, readTimeout, randErr,
							)
							readMessageFail(t, recvStub, maxMsgBytes, readTimeout, wantErr, wantFailure(
								randErr,
							))

							if includeBaseData {
								testnetwork.ExpectBytesAvailable(
									t, recvConn, network, address, recvServer, waitForDataTimeout, wireData,
								)
							}
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if applied readTimeout cannot be cleared after a read error", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{0, readDataTimeout}
			useBaseData := []bool{false, true}
			wantErrs := [2]error{protocol.ErrTimeout, protocol.ErrReadFailed}
			wantFailures := func(
				conn net.Conn, network string, address string, serverSide bool, targetErr error,
			) [2]string {
				return [2]string{
					testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide),
					fmt.Sprintf("failed to unset read deadline: %v", targetErr),
				}
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, includeBaseData := range useBaseData {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if includeBaseData {
								wireData = testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)
								testnetwork.ExpectBytesWritten(t, sendConn, wireData)
							}

							randErr := testutils.GenerateError(t)
							recvStub := testnetwork.NewConnStubFailingToCancelReadDeadline(
								t, recvConn, readTimeout, randErr,
							)
							readMessageDoubleFail(
								t, recvStub, maxMsgBytes, readTimeout, wantErrs, wantFailures(
									recvStub, network, address, recvServer, randErr,
								),
							)

							if includeBaseData {
								restData := []byte{}
								if readTimeout == 0 {
									restData = wireData
								}
								testnetwork.ExpectBytesAvailable(
									t, recvConn, network, address, recvServer, waitForDataTimeout, restData,
								)
							}
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if applied readTimeout cannot be cleared after a successful read", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			useBaseData := []bool{true}
			wantErr := protocol.ErrReadFailed
			wantFailure := func(targetErr error) string {
				return fmt.Sprintf("failed to unset read deadline: %v", targetErr)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, includeBaseData := range useBaseData {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if includeBaseData {
								wireData = testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)
								writeMessagePass(t, sendConn, wireData)
							}

							randErr := testutils.GenerateError(t)
							recvStub := testnetwork.NewConnStubFailingToCancelReadDeadline(
								t, recvConn, readTimeout, randErr,
							)
							readMessageFail(t, recvStub, maxMsgBytes, readTimeout, wantErr, wantFailure(
								randErr,
							))

							if includeBaseData {
								testnetwork.ExpectBytesAvailable(
									t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
								)
							}
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if applied readTimeout is 0", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{0}
			wireInputs := [][]byte{nil, []byte(""), testnetwork.GenerateNetProtocolMessage(t)}
			wantErr := protocol.ErrTimeout
			wantFailure := func(conn net.Conn, network string, address string, serverSide bool) string {
				return testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, wireData := range wireInputs {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							writeMessagePass(t, sendConn, wireData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure(
								recvConn, network, address, recvServer,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, append(
									bytes.Clone(wireData), protocol.MessageEndByte,
								),
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails if data cannot be read from the connection", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			wireInputs := [][]byte{nil, []byte(""), testnetwork.GenerateNetProtocolMessage(t)}
			wantErr := protocol.ErrReadFailed
			wantFailure := func(targetErr error) string {
				return fmt.Sprintf("%v", targetErr)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, wireData := range wireInputs {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							writeMessagePass(t, sendConn, wireData)

							randErr := testutils.GenerateError(t)
							recvStub := testnetwork.NewConnStubFailingToRead(t, recvConn, randErr)
							readMessageFail(t, recvStub, maxMsgBytes, readTimeout, wantErr, wantFailure(
								randErr,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, append(
									bytes.Clone(wireData), protocol.MessageEndByte,
								),
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 3+ bytes (MessageEndByte is 1st)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the 1st byte is the first terminator
				{protocol.MessageEndByte, 'a', 'b'},
				{protocol.MessageEndByte, 'a', protocol.MessageEndByte},
				{protocol.MessageEndByte, protocol.MessageEndByte, 'b'},
				{protocol.MessageEndByte, protocol.MessageEndByte, protocol.MessageEndByte},
				append([]byte{protocol.MessageEndByte}, []byte("😀")...), // longer than 3 bytes
			}
			wantErr := protocol.ErrMessageTooLarge
			wantFailure := func(maxMsgBytes int) string {
				return fmt.Sprintf("exceeds the allowed %d-byte limit", maxMsgBytes)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure(
								maxMsgBytes,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, extraData[2:],
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 3+ bytes (MessageEndByte is 2nd)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the 2nd byte is the first terminator
				{'a', protocol.MessageEndByte, 'c'},
				{'a', protocol.MessageEndByte, protocol.MessageEndByte},
				append([]byte{'a', protocol.MessageEndByte}, []byte("😀")...), // longer than 3 bytes
			}
			wantErr := protocol.ErrMessageTooLarge
			wantFailure := func(maxMsgBytes int) string {
				return fmt.Sprintf("exceeds the allowed %d-byte limit", maxMsgBytes)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure(
								maxMsgBytes,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, extraData[2:],
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 3+ bytes (MessageEndByte elsewhere)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the 3rd byte is the first terminator
				{'a', 'b', protocol.MessageEndByte},
				{'a', 'b', protocol.MessageEndByte, protocol.MessageEndByte},      // longer than 3 bytes
				append([]byte{'a', 'b', protocol.MessageEndByte}, []byte("😀")...), // longer than 3 bytes

				// the 4th or a later byte is the first terminator
				{'a', 'b', 'c', protocol.MessageEndByte},      // longer than 3 bytes
				{'a', 'b', 'c', 'd', protocol.MessageEndByte}, // longer than 3 bytes

				// the terminator is absent
				[]byte("abc"),
				append([]byte{'a', 'b'}, []byte("😀")...), // longer than 3 bytes
			}
			wantErr := protocol.ErrMessageIncomplete
			wantFailure := "end marker not found"

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, extraData[2:],
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 2 bytes (MessageEndByte is 1st)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the 1st byte is the first terminator
				{protocol.MessageEndByte, 'a'},
				{protocol.MessageEndByte, protocol.MessageEndByte},
			}
			wantErr := protocol.ErrMessageTooLarge
			wantFailure := func(maxMsgBytes int) string {
				return fmt.Sprintf("exceeds the allowed %d-byte limit", maxMsgBytes)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure(
								maxMsgBytes,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 2 bytes (MessageEndByte is 2nd)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the 2nd byte is the first terminator
				{'a', protocol.MessageEndByte},
			}
			wantErr := protocol.ErrMessageTooLarge
			wantFailure := func(maxMsgBytes int) string {
				return fmt.Sprintf("exceeds the allowed %d-byte limit", maxMsgBytes)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure(
								maxMsgBytes,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 2 bytes (MessageEndByte elsewhere)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the terminator is absent
				{'a', 'b'},
			}
			wantErr := protocol.ErrMessageIncomplete
			wantFailure := "end marker not found"

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("succeeds when data exceeds maxMsgBytes by 1 byte (MessageEndByte)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the terminator is the only byte
				{protocol.MessageEndByte},
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessagePass(t, recvConn, maxMsgBytes, readTimeout, maxData)

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data exceeds maxMsgBytes by 1 byte (not MessageEndByte)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{0, 25}
			timeouts := []time.Duration{readDataTimeout}
			extraByteEntries := [][]byte{
				// the terminator is absent
				{'a'},
			}
			wantErr := protocol.ErrTimeout
			wantFailure := func(conn net.Conn, network string, address string, serverSide bool) string {
				return testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, extraData := range extraByteEntries {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)

							testnetwork.ExpectBytesWritten(t, sendConn, maxData)
							testnetwork.ExpectBytesWritten(t, sendConn, extraData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure(
								recvConn, network, address, recvServer,
							))

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("fails when data is at most maxMsgBytes long (without MessageEndByte)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{25, 25, 25, -1} // negative value ==> nil wireData
			bytesBelowMax := []int{0, 10, 25, 0}
			timeouts := []time.Duration{readDataTimeout}
			wantErr := protocol.ErrTimeout
			wantFailure := func(conn net.Conn, network string, address string, serverSide bool) string {
				return testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for i, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						sendConn := exchange.sender.baseConn
						recvConn := exchange.receiver.baseConn
						recvServer := exchange.receiver.isServer

						var wireData []byte
						if maxMsgBytes >= 0 {
							wireLen := maxMsgBytes - bytesBelowMax[i]
							wireData = testnetwork.GenerateSizedNetProtocolMessage(t, wireLen)
						}

						testnetwork.ExpectBytesWritten(t, sendConn, wireData)

						readMessageFail(t, recvConn, max(maxMsgBytes, 0), readTimeout, wantErr, wantFailure(
							recvConn, network, address, recvServer,
						))

						testnetwork.ExpectBytesAvailable(
							t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
						)
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("succeeds when data is at most maxMsgBytes long (with 1+ MessageEndByte)", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			// note: maxMsgByteValues and bytesBelowMax are set so that "i >= 4" can be used to add multiple
			//       message end markers for each theoretical scenario (when available data is nil, exactly
			//       maxMsgBytes bytes long, or shorter than maxMsgBytes)
			maxMsgByteValues := []int{25, 25, 25, -1, -1, 18, 18, 17, 17} // negative value ==> nil wireData
			bytesBelowMax := []int{0, 10, 24, 0, 0, 0, 10, 0, 10}
			timeouts := []time.Duration{readDataTimeout}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for i, maxMsgBytes := range maxMsgByteValues {
					endMarkerCounts := []int{1}
					if i >= 4 {
						endMarkerCounts = append(endMarkerCounts, 5)
					}

					for _, markerCount := range endMarkerCounts {
						for _, readTimeout := range timeouts {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							var wireData []byte
							if maxMsgBytes >= 0 {
								wireLen := maxMsgBytes - bytesBelowMax[i]
								wireData = testnetwork.GenerateSizedNetProtocolMessage(t, wireLen)
								testutils.PlaceByteRandomly(t, wireData, protocol.MessageEndByte, markerCount)
							}

							testnetwork.ExpectBytesWritten(t, sendConn, wireData)
							if wireData == nil {
								testnetwork.ExpectBytesWritten(t, sendConn, bytes.Repeat(
									[]byte{protocol.MessageEndByte}, markerCount,
								))
							}

							if wireData == nil {
								readMessagePass(t, recvConn, markerCount, readTimeout, []byte{})
							} else {
								wantData := testutils.PrefixBeforeByte(wireData, protocol.MessageEndByte)
								readMessagePass(t, recvConn, maxMsgBytes, readTimeout, wantData)
							}

							testnetwork.ExpectBytesAvailable(
								t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
							)
						}
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})
}

func TestReadFromConnectionConsecutiveReads(t *testing.T) {
	t.Run("allows consecutive reads from a long byte stream to eventually succeed", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			maxMsgByteValues := []int{25}
			timeouts := []time.Duration{readDataTimeout}
			wantErr := protocol.ErrMessageIncomplete
			wantFailure := "end marker not found"

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						sendConn := exchange.sender.baseConn
						recvConn := exchange.receiver.baseConn
						recvServer := exchange.receiver.isServer

						maxData := testnetwork.GenerateSizedNetProtocolMessage(t, maxMsgBytes)
						extraData := testnetwork.GenerateSizedNetProtocolMessage(t, 2)

						testnetwork.ExpectBytesWritten(t, sendConn, maxData)
						testnetwork.ExpectBytesWritten(t, sendConn, extraData)
						testnetwork.ExpectBytesWritten(t, sendConn, maxData)
						testnetwork.ExpectBytesWritten(t, sendConn, extraData)
						writeMessagePass(t, sendConn, maxData)

						readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)
						readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErr, wantFailure)
						readMessagePass(t, recvConn, maxMsgBytes, readTimeout, maxData)

						testnetwork.ExpectBytesAvailable(
							t, recvConn, network, address, recvServer, waitForDataTimeout, []byte{},
						)
					}
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})
}

func TestWriteFromConnection(t *testing.T) {
	t.Run("fails if the connection is not writable", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			wireInputs := [][]byte{nil, []byte(""), testnetwork.GenerateNetProtocolMessage(t)}
			wantErr := protocol.ErrWriteFailed
			wantFailure := func(targetErr error) string {
				return fmt.Sprintf("for data: %v", targetErr)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, wireData := range wireInputs {
					recvConn := exchange.receiver.baseConn

					randErr := testutils.GenerateError(t)
					recvStub := testnetwork.NewConnStubFailingToWrite(t, recvConn, randErr)
					writeMessageFail(t, recvStub, wireData, wantErr, wantFailure(randErr))
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})

	t.Run("writes input data followed by MessageEndByte otherwise", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			clientConn := testnetwork.ConnectToSocketServer(t, network, address)
			defer clientConn.Close()

			serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
			defer serverConn.Close()

			wireInputs := [][]byte{nil, []byte(""), testnetwork.GenerateNetProtocolMessage(t)}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, wireData := range wireInputs {
					sendConn := exchange.sender.baseConn
					recvConn := exchange.receiver.baseConn
					recvServer := exchange.receiver.isServer

					writeMessagePass(t, sendConn, wireData)

					testnetwork.ExpectBytesAvailable(
						t, recvConn, network, address, recvServer, waitForDataTimeout, append(
							bytes.Clone(wireData), protocol.MessageEndByte,
						),
					)
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(testnetwork.SocketTCPIPAny)
		check(testnetwork.SocketTCPIPv4)
		check(testnetwork.SocketTCPIPv6)
		check(testnetwork.SocketUnixStream(t))
		if runtime.GOOS != "windows" {
			check(testnetwork.SocketUnixPacket(t)) // not supported on excluded platforms
		}
	})
}
