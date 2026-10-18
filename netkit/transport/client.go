package transport

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/arlogy/deltapilot/internal/channels"
	"github.com/arlogy/deltapilot/netkit/protocol"
)

// ClientConfig contains the configuration of a client.
//   - AckTimeout should be at least as long as the read timeout configured for the server the client connects
//     to, to avoid the client timing out before the server finishes waiting for data.
type ClientConfig struct {
	Network    string
	Address    string
	AckTimeout time.Duration
}

// StartClient creates and connects a client to a server endpoint.
//   - cfg describes the client configuration.
//   - logger logs client diagnostics and is assumed to be non-nil; when logging is not needed, NewNopLogger()
//     may be passed.
//   - readClientData provides the data that the client must send to the server.
//   - onAck is an optional callback called when the client receives acknowledgement from the server.
//   - stopCtx signals when an early stop is requested, allowing proper cleanup in that case.
//
// I/O capabilities are accessed by calling protocol.* functions directly, rather than through a DataIO
// wrapper passed to the function. This avoids exposing internally managed connection objects.
func StartClient(
	cfg ClientConfig,
	logger Logger,
	readClientData func() ([]byte, error),
	onAck func(logger Logger),
	stopCtx context.Context,
) bool {
	network := cfg.Network
	address := cfg.Address
	ackTimeout := cfg.AckTimeout

	// validate configuration values
	if !validateClientConfig(network, address, ackTimeout, logger) {
		return false
	}

	// read the client data
	if readClientData == nil {
		logger.LogError("failed to read client data: reader function not provided")
		return false
	}
	wireData, err := readClientData()
	if err != nil {
		logger.LogError("failed to read client data: %v", err)
		return false
	}

	// connect to the server
	conn, err := net.Dial(network, address)
	if err != nil {
		logger.LogError("failed to connect to server: %v", err)
		return false
	}
	defer conn.Close()

	// set up an early-stop listener
	channels.RunWhenUnblocked(stopCtx.Done(), func() {
		conn.Close() // this unblocks pending operations on conn, causing them to return an error
	})

	// send the client data to the server
	if err := protocol.WriteToConnection(conn, wireData); err != nil {
		logger.LogError("failed to send client data to server%s: %v", clientStopAnnotation(stopCtx), err)
		return false
	}

	// receive server acknowledgement; failure to do so means successful processing cannot be confirmed
	ackData, err := protocol.ReadFromConnection(conn, len(serverAck), ackTimeout)
	if err != nil {
		logger.LogError("failed to read server acknowledgement%s: %v", clientStopAnnotation(stopCtx), err)
		return false
	}
	if !bytes.Equal(ackData, []byte(serverAck)) {
		logger.LogError("unexpected server acknowledgement: %q", ackData)
		return false
	}
	logger.LogInfo("received server acknowledgement: %q", string(ackData))
	if onAck != nil {
		onAck(logger)
	}

	return true
}

func clientStopAnnotation(stopCtx context.Context) string {
	if stopCtx.Err() != nil { // stop requested
		return " (and client stop was requested)"
	}
	return ""
}

func validateClientConfig(network string, address string, ackTimeout time.Duration, logger Logger) bool {
	logger.LogInfo("configuration:")
	logger.LogInfo("    endpoint: network (%s), address (%s)", network, address)
	logger.LogInfo("    acknowledgement timeout: %s", ackTimeout)

	if ackTimeout <= 0 {
		logger.LogError("acknowledgement timeout must be greater than zero")
		return false
	}

	return true
}
