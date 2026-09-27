package transport

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/arlogy/deltapilot/internal/process"
)

const MessageEndByte = process.EndOfTransmission

// ReadFromConnection reads a message from a connection and handles the following scenarios.
//   - BlockingRead: data is never sent or is sent slowly enough to block the read indefinitely.
//   - OversizedMessage: a message larger than expected exhausts application memory.
//
// ReadFromConnection does not protect against ConcurrentFlooding.
//   - ConcurrentFlooding: enough concurrent read operations are performed to saturate application memory.
//   - This can be prevented by limiting concurrent calls based on a desired memory limit.
func ReadFromConnection(conn net.Conn, maxMsgBytes int, readTimeout time.Duration) ([]byte, error) {
	// set a read deadline to limit BlockingRead to readTimeout
	conn.SetReadDeadline(time.Now().Add(readTimeout))

	// use a reader with a fixed byte limit to prevent OversizedMessage
	// allow two extra bytes: one for MessageEndByte and one to detect a message exceeding maxMsgBytes
	limitReader := io.LimitReader(conn, int64(maxMsgBytes)+2)

	bufReader := bufio.NewReader(limitReader)
	bufData, err := bufReader.ReadBytes(MessageEndByte)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("connection closed without end marker")
		}
		return nil, err
	}

	bufData = bytes.TrimSuffix(bufData, []byte{MessageEndByte})
	if len(bufData) == maxMsgBytes+1 {
		return nil, fmt.Errorf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
	}

	return bufData, nil
}

// WriteToConnection writes data to conn, followed by MessageEndByte.
func WriteToConnection(conn net.Conn, data []byte) error {
	if _, err := conn.Write(data); err != nil {
		return err
	}

	if _, err := conn.Write([]byte{MessageEndByte}); err != nil {
		return err
	}

	return nil
}
