package transport

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/arlogy/deltapilot/internal/channels"
)

const serverAck = "OK"

// ServerConfig contains the configuration of a server.
type ServerConfig struct {
	Network           string
	Address           string
	MaxMsgBytes       int
	MaxConcurrentMsgs int
	ReadTimeout       time.Duration
}

// ServerRuntime contains runtime information about a server.
type ServerRuntime struct {
	// the address a server listener is bound to
	serverAddr net.Addr
	// the number of clients accepted but not yet scheduled for serving; at most 1
	clientPendingCount atomic.Int64
	// the number of clients currently being served; at most the allowed limit
	clientActiveCount atomic.Int64
}

func (r *ServerRuntime) BoundAddr() net.Addr {
	return r.serverAddr
}

func (r *ServerRuntime) ConnPendingCount() int64 {
	return r.clientPendingCount.Load()
}

func (r *ServerRuntime) ConnActiveCount() int64 {
	return r.clientActiveCount.Load()
}

// StartServer creates a server endpoint, starts serving clients, and blocks until stopped.
//   - cfg describes the server configuration.
//   - logger logs server diagnostics and is assumed to be non-nil; when logging is not needed, NewNopLogger()
//     may be passed.
//   - onMessage is an optional callback called when the server receives a message from a client.
//   - stopCtx signals when the server must stop, allowing proper cleanup.
//   - runtimeCh is dynamically updated with runtime information about the server. It is optional and provides
//     access to server information while keeping the entire server lifecycle (create, serve, stop) within a
//     single function, avoiding the need to handle invalid operations such as attempting to start serving
//     clients more than once or after the server has stopped and its listener has been closed.
func StartServer(
	cfg ServerConfig,
	logger Logger,
	onMessage func(msgData []byte, logger Logger),
	stopCtx context.Context,
	runtimeCh chan<- *ServerRuntime,
) bool {
	network := cfg.Network
	address := cfg.Address
	maxMsgBytes := cfg.MaxMsgBytes
	maxConcurrentMsgs := cfg.MaxConcurrentMsgs
	readTimeout := cfg.ReadTimeout

	// validate configuration values
	if !validateServerConfig(network, address, maxMsgBytes, maxConcurrentMsgs, readTimeout, logger) {
		return false
	}

	// create the server listener
	listener, err := net.Listen(network, address)
	if err != nil {
		logger.LogError("failed to listen on %q (%q): %v", address, network, err)
		return false
	}
	defer listener.Close()
	logger.LogInfo("listening on %q (%q)", listener.Addr().String(), listener.Addr().Network())

	// report the server runtime information if requested
	runtimeInfo := ServerRuntime{
		serverAddr:         listener.Addr(),
		clientPendingCount: atomic.Int64{},
		clientActiveCount:  atomic.Int64{},
	}
	if runtimeCh != nil {
		runtimeCh <- &runtimeInfo
	}

	// set up a stop listener
	channels.RunOnReceive(stopCtx.Done(), func() {
		listener.Close() // this unblocks listener.Accept(), causing it to return an error
	})

	// wait for clients to connect
	clientSlots := make(chan struct{}, maxConcurrentMsgs) // a semaphore to prevent ConcurrentFlooding
serveLoop:
	for {
		// accept a connection from a client
		conn, err := listener.Accept()
		if err != nil {
			if stopCtx.Err() != nil {
				break // stop requested
			}
			logger.LogError("failed to accept client connection: %v", err)
			continue
		}
		runtimeInfo.clientPendingCount.Add(1)

		// wait if maxConcurrentMsgs clients are already being handled
		select {
		case clientSlots <- struct{}{}: // acquire a slot or wait for one to become available
		case <-stopCtx.Done(): // stop requested
			runtimeInfo.clientPendingCount.Add(-1)
			break serveLoop
		}
		runtimeInfo.clientPendingCount.Add(-1)

		// handle the client connection concurrently with other connections
		go func() {
			runtimeInfo.clientActiveCount.Add(1)
			defer func() {
				<-clientSlots // release the slot
				runtimeInfo.clientActiveCount.Add(-1)
			}()
			handleClientConnection(conn, maxMsgBytes, readTimeout, logger, onMessage)
		}()
	}

	logger.LogInfo("server stopped")
	return true
}

func handleClientConnection(
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
	logger Logger,
	onMessage func(msgData []byte, logger Logger),
) {
	defer conn.Close()

	// receive a message from the client
	msgData, err := ReadFromConnection(conn, maxMsgBytes, readTimeout)
	if err != nil {
		logger.LogError("failed to read client message: %v", err)
		return
	}
	if onMessage != nil {
		onMessage(msgData, logger)
	}

	// send an acknowledgement to the client
	if err := WriteToConnection(conn, []byte(serverAck)); err != nil {
		logger.LogError("failed to send acknowledgement to client: %v", err)
		return
	}
}

func validateServerConfig(
	network string,
	address string,
	maxMsgBytes int,
	maxConcurrentMsgs int,
	readTimeout time.Duration,
	logger Logger,
) bool {
	logger.LogInfo("configuration:")
	logger.LogInfo("    endpoint: network (%s), address (%s)", network, address)
	logger.LogInfo("    max bytes per message: %d", maxMsgBytes)
	logger.LogInfo("    max concurrent messages: %d", maxConcurrentMsgs)
	logger.LogInfo("    read timeout: %s", readTimeout)

	if maxMsgBytes <= 0 {
		logger.LogError("max bytes per message must be greater than zero")
		return false
	}

	if maxConcurrentMsgs <= 0 {
		logger.LogError("max concurrent messages must be greater than zero")
		return false
	}

	if readTimeout <= 0 {
		logger.LogError("read timeout must be greater than zero")
		return false
	}

	return true
}
