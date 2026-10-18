package testnetwork

import (
	"net"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/sequencer"
	"github.com/arlogy/deltapilot/internal/testutils"
)

const readDeadlineTolerance = time.Millisecond - 1000*time.Nanosecond

func NewConnStubFailingToApplyReadDeadline(
	t *testing.T,
	conn net.Conn,
	timeout time.Duration,
	err error,
) *StubConn {
	t.Helper()

	return &StubConn{
		Conn: conn,
		StubInfo: testutils.StubInfo{
			ReturnOverrides: map[string]any{
				"SetReadDeadline": []func(t time.Time) error{
					func(d time.Time) error {
						testutils.AssertTimeWas(t, d, time.Now().Add(timeout), readDeadlineTolerance).
							Critical()
						testutils.AssertErrorNotNil(t, err).Critical()
						return err
					},
					func(_ time.Time) error {
						t.Fatal("unexpected call")
						return nil
					},
				},
			},
			OverrideSequencers: map[string]sequencer.ValueAdvancer[int64]{
				"SetReadDeadline": &sequencer.Numeric{},
			},
		},
	}
}

func NewConnStubFailingToCancelReadDeadline(
	t *testing.T,
	conn net.Conn,
	timeout time.Duration,
	err error,
) *StubConn {
	t.Helper()

	stub := &StubConn{
		Conn: conn,
	}

	stub.StubInfo = testutils.StubInfo{
		OverrideSequencers: map[string]sequencer.ValueAdvancer[int64]{
			"SetReadDeadline": &sequencer.Numeric{},
		},
		ReturnOverrides: map[string]any{
			"SetReadDeadline": []func(t time.Time) error{
				func(d time.Time) error {
					testutils.AssertTimeWas(t, d, time.Now().Add(timeout), readDeadlineTolerance).Critical()
					return stub.Conn.SetReadDeadline(d)
				},
				func(d time.Time) error {
					testutils.AssertEqual(t, d, time.Time{}).Critical()
					testutils.AssertErrorNotNil(t, err).Critical()
					return err
				},
				func(_ time.Time) error {
					t.Fatal("unexpected call")
					return nil
				},
			},
		},
	}

	return stub
}

func NewConnStubFailingToWrite(t *testing.T, conn net.Conn, err error) *StubConn {
	t.Helper()

	testutils.AssertErrorNotNil(t, err).Critical()

	return &StubConn{
		Conn: conn,
		StubInfo: testutils.StubInfo{
			ReturnOverrides: map[string]any{
				"Write": err,
			},
		},
	}
}
