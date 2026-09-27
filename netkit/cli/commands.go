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
	logger transport.NetLogger,
	onMessage func(msgData []byte, logger transport.NetLogger),
) {
	subFlags := flag.NewFlagSet(subCommand, flag.ExitOnError)
	nettype := subFlags.String("nettype", defaultNet, "network type (tcp, tcp4, tcp6, unix or unixpacket)")
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

	transport.StartServer(
		*nettype, *netaddr, *maxMsgBytes, *maxConcurrentMsgs, *readTimeout, logger, onMessage,
	)
}

func runClient(
	subCommand string,
	args []string,
	defaultNet string,
	defaultAddr string,
	defaultTimeout time.Duration,
	logger transport.NetLogger,
	onAck func(ackData []byte, logger transport.NetLogger),
) {
	subFlags := flag.NewFlagSet(subCommand, flag.ExitOnError)
	nettype := subFlags.String("nettype", defaultNet, "network type (tcp, tcp4, tcp6, unix or unixpacket)")
	netaddr := subFlags.String("netaddr", defaultAddr, "network address to connect to for nettype")
	ackTimeout := subFlags.Duration(
		"ack-timeout",
		defaultTimeout,
		strings.Join([]string{
			"should be at least as long as the server's read timeout to prevent premature timeouts",
			"it is the maximum time to wait for an acknowledgement",
		}, "\n"),
	)

	subFlags.Parse(args)
	if subFlags.NArg() > 0 {
		fmt.Fprintf(subFlags.Output(), "unexpected positional arguments: %q\n", subFlags.Args())
		os.Exit(2) // same exit status as subFlags.Parse() above
	}

	transport.StartClient(*nettype, *netaddr, *ackTimeout, logger, process.ReadFromStdin, onAck)
}
