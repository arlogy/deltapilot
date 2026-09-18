package main

import (
	"github.com/arlogy/deltapilot/netkit/transport"
)

func handleMessageReceived(msgData []byte, logger transport.NetLogger) {
	logger.LogInfo("received message: %q", string(msgData)) // temporary handling
}

func handleAckReceived(msgData []byte, logger transport.NetLogger) {
	logger.LogInfo("received acknowledgement: %q", string(msgData)) // temporary handling
}
