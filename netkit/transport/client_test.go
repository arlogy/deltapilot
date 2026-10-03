package transport_test

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/channels"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/netkit/transport"
)

func clientLogHeader(cfg *transport.ClientConfig) []string {
	return []string{
		fmt.Sprintf("[INFO] configuration:%s", ""),
		fmt.Sprintf("[INFO]     endpoint: network (%s), address (%s)", cfg.Network, cfg.Address),
		fmt.Sprintf("[INFO]     acknowledgement timeout: %s", cfg.AckTimeout),
	}
}

func TestStartClient(t *testing.T) {
	t.Run("halts when configuration values are invalid: ackTimeout <= 0", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			network := e.Network
			address := e.Address

			timeouts := []time.Duration{-time.Millisecond, 0}
			logger := testutils.NetSpyLogger{}

			clientMsgReadLog := "readClientMessage: " + testutils.GenerateAlphanumeric(t)
			clientMsgReadErrs := []error{testutils.GenerateError(t), testutils.GenerateError(t)}
			clientMsgReaders := []func() ([]byte, error){
				nil,
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, clientMsgReadErrs[0]
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), clientMsgReadErrs[1]
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, nil
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), nil
				},
			}

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for _, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, &logger, readClientMessage, onAck, stopCtx)
						testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								"[ERROR] acknowledgement timeout must be greater than zero",
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, false).Critical()
						logger.Logs = []string{} // reset
					}
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

	t.Run("halts when readClientMessage is nil", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			network := e.Network
			address := e.Address

			timeouts := []time.Duration{time.Millisecond}
			logger := testutils.NetSpyLogger{}

			clientMsgReaders := []func() ([]byte, error){nil}

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for _, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, &logger, readClientMessage, onAck, stopCtx)
						testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								"[ERROR] failed to read client message: reader function not provided",
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, false).Critical()
						logger.Logs = []string{} // reset
					}
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

	t.Run("halts when readClientMessage returns an error", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			network := e.Network
			address := e.Address

			timeouts := []time.Duration{time.Millisecond}
			logger := testutils.NetSpyLogger{}

			clientMsgReadLog := "readClientMessage: " + testutils.GenerateAlphanumeric(t)
			clientMsgReadErrs := []error{testutils.GenerateError(t), testutils.GenerateError(t)}
			clientMsgReaders := []func() ([]byte, error){
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, clientMsgReadErrs[0]
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), clientMsgReadErrs[1]
				},
			}
			testutils.AssertEqual(t, len(clientMsgReadErrs), len(clientMsgReaders)).Critical()

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for readerIdx, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, &logger, readClientMessage, onAck, stopCtx)
						testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								"[INFO] "+clientMsgReadLog,
								fmt.Sprintf(
									"[ERROR] failed to read client message: %v", clientMsgReadErrs[readerIdx],
								),
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, false).Critical()
						logger.Logs = []string{} // reset
					}
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

	t.Run("fails to connect to an unreachable server when readClientMessage succeeds", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()
			listener.Close() // make the server unreachable

			timeouts := []time.Duration{time.Millisecond}
			logger := testutils.NetSpyLogger{}

			clientMsgReadLog := "readClientMessage: " + testutils.GenerateAlphanumeric(t)
			clientMsgReaders := []func() ([]byte, error){
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, nil
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), nil
				},
			}

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for _, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, &logger, readClientMessage, onAck, stopCtx)
						testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								"[INFO] "+clientMsgReadLog,
								func() string {
									lastLog := logger.Logs[len(logger.Logs)-1]
									if strings.HasPrefix(lastLog, fmt.Sprintf(
										"[ERROR] failed to connect to server: dial %s %s:", network, address,
									)) {
										return lastLog
									}
									// no expected scenario was matched
									return testutils.GenerateError(t).Error()
								}(),
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, false).Critical()
						logger.Logs = []string{} // reset
					}
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

	t.Run("communicates with server without ACK when readClientMessage succeeds", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			listener := testutils.SetupSocketServer(t, e.Network, e.Address)
			network := e.Network
			address := listener.Addr().String()

			timeouts := []time.Duration{time.Millisecond}
			logger := testutils.NetSpyLogger{}

			clientMsgReadLog := "readClientMessage: " + testutils.GenerateAlphanumeric(t)
			clientMsgReaders := []func() ([]byte, error){
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, nil
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), nil
				},
			}

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for _, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, &logger, readClientMessage, onAck, stopCtx)
						testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
							Logs: append(
								clientLogHeader(&cfg),
								"[INFO] "+clientMsgReadLog,
								func() string {
									lastLog := logger.Logs[len(logger.Logs)-1]
									if strings.HasPrefix(lastLog, fmt.Sprintf(
										"[ERROR] failed to read server acknowledgement: read %s", network,
									)) {
										return lastLog
									}
									// no expected scenario was matched
									return testutils.GenerateError(t).Error()
								}(),
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, false).Critical()
						logger.Logs = []string{} // reset
					}
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

	t.Run("communicates with server and receives ACK when readClientMessage succeeds", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			// this duration reduces premature client-side read timeouts in closely repeated test runs
			baseTimeout := 15 * time.Millisecond

			runtimeCh := make(chan *transport.ServerRuntime)
			go func() {
				cfg := transport.ServerConfig{
					Network:           e.Network,
					Address:           e.Address,
					MaxMsgBytes:       1024,
					MaxConcurrentMsgs: 1,
					ReadTimeout:       baseTimeout,
				}
				transport.StartServer(cfg, transport.NewNopLogger(), nil, context.Background(), runtimeCh)
			}()

			runtimeInfo := <-runtimeCh // wait for the server's runtime information
			network := e.Network
			address := runtimeInfo.BoundAddr().String()

			timeouts := []time.Duration{baseTimeout}
			logger := testutils.NetSpyLogger{}

			clientMsgReadLog := "readClientMessage: " + testutils.GenerateAlphanumeric(t)
			clientMsgReaders := []func() ([]byte, error){
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, nil
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), nil
				},
			}

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger_.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for _, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						stopCtx := context.Background()
						gotSuccess := transport.StartClient(cfg, &logger, readClientMessage, onAck, stopCtx)
						baseLogs := append(
							clientLogHeader(&cfg),
							"[INFO] "+clientMsgReadLog,
							fmt.Sprintf("[INFO] received server acknowledgement: %q", "OK"),
						)
						testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
							Logs: append(
								append([]string{}, baseLogs...),
								func() []string {
									if onAck == nil {
										return []string{}
									}
									return []string{"[INFO] " + akcLog}
								}()...,
							),
						}).Critical()
						testutils.AssertEqual(t, gotSuccess, true).Critical()
						logger.Logs = []string{} // reset
					}
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

	t.Run("communicates with server until stopped when readClientMessage succeeds", func(t *testing.T) {
		check := func(e testutils.SocketEndpoint) {
			defer e.Release(t)

			baseTimeout := time.Millisecond

			runtimeCh := make(chan *transport.ServerRuntime)
			go func() {
				cfg := transport.ServerConfig{
					Network:           e.Network,
					Address:           e.Address,
					MaxMsgBytes:       1024,
					MaxConcurrentMsgs: 1,
					ReadTimeout:       baseTimeout,
				}
				transport.StartServer(cfg, transport.NewNopLogger(), nil, context.Background(), runtimeCh)
			}()

			runtimeInfo := <-runtimeCh // wait for the server's runtime information
			network := e.Network
			address := runtimeInfo.BoundAddr().String()

			timeouts := []time.Duration{time.Millisecond}
			logger := testutils.NetSpyLogger{}

			clientMsgReadLog := "readClientMessage: " + testutils.GenerateAlphanumeric(t)
			clientMsgReaders := []func() ([]byte, error){
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return nil, nil
				},
				func() ([]byte, error) {
					logger.LogInfo("%s", clientMsgReadLog)
					return testutils.GenerateBytes(t), nil
				},
			}

			akcLog := "onAck: " + testutils.GenerateAlphanumeric(t)
			ackCallbacks := []func(transport.Logger){
				nil,
				func(logger_ transport.Logger) {
					logger.LogInfo("%s", akcLog)
				},
			}

			for _, ackTimeout := range timeouts {
				for _, readClientMessage := range clientMsgReaders {
					for _, onAck := range ackCallbacks {
						cfg := transport.ClientConfig{
							Network:    network,
							Address:    address,
							AckTimeout: ackTimeout,
						}
						gotSuccess := false
						clientStarted := channels.WaitForReturn(
							15*time.Millisecond, // this duration allows the client to start
							func() {
								stopCtx, cancel := context.WithCancel(context.Background())
								go cancel() // trigger the stop
								gotSuccess = transport.StartClient(
									cfg, &logger, readClientMessage, onAck, stopCtx,
								)
							},
						)
						if !clientStarted {
							t.Fatalf("StartClient did not finish within the expected time")
						}
						logCount := len(logger.Logs)
						switch logCount {
						case 5:
							wantSuccess := false
							testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
								Logs: append(
									clientLogHeader(&cfg),
									"[INFO] "+clientMsgReadLog,
									func() string {
										// the scenarios below cover different circumstances under which
										// stopCtx may be cancelled
										lastLog := logger.Logs[len(logger.Logs)-1]
										if strings.HasPrefix(lastLog, fmt.Sprintf(
											"[ERROR] failed to send message to server: write %s", network,
										)) {
											return lastLog
										}
										if strings.HasPrefix(lastLog, fmt.Sprintf(
											"[ERROR] failed to read server acknowledgement: read %s", network,
										)) {
											return lastLog
										}
										if strings.HasPrefix(lastLog, fmt.Sprintf(
											"[INFO] received server acknowledgement: %q", "OK",
										)) {
											wantSuccess = true
											return lastLog // this case is possible but rarely occurs
										}
										// no expected scenario was matched
										return testutils.GenerateError(t).Error()
									}(),
								),
							}).Critical()
							testutils.AssertEqual(t, gotSuccess, wantSuccess).Critical()
						case 6: // this case is possible but rarely occurs
							testutils.AssertEqual(t, logger, testutils.NetSpyLogger{
								Logs: append(
									clientLogHeader(&cfg),
									"[INFO] "+clientMsgReadLog,
									fmt.Sprintf("[INFO] received server acknowledgement: %q", "OK"),
									"[INFO] "+akcLog,
								),
							}).Critical()
							testutils.AssertEqual(t, gotSuccess, true).Critical()
						default:
							t.Errorf("unexpected number of logs: got %d", logCount)
							// deliberately compare against an empty slice to inspect the actual logs
							testutils.AssertEqual(t, logger, testutils.NetSpyLogger{Logs: []string{}}).
								Critical()
						}
						logger.Logs = []string{} // reset
					}
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
