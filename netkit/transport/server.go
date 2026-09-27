package transport

import (
	"net"
	"time"

	"github.com/arlogy/deltapilot/internal/process"
)

const serverAck = "OK"

// StartServer creates a server endpoint.
func StartServer(
	network string,
	address string,
	maxMsgBytes int,
	maxConcurrentMsgs int,
	readTimeout time.Duration,
	logger NetLogger,
	onMessage func(msgData []byte, logger NetLogger),
) {
	// validate configuration values
	if !validateServerConfig(maxMsgBytes, maxConcurrentMsgs, readTimeout, logger) {
		return
	}

	// create the server listener
	listener, err := net.Listen(network, address)
	if err != nil {
		logger.LogError("failed to listen on %q (%q): %v", address, network, err)
		return
	}
	defer listener.Close()
	logger.LogInfo("listening on %q (%q)", address, network)

	// set up a shutdown listener
	ctx := process.RegisterShutdownHandler(func() {
		listener.Close() // this unblocks listener.Accept(), causing it to return an error
	})

	// wait for clients to connect
	clientSlots := make(chan struct{}, maxConcurrentMsgs) // a semaphore to prevent ConcurrentFlooding
	for {
		// accept a connection from a client
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break // shutdown requested
			}
			logger.LogError("failed to accept client connection: %v", err)
			continue
		}

		// handle the client connection concurrently with other connections, up to maxConcurrentMsgs clients
		clientSlots <- struct{}{} // acquire a slot or wait for one to become available
		go func() {
			defer func() {
				<-clientSlots // release the slot
			}()
			handleClientConnection(conn, maxMsgBytes, readTimeout, logger, onMessage)
		}()
	}

	logger.LogInfo("server stopped")
}

func handleClientConnection(
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
	logger NetLogger,
	onMessage func(msgData []byte, logger NetLogger),
) {
	defer conn.Close()

	// receive a message from the client
	msgData, err := ReadFromConnection(conn, maxMsgBytes, readTimeout)
	if err != nil {
		logger.LogError("failed to read client message: %v", err)
		return
	}
	onMessage(msgData, logger)

	// send an acknowledgement to the client
	if err := WriteToConnection(conn, []byte(serverAck)); err != nil {
		logger.LogError("failed to send acknowledgement to client: %v", err)
		return
	}
}

func validateServerConfig(
	maxMsgBytes int,
	maxConcurrentMsgs int,
	readTimeout time.Duration,
	logger NetLogger,
) bool {
	logger.LogInfo(
		"configuration: max bytes per message (%d), max concurrent messages (%d), read timeout (%s)",
		maxMsgBytes,
		maxConcurrentMsgs,
		readTimeout,
	)

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
