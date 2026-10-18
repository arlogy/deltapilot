package protocol_test

import (
	"bytes"
	"fmt"
	"math/rand/v2"
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
	wantErrMsg string,
) {
	t.Helper()

	msgData, err := protocol.ReadFromConnection(conn, maxMsgBytes, readTimeout)
	testutils.AssertErrorMessage(t, err, wantErrMsg).Critical()
	testutils.AssertEqual(t, msgData == nil, true).Critical()
}

func readMessageErrorAfterRead(
	t *testing.T,
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
	wantErrMsg string,
	wantData []byte,
) {
	t.Helper()

	msgData, err := protocol.ReadFromConnection(conn, maxMsgBytes, readTimeout)
	testutils.AssertErrorMessage(t, err, wantErrMsg).Critical()
	testutils.AssertEqual(t, msgData, wantData).Critical()
}

func writeMessagePass(t *testing.T, conn net.Conn, data []byte) {
	t.Helper()

	err := protocol.WriteToConnection(conn, data)
	testutils.AssertErrorIs(t, err, nil).Critical()
}

func writeMessageFail(t *testing.T, conn net.Conn, data []byte, wantErrMsg string) {
	t.Helper()

	err := protocol.WriteToConnection(conn, data)
	testutils.AssertErrorMessage(t, err, wantErrMsg).Critical()
}

func peerExchanges(clientConn net.Conn, serverConn net.Conn) []ioExchange {
	clientPeer := &ioPeer{baseConn: clientConn, isServer: false}
	serverPeer := &ioPeer{baseConn: serverConn, isServer: true}

	return []ioExchange{
		{sender: clientPeer, receiver: serverPeer},
		{sender: serverPeer, receiver: clientPeer},
	}
}

func placeByteRandomly(t *testing.T, data []byte, b byte, count int) {
	t.Helper()

	n := len(data)

	if count < 0 || count > n {
		t.Fatal("failed to place byte randomly: repetition count is out of range")
	}

	for _, i := range rand.Perm(n)[:count] {
		data[i] = b
	}
}

func prefixBeforeMatch(data []byte, b byte) []byte {
	result := data
	if i := bytes.IndexByte(result, b); i >= 0 {
		result = result[:i]
	}

	return bytes.Clone(result)
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
			wantErrMsg := "max bytes per message must be non-negative"

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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg)

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
			wantErrMsg := fmt.Sprintf(
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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg)

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
			wantErrMsg := "read timeout must be non-negative"

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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg)

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
			wantErrMsg := func(targetErr error) string {
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
							readMessageFail(t, recvStub, maxMsgBytes, readTimeout, wantErrMsg(randErr))

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
			wantErrMsg := func(
				conn net.Conn, network string, address string, serverSide bool, targetErr error,
			) string {
				readErrMsg := "message read failed: "
				readErrMsg += testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
				return fmt.Sprintf("%s\nfailed to unset read deadline: %v", readErrMsg, targetErr)
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
							readMessageFail(t, recvStub, maxMsgBytes, readTimeout, wantErrMsg(
								recvStub, network, address, recvServer, randErr,
							))

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
			wantErrMsg := func(targetErr error) string {
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
							readMessageErrorAfterRead(
								t, recvStub, maxMsgBytes, readTimeout, wantErrMsg(randErr), wireData,
							)

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
			wantErrMsg := func(conn net.Conn, network string, address string, serverSide bool) string {
				return "message read failed: " +
					testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, maxMsgBytes := range maxMsgByteValues {
					for _, readTimeout := range timeouts {
						for _, wireData := range wireInputs {
							sendConn := exchange.sender.baseConn
							recvConn := exchange.receiver.baseConn
							recvServer := exchange.receiver.isServer

							writeMessagePass(t, sendConn, wireData)

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg(
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
			wantErrMsg := func(maxMsgBytes int) string {
				return fmt.Sprintf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg(maxMsgBytes))

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
			wantErrMsg := func(maxMsgBytes int) string {
				return fmt.Sprintf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg(maxMsgBytes))

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
			wantErrMsg := "message end marker not found"

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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg)

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
			wantErrMsg := func(maxMsgBytes int) string {
				return fmt.Sprintf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg(maxMsgBytes))

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
			wantErrMsg := func(maxMsgBytes int) string {
				return fmt.Sprintf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg(maxMsgBytes))

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
			wantErrMsg := "message end marker not found"

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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg)

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
			wantErrMsg := func(conn net.Conn, network string, address string, serverSide bool) string {
				return "message read failed: " +
					testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
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

							readMessageFail(t, recvConn, maxMsgBytes, readTimeout, wantErrMsg(
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
			wantErrMsg := func(conn net.Conn, network string, address string, serverSide bool) string {
				return "message read failed: " +
					testnetwork.FormatBytesReadTimeout(conn, network, address, serverSide)
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

						readMessageFail(t, recvConn, max(maxMsgBytes, 0), readTimeout, wantErrMsg(
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
								placeByteRandomly(t, wireData, protocol.MessageEndByte, markerCount)
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
								readMessagePass(t, recvConn, maxMsgBytes, readTimeout, prefixBeforeMatch(
									wireData, protocol.MessageEndByte,
								))
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
			wantErrMsg := func(targetErr error) string {
				return fmt.Sprintf("data write failed: %v", targetErr)
			}

			for _, exchange := range peerExchanges(clientConn, serverConn) {
				for _, wireData := range wireInputs {
					recvConn := exchange.receiver.baseConn

					randErr := testutils.GenerateError(t)
					recvStub := testnetwork.NewConnStubFailingToWrite(t, recvConn, randErr)
					writeMessageFail(t, recvStub, wireData, wantErrMsg(randErr))
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
