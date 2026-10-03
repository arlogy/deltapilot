package main

import (
	"github.com/arlogy/deltapilot/netkit/transport"
)

func handleMessageReceived(msgData []byte, logger transport.Logger) {
	logger.LogInfo("received message: %q", string(msgData)) // temporary handling
}

func handleAckReceived(logger transport.Logger) {
	logger.LogInfo("done!")
}
