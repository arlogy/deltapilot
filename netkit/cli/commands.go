package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arlogy/deltapilot/internal/process"
	"github.com/arlogy/deltapilot/netkit/transport"
)

func runServer(
	subCommand string,
	args []string,
	defaultNet string,
	defaultAddr string,
	defaultMaxBytesPerMsg int,
	defaultMaxMsgs int,
	defaultTimeout time.Duration,
	logger transport.Logger,
	onMessage func(msgData []byte, logger transport.Logger),
) {
	subFlags := flag.NewFlagSet(subCommand, flag.ExitOnError)
	nettype := subFlags.String("nettype", defaultNet, "network type passed to the networking layer")

	netaddr := subFlags.String("netaddr", defaultAddr, "network address to listen on for nettype")
	maxMsgBytes := subFlags.Int("max-message-bytes", defaultMaxBytesPerMsg, "maximum message size in bytes")
	maxConcurrentMsgs := subFlags.Int(
		"max-concurrent-messages", defaultMaxMsgs, "maximum number of concurrent messages",
	)
	readTimeout := subFlags.Duration(
		"read-timeout", defaultTimeout, "maximum time to wait for a read from a client",
	)

	subFlags.Parse(args)
	if subFlags.NArg() > 0 {
		fmt.Fprintf(subFlags.Output(), "unexpected positional arguments: %q\n", subFlags.Args())
		os.Exit(2) // same exit status as subFlags.Parse() above
	}

	cfg := transport.ServerConfig{
		Network:           *nettype,
		Address:           *netaddr,
		MaxMsgBytes:       *maxMsgBytes,
		MaxConcurrentMsgs: *maxConcurrentMsgs,
		ReadTimeout:       *readTimeout,
	}
	stopCtx, cancel := process.CreateShutdownContext()
	defer cancel() // stop signal interception only after the server has finished, including its cleanup
	transport.StartServer(cfg, logger, onMessage, stopCtx, nil)
}

func runClient(
	subCommand string,
	args []string,
	defaultNet string,
	defaultAddr string,
	defaultTimeout time.Duration,
	logger transport.Logger,
	onAck func(logger transport.Logger),
) {
	subFlags := flag.NewFlagSet(subCommand, flag.ExitOnError)
	nettype := subFlags.String("nettype", defaultNet, "network type passed to the networking layer")
	netaddr := subFlags.String("netaddr", defaultAddr, "network address to connect to for nettype")
	ackTimeout := subFlags.Duration(
		"ack-timeout",
		defaultTimeout,
		strings.Join([]string{
			"should be at least as long as the server's read timeout so the client does not time out first",
			"it is the maximum time to wait for an acknowledgement",
		}, "\n"),
	)

	subFlags.Parse(args)
	if subFlags.NArg() > 0 {
		fmt.Fprintf(subFlags.Output(), "unexpected positional arguments: %q\n", subFlags.Args())
		os.Exit(2) // same exit status as subFlags.Parse() above
	}

	cfg := transport.ClientConfig{
		Network:    *nettype,
		Address:    *netaddr,
		AckTimeout: *ackTimeout,
	}
	stopCtx, cancel := process.CreateShutdownContext()
	defer cancel() // stop signal interception only after the client has finished, including its cleanup
	transport.StartClient(cfg, logger, process.ReadFromStdin, onAck, stopCtx)
}
