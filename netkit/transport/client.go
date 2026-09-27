package transport

import (
	"bytes"
	"net"
	"time"

	"github.com/arlogy/deltapilot/internal/process"
)

// StartClient creates and connects a client to a server endpoint.
//   - ackTimeout should be at least as long as the server's read timeout to prevent premature timeouts.
func StartClient(
	network string,
	address string,
	ackTimeout time.Duration,
	logger NetLogger,
	readClientMessage func() ([]byte, error),
	onAck func(ackData []byte, logger NetLogger),
) {
	// validate configuration values
	if !validateClientConfig(ackTimeout, logger) {
		return
	}

	// read client message
	msgData, err := readClientMessage()
	if err != nil {
		logger.LogError("failed to read client message: %v", err)
		return
	}

	// connect to the server
	conn, err := net.Dial(network, address)
	if err != nil {
		logger.LogError("failed to connect to server: %v", err)
		return
	}
	defer conn.Close()

	// set up a shutdown listener
	process.RegisterShutdownHandler(func() {
		conn.Close() // this unblocks pending operations on conn, causing them to return an error
	})

	// send the message to the server
	if err := WriteToConnection(conn, msgData); err != nil {
		logger.LogError("failed to send message to server: %v", err)
		return
	}

	// receive server acknowledgement; failure to do so means successful processing cannot be confirmed
	ackData, err := ReadFromConnection(conn, len(serverAck), ackTimeout)
	if err != nil {
		logger.LogError("failed to read server acknowledgement: %v", err)
		return
	}
	if !bytes.Equal(ackData, []byte(serverAck)) {
		logger.LogError("unexpected server acknowledgement: %q", ackData)
		return
	}
	onAck(ackData, logger)
}

func validateClientConfig(ackTimeout time.Duration, logger NetLogger) bool {
	logger.LogInfo("configuration: acknowledgement timeout (%s)", ackTimeout)

	if ackTimeout <= 0 {
		logger.LogError("acknowledgement timeout must be greater than zero")
		return false
	}

	return true
}
