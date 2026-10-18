package testnetwork

import (
	"fmt"
	"net"
	"time"

	"github.com/arlogy/deltapilot/internal/testutils"
)

type StubConn struct {
	net.Conn
	testutils.StubInfo
}

func (c *StubConn) SetReadDeadline(t time.Time) error {
	methodKey := "SetReadDeadline"

	v, ok := c.ReturnOverrides[methodKey]
	if !ok {
		return c.Conn.SetReadDeadline(t)
	}

	if v == nil { // untyped nil
		return nil
	}

	if err, ok := v.(error); ok {
		return err
	}

	if fn, ok := v.(func(time.Time) error); ok {
		return fn(t)
	}

	if fns, ok := v.([]func(time.Time) error); ok {
		if seq, ok := c.OverrideSequencers[methodKey]; ok && seq != nil {
			if i, err := seq.GetValue(); err == nil && i >= 0 && i < int64(len(fns)) {
				if j, err := seq.Next(); err == nil && j == i+1 {
					return fns[i](t)
				}
			}
		}
	}

	panic(fmt.Sprintf("no valid return override for %s", methodKey))
}

func (c *StubConn) Write(b []byte) (int, error) {
	methodKey := "Write"

	v, ok := c.ReturnOverrides[methodKey]
	if !ok {
		return c.Conn.Write(b)
	}

	if v == nil { // untyped nil
		return 0, nil
	}

	if err, ok := v.(error); ok {
		return 0, err
	}

	if fn, ok := v.(func([]byte) (int, error)); ok {
		return fn(b)
	}

	if fns, ok := v.([]func([]byte) (int, error)); ok {
		if seq, ok := c.OverrideSequencers[methodKey]; ok && seq != nil {
			if i, err := seq.GetValue(); err == nil && i >= 0 && i < int64(len(fns)) {
				if j, err := seq.Next(); err == nil && j == i+1 {
					return fns[i](b)
				}
			}
		}
	}

	panic(fmt.Sprintf("no valid return override for %s", methodKey))
}
