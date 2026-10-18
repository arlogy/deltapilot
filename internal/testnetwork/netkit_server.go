package testnetwork

import (
	"context"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/channels"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/netkit/transport"
)

var netServerRuntimeTimeout = 30 * time.Millisecond // empirically chosen to avoid flaky tests

func StartNetServerAsync(
	t *testing.T,
	cfg transport.ServerConfig,
	logger transport.Logger,
	onMessage func(msgData []byte, logger transport.Logger),
	runtimeCh chan<- *transport.ServerRuntime,
) func() {
	t.Helper()

	// stopServer is returned to the caller rather than registered with t.Cleanup(), because fatal assertions
	// in cleanup functions do not abort the test immediately; this also lets the caller control the server
	// lifecycle independently of t

	serverSucceeded := false

	stopCtx, cancel := context.WithCancel(context.Background())
	stoppedCh := make(chan struct{})
	stopServer := func() {
		t.Helper()

		stopTimeout := 15 * time.Millisecond // empirically chosen to avoid flaky tests
		if err := testutils.StopOperation("StartServer", cancel, stoppedCh, stopTimeout); err != nil {
			t.Fatalf("failed to stop server: %v", err)
		}

		if !serverSucceeded {
			t.Fatal("server did not terminate successfully")
		}
	}

	go func() {
		serverSucceeded = transport.StartServer(cfg, logger, onMessage, stopCtx, runtimeCh)
		close(stoppedCh)
	}()

	return stopServer
}

func ReadNetRuntime(t *testing.T, runtimeCh <-chan *transport.ServerRuntime) *transport.ServerRuntime {
	t.Helper()

	runtimeInfo, err := channels.WaitForValue(netServerRuntimeTimeout, runtimeCh)
	if err != nil {
		t.Fatalf("failed to receive runtime information: %v", err)
	}
	if runtimeInfo == nil {
		t.Fatal("got nil runtime information, want non-nil")
	}

	return runtimeInfo
}
