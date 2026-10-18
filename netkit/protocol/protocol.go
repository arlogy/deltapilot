package protocol

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/arlogy/deltapilot/internal/process"
	"github.com/arlogy/deltapilot/internal/stream"
)

const MaxReadMessageBytes = 1024 * 1024 * 1024 // 1 GiB

// MessageEndByte marks the end of a message, allowing message completeness to be determined rather than
// assumed when the read limit is reached.
const MessageEndByte = process.EndOfTransmission

// ReadFromConnection reads a message from a connection until the message is complete or an error occurs. It
// must not be called concurrently on the same connection because it modifies the connection's read deadline
// and consumes data from it. Concurrent calls on different connections are safe.
//
// The function protects against the following scenarios.
//   - BlockingRead: data is never sent or arrives slowly enough to block the read indefinitely.
//   - OversizedRead: an unexpectedly large message is read without a size limit.
//
// The function does not protect against ConcurrentFlooding.
//   - ConcurrentFlooding: enough concurrent read operations are performed to exhaust available resources.
//   - This can be prevented by limiting the number of concurrent calls based on available resources.
func ReadFromConnection(
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
) (retData []byte, retErr error) {
	if maxMsgBytes < 0 {
		return nil, errors.New("max bytes per message must be non-negative")
	}

	// allow reasonably large maxMsgBytes values while preventing overflow on both 32-bit and 64-bit systems
	if maxMsgBytes > MaxReadMessageBytes {
		return nil, fmt.Errorf("max bytes per message must be at most %d", MaxReadMessageBytes)
	}

	if readTimeout < 0 {
		return nil, fmt.Errorf("read timeout must be non-negative")
	}

	// set a read deadline to limit BlockingRead to readTimeout
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %v", err)
	}
	defer func() {
		// clear the read deadline before returning control to caller
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			if retErr == nil {
				retErr = fmt.Errorf("failed to unset read deadline: %v", err)
			} else {
				retErr = fmt.Errorf("%v\nfailed to unset read deadline: %v", retErr, err)
			}
		}
	}()

	// use a reader with a fixed byte limit to prevent OversizedRead
	//     - this reader never reads beyond its limit and returns EOF if the available data exceeds it
	// the read limit includes two extra bytes:
	//     - the first is for MessageEndByte
	//     - the second is for detecting messages exceeding maxMsgBytes; otherwise, if the read limit were
	//       maxMsgBytes+1, an oversized message whose first maxMsgBytes+1 bytes end with MessageEndByte would
	//       be silently truncated
	limitReader := io.LimitReader(conn, int64(maxMsgBytes)+2)

	// read until and including MessageEndByte, or fail with an error (possibly EOF)
	// note: we observed that bufReader.ReadBytes() reads beyond the first occurrence of its input delimiter,
	//       up to the limit configured for limitReader, when the delimiter is found
	bufReader := bufio.NewReader(limitReader)
	bufData, err := bufReader.ReadBytes(MessageEndByte)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("message end marker not found")
		}

		return nil, fmt.Errorf("message read failed: %v", err)
	}

	bufLen := len(bufData)
	if bufLen == maxMsgBytes+2 {
		return nil, fmt.Errorf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
	}

	if bufLen == maxMsgBytes+1 && bufReader.Buffered() == 1 {
		return nil, fmt.Errorf("received message exceeds the allowed %d-byte size limit", maxMsgBytes)
	}

	bufData = bufData[:bufLen-1] // trim MessageEndByte
	return bufData, nil
}

// WriteToConnection writes data to a connection, followed by MessageEndByte. It is not intended to be called
// concurrently on the same connection. Concurrent calls on different connections are safe.
func WriteToConnection(conn net.Conn, data []byte) error {
	// we write MessageEndByte separately to avoid allocating and copying a potentially large buffer or
	// modifying the original data

	if err := stream.WriteExactBytes(conn, data); err != nil {
		return fmt.Errorf("data write failed: %v", err)
	}

	if err := stream.WriteExactBytes(conn, []byte{MessageEndByte}); err != nil {
		return fmt.Errorf("data write failed: %v", err)
	}

	return nil
}
