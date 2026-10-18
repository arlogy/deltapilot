package transport_test

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/channels"
	"github.com/arlogy/deltapilot/internal/testnetwork"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/netkit/protocol"
	"github.com/arlogy/deltapilot/netkit/transport"
)

var formatInfo = testnetwork.NetFormatInfo
var formatError = testnetwork.NetFormatError

var unexpectedScenarioPlaceholder = "<no expected scenario was matched>"

func clientLogHeader(cfg *transport.ClientConfig) []string {
	return []string{
		formatInfo("configuration:"),
		formatInfo("    endpoint: network (%s), address (%s)", cfg.Network, cfg.Address),
		formatInfo("    acknowledgement timeout: %s", cfg.AckTimeout),
	}
}

func clientDataReaders(
	t *testing.T,
	callSpy transport.Logger,
	noReader bool,
	errorReader bool,
	successReader bool,
) ([]func() ([]byte, error), []error, [][]byte, string) {
	t.Helper()

	readerFuncs := []func() ([]byte, error){}
	readerErrs := []error{}
	readerData := [][]byte{}
	readerCallMarker := testutils.GenerateMarker(t, "readClientData")

	if noReader {
		readerFuncs = append(readerFuncs, nil)
	}

	if errorReader {
		readerErrs = []error{testutils.GenerateError(t), testutils.GenerateError(t)}
		for _, err := range readerErrs {
			readerFuncs = append(
				readerFuncs,
				func() ([]byte, error) {
					callSpy.LogInfo(readerCallMarker)
					return nil, err
				},
			)
		}
	}

	if successReader {
		readerData = [][]byte{
			// note: a server started with transport.StartServer() accepts messages up to, but not including,
			//       protocol.MessageEndByte
			nil,
			testnetwork.GenerateNetProtocolMessage(t), // does not contain protocol.MessageEndByte

			// these test inputs cause the tests to fail with the current implementation, so we'll switch to a
			// length-prefixed message format, making netkit byte-agnostic and allowing arbitrary payloads
			//{protocol.MessageEndByte},
			//testutils.GenerateBytes(t), // may or may not contain protocol.MessageEndByte
		}
		for _, data := range readerData {
			readerFuncs = append(
				readerFuncs,
				func() ([]byte, error) {
					callSpy.LogInfo(readerCallMarker)
					return data, nil
				},
			)
		}
	}

	return readerFuncs, readerErrs, readerData, readerCallMarker
}

func clientAckCallbacks(t *testing.T, callSpy transport.Logger) ([]func(transport.Logger), string) {
	t.Helper()

	callbackCallMarker := testutils.GenerateMarker(t, "onAck")
	callbackFuncs := []func(transport.Logger){
		nil,
		func(logger transport.Logger) {
			if logger == callSpy {
				callSpy.LogInfo(callbackCallMarker)
			} else {
				callSpy.LogInfo("%s: function called with unexpected logger", callbackCallMarker)
			}
		},
	}

	return callbackFuncs, callbackCallMarker
}

func testClient(
	timeouts []time.Duration,
	clientReaders []func() ([]byte, error),
	clientAckCbs []func(transport.Logger),
	handler func(
		ackTimeout time.Duration,
		readClientData func() ([]byte, error),
		onAck func(transport.Logger),
	),
) {
	for _, ackTimeout := range timeouts {
		for _, readClientData := range clientReaders {
			for _, onAck := range clientAckCbs {
				handler(ackTimeout, readClientData, onAck)
			}
		}
	}
}

func TestStartClient(t *testing.T) {
	t.Run("halts when configuration values are invalid: ackTimeout <= 0", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			network := e.Network
			address := e.Address

			timeouts := []time.Duration{-time.Millisecond, 0}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, _ := clientDataReaders(t, logger, true, true, true)
			clientAckCbs, _ := clientAckCallbacks(t, logger)

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx := context.Background()
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
						Logs: append(
							clientLogHeader(&cfg),
							formatError("acknowledgement timeout must be greater than zero"),
						),
					}).Critical()
					testutils.AssertEqual(t, gotSuccess, false).Critical()

					logger.Reset()
				},
			)
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

	t.Run("halts when readClientData is nil", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			network := e.Network
			address := e.Address

			timeouts := []time.Duration{time.Millisecond}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, _ := clientDataReaders(t, logger, true, false, false)
			clientAckCbs, _ := clientAckCallbacks(t, logger)

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx := context.Background()
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
						Logs: append(
							clientLogHeader(&cfg),
							formatError("failed to read client data: reader function not provided"),
						),
					}).Critical()
					testutils.AssertEqual(t, gotSuccess, false).Critical()

					logger.Reset()
				},
			)
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

	t.Run("halts when readClientData returns an error", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			network := e.Network
			address := e.Address

			timeouts := []time.Duration{time.Millisecond}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, clientReadErrors, _, clientReadMarker := clientDataReaders(
				t, logger, false, true, false,
			)
			testutils.AssertEqual(t, len(clientReaders), len(clientReadErrors)).Critical()
			clientAckCbs, _ := clientAckCallbacks(t, logger)

			// testClient() is not used here because readerIdx is needed
			for _, ackTimeout := range timeouts {
				for readerIdx, readClientData := range clientReaders {
					for _, onAck := range clientAckCbs {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

						testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								formatInfo(clientReadMarker),
								formatError(
									"failed to read client data: %v", clientReadErrors[readerIdx],
								),
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, false).Critical()

						logger.Reset()
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

	t.Run("fails to connect to an unreachable server when readClientData succeeds", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()
			listener.Close() // make the server unreachable to clients

			timeouts := []time.Duration{time.Millisecond}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, clientReadMarker := clientDataReaders(t, logger, false, false, true)
			clientAckCbs, _ := clientAckCallbacks(t, logger)

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx := context.Background()
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
						Logs: append(
							clientLogHeader(&cfg),
							formatInfo(clientReadMarker),
							func() string {
								lastLog := logger.Logs[len(logger.Logs)-1]
								if strings.HasPrefix(lastLog, formatError(
									"failed to connect to server: dial %s %s:", network, address,
								)) {
									return lastLog
								}
								return unexpectedScenarioPlaceholder
							}(),
						),
					}).Critical()
					testutils.AssertEqual(t, gotSuccess, false).Critical()

					logger.Reset()
				},
			)
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

	t.Run("halts on missing server ACK when readClientData succeeds", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			timeouts := []time.Duration{time.Millisecond}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, clientReadMarker := clientDataReaders(t, logger, false, false, true)
			clientAckCbs, _ := clientAckCallbacks(t, logger)

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx := context.Background()
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
						Logs: append(
							clientLogHeader(&cfg),
							formatInfo(clientReadMarker),
							func() string {
								lastLog := logger.Logs[len(logger.Logs)-1]
								if strings.HasPrefix(lastLog, formatError(
									"failed to read server acknowledgement: message read failed: read %s",
									network,
								)) {
									return lastLog
								}
								return unexpectedScenarioPlaceholder
							}(),
						),
					}).Critical()
					testutils.AssertEqual(t, gotSuccess, false).Critical()

					logger.Reset()
				},
			)
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

	t.Run("halts on wrong server ACK when readClientData succeeds", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			// empirically chosen to avoid client-side read timeouts across closely spaced test runs
			baseTimeout := 30 * time.Millisecond

			listener := testnetwork.SetupSocketServer(t, e.Network, e.Address)
			defer listener.Close()
			network := e.Network
			address := listener.Addr().String()

			timeouts := []time.Duration{baseTimeout}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, clientReadMarker := clientDataReaders(t, logger, false, false, true)
			clientAckCbs, _ := clientAckCallbacks(t, logger)
			wrongAck := "KO" // same length as the real server acknowledgement

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					serverWriteFailure := make(chan error)
					go func() {
						serverConn := testnetwork.WaitForConnectionToSocketServer(t, listener)
						defer serverConn.Close()
						err := protocol.WriteToConnection(serverConn, []byte(wrongAck))
						serverWriteFailure <- err
					}()

					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx := context.Background()
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
						Logs: append(
							clientLogHeader(&cfg),
							formatInfo(clientReadMarker),
							func() string {
								lastLog := logger.Logs[len(logger.Logs)-1]
								if strings.HasPrefix(lastLog, formatError(
									"unexpected server acknowledgement: %q", wrongAck,
								)) {
									return lastLog
								}
								return unexpectedScenarioPlaceholder
							}(),
						),
					}).Critical()
					testutils.AssertEqual(t, gotSuccess, false).Critical()

					// receiving wrongAck does not guarantee that the server goroutine has published its
					// result to serverWriteFailure yet, so we must wait for a value to become available
					serverWriteErr, err := channels.WaitForValue(time.Millisecond, serverWriteFailure)
					testutils.AssertErrorIs(t, err, nil).Critical()
					testutils.AssertErrorIs(t, serverWriteErr, nil).Critical()

					logger.Reset()
				},
			)
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

	t.Run("receives valid server ACK when readClientData succeeds", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			// empirically chosen to avoid client-side read timeouts across closely spaced test runs
			baseTimeout := 30 * time.Millisecond

			runtimeCh := make(chan *transport.ServerRuntime)
			stopNetServer := testnetwork.StartNetServerAsync(
				t, transport.ServerConfig{
					Network:           e.Network,
					Address:           e.Address,
					MaxMsgBytes:       512,
					MaxConcurrentMsgs: 1,
					ReadTimeout:       baseTimeout,
				}, transport.NewNopLogger(), nil, runtimeCh,
			)
			defer func() {
				stopNetServer()
			}()
			runtimeInfo := testnetwork.ReadNetRuntime(t, runtimeCh)
			network := e.Network
			address := runtimeInfo.BoundAddr().String()

			timeouts := []time.Duration{baseTimeout}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, clientReadMarker := clientDataReaders(t, logger, false, false, true)
			clientAckCbs, clientAckMarker := clientAckCallbacks(t, logger)

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx := context.Background()
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					baseLogs := append(
						clientLogHeader(&cfg),
						formatInfo(clientReadMarker),
						formatInfo("received server acknowledgement: %q", "OK"),
					)
					testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
						Logs: append(
							append([]string{}, baseLogs...),
							func() []string {
								if onAck != nil {
									return []string{formatInfo(clientAckMarker)}
								}
								return []string{}
							}()...,
						),
					}).Critical()
					testutils.AssertEqual(t, gotSuccess, true).Critical()

					logger.Reset()
				},
			)
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

	t.Run("communicates with server until client stops when readClientData succeeds", func(t *testing.T) {
		check := func(e testnetwork.SocketEndpoint) {
			defer e.Release()

			baseTimeout := time.Millisecond

			runtimeCh := make(chan *transport.ServerRuntime)
			stopNetServer := testnetwork.StartNetServerAsync(
				t, transport.ServerConfig{
					Network:           e.Network,
					Address:           e.Address,
					MaxMsgBytes:       512,
					MaxConcurrentMsgs: 1,
					ReadTimeout:       baseTimeout,
				}, transport.NewNopLogger(), nil, runtimeCh,
			)
			defer func() {
				stopNetServer()
			}()
			runtimeInfo := testnetwork.ReadNetRuntime(t, runtimeCh)
			network := e.Network
			address := runtimeInfo.BoundAddr().String()

			timeouts := []time.Duration{baseTimeout}
			logger := &testnetwork.NetSpyLogger{}

			clientReaders, _, _, clientReadMarker := clientDataReaders(t, logger, false, false, true)
			clientAckCbs, clientAckMarker := clientAckCallbacks(t, logger)
			clientFailedAtLeastOnce := false

			testClient(
				timeouts, clientReaders, clientAckCbs,
				func(
					ackTimeout time.Duration,
					readClientData func() ([]byte, error),
					onAck func(transport.Logger),
				) {
					cfg := transport.ClientConfig{
						Network:    network,
						Address:    address,
						AckTimeout: ackTimeout,
					}
					stopCtx, cancel := context.WithCancel(context.Background())
					go cancel() // request the client to stop
					gotSuccess := transport.StartClient(cfg, logger, readClientData, onAck, stopCtx)

					logCount := len(logger.Logs)
					switch logCount {
					case 5:
						stopNote := " (and client stop was requested)"
						wantSuccess := false
						testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								formatInfo(clientReadMarker),
								func() string {
									// the scenarios below cover different circumstances under which stopCtx
									// may be cancelled
									lastLog := logger.Logs[len(logger.Logs)-1]
									if strings.HasPrefix(lastLog, formatError(
										"failed to send client data to server%s: ", stopNote,
									)) {
										return lastLog
									}
									if strings.HasPrefix(lastLog, formatError(
										"failed to read server acknowledgement%s: ", stopNote,
									)) {
										return lastLog
									}
									if strings.HasPrefix(lastLog, formatInfo(
										"received server acknowledgement: %q", "OK", // success with nil onAck
									)) {
										wantSuccess = true
										return lastLog // this case is also possible but less common
									}
									return unexpectedScenarioPlaceholder
								}(),
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, wantSuccess).Critical()
						if !gotSuccess {
							clientFailedAtLeastOnce = true
						}

					case 6: // this case is also possible but less common
						testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								formatInfo(clientReadMarker),
								formatInfo("received server acknowledgement: %q", "OK"),
								formatInfo(clientAckMarker), // success with non-nil onAck
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, true).Critical()

					default:
						t.Errorf("unexpected number of logs: got %d", logCount)
						// deliberately compare against an empty slice to inspect the actual logs
						testutils.AssertEqual(t, logger, &testnetwork.NetSpyLogger{Logs: []string{}}).
							Critical()
					}

					logger.Reset()
				},
			)

			testutils.AssertEqual(t, clientFailedAtLeastOnce, true).Critical()
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
