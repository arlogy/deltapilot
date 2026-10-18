package testnetwork

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/stream"
	"github.com/arlogy/deltapilot/internal/testutils"
)

func ConfigureReadDeadline(t *testing.T, conn net.Conn, timeout time.Duration) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("failed to configure read deadline: %v", err)
	}
}

func ClearReadDeadline(t *testing.T, conn net.Conn) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("failed to clear read deadline: %v", err)
	}
}

func ExpectBytesAvailable(
	t *testing.T,
	conn net.Conn,
	network string,
	address string,
	serverSide bool,
	observationPeriod time.Duration,
	wantData []byte,
) {
	t.Helper()

	gotData := ReadAvailableBytes(t, conn, network, address, serverSide, observationPeriod)
	testutils.AssertEqual(t, gotData, wantData).Critical()
}

func ExpectBytesWritten(t *testing.T, conn net.Conn, data []byte) {
	t.Helper()

	err := stream.WriteExactBytes(conn, data)
	testutils.AssertErrorIs(t, err, nil).Critical()
}

func FormatBytesReadTimeout(conn net.Conn, network string, address string, serverSide bool) string {
	if serverSide {
		return fmt.Sprintf(
			"read %s %s->%s: i/o timeout", network, address, conn.RemoteAddr(),
		)
	} else {
		return fmt.Sprintf(
			"read %s %s->%s: i/o timeout", network, conn.LocalAddr(), address,
		)
	}
}

func ReadAvailableBytes(
	t *testing.T,
	conn net.Conn,
	network string,
	address string,
	serverSide bool,
	observationPeriod time.Duration,
) []byte {
	t.Helper()

	ConfigureReadDeadline(t, conn, observationPeriod)
	defer ClearReadDeadline(t, conn)

	gotData, err := io.ReadAll(conn)
	testutils.AssertErrorMessage(t, err, FormatBytesReadTimeout(conn, network, address, serverSide)).
		Critical()

	return gotData
}
