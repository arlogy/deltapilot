package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/arlogy/deltapilot/netkit/cli"
	"github.com/arlogy/deltapilot/netkit/transport"
)

const KiB = 1024
const MiB = KiB * KiB

func main() {
	defaultNet := "unix"
	defaultAddr := filepath.Join(os.TempDir(), "deltapilot.sock") // default Unix socket file path

	defaultMaxBytesPerMsg := 4 * MiB
	defaultMaxMsgs := 64
	defaultTimeout := 30 * time.Second

	logger := transport.NewtDefaultLogger()

	cli.RunApp(
		defaultNet,
		defaultAddr,
		defaultMaxBytesPerMsg,
		defaultMaxMsgs,
		defaultTimeout,
		logger,
		handleMessageReceived,
		handleAckReceived,
	)
}
