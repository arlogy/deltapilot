package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/arlogy/deltapilot/netkit/transport"
)

func RunApp(
	defaultNet string,
	defaultAddr string,
	defaultMaxBytesPerMsg int,
	defaultMaxMsgs int,
	defaultTimeout time.Duration,
	logger transport.NetLogger,
	messageReceivedHandler func(msgData []byte, logger transport.NetLogger),
	ackReceivedHandler func(msgData []byte, logger transport.NetLogger),
) {
	subCommand := ""
	if len(os.Args) > 1 {
		subCommand = os.Args[1]
	}

	switch subCommand {
	case "server":
		runServer(
			subCommand, os.Args[2:],
			defaultNet, defaultAddr, defaultMaxBytesPerMsg, defaultMaxMsgs, defaultTimeout,
			logger, messageReceivedHandler,
		)

	case "client":
		runClient(
			subCommand, os.Args[2:],
			defaultNet, defaultAddr, defaultTimeout,
			logger, ackReceivedHandler,
		)

	default:
		// logger is not used here because subCommand flag parsing in runServer() and runClient() does not use
		// it either

		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintf(os.Stderr, "  %s <subcommand> [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr)

		fmt.Fprintln(os.Stderr, "subcommand:")
		fmt.Fprintln(os.Stderr, "  server  starts a server with default options")
		fmt.Fprintln(os.Stderr, "  client  starts a client with default options")

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "options:")
		fmt.Fprintln(os.Stderr, "  -h      displays help and options for the selected subcommand")

		os.Exit(2)
	}
}
