package protocol

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/arlogy/deltapilot/internal/process"
	"github.com/arlogy/deltapilot/internal/stream"
)

const MaxReadMessageBytes = 1024 * 1024 * 1024 // 1 GiB

// MessageEndByte marks the end of a message, allowing message completeness to be determined rather than
// assumed when the maximum expected message size is reached.
const MessageEndByte = process.EndOfTransmission

// ReadFromConnection reads a message from a connection until the message is complete or an error occurs. It
// must not be called concurrently on the same connection because it sets the connection's read deadline,
// consumes data from the connection, and clears the read deadline upon termination. Concurrent calls on
// different connections are safe.
//
// The function protects against the following scenarios.
//   - IndefiniteWait: data is never sent or arrives slowly enough to block the read indefinitely.
//   - UnboundedRead: an unexpectedly large message is read without a size limit.
//
// The function does not protect against ConcurrentFlooding.
//   - ConcurrentFlooding: enough concurrent read operations are performed to exhaust available resources.
//   - This can be prevented by limiting the number of concurrent calls based on available resources.
//
// The parameters are:
//   - conn: the connection to read from.
//   - maxMsgBytes: the maximum allowed message size in bytes, used to prevent UnboundedRead. Two additional
//     bytes are used internally: one for MessageEndByte and one to detect messages exceeding the limit. As a
//     result, each function call may require a buffer of up to maxMsgBytes+2 bytes.
//   - readTimeout: the maximum duration allowed for reading the message, used to prevent IndefiniteWait.
//
// The return values are:
//   - On failure, the function returns a nil byte slice and an error wrapping one of the following sentinel
//     errors: ErrInvalidArgument, ErrMessageIncomplete, ErrMessageTooLarge, ErrTimeout, or ErrReadFailed
//     (generic fallback).
//   - On success, the function returns the byte slice containing the message read and a nil error.
//
// Important notes:
//   - When an error occurs, the connection may no longer be safe to use for reading subsequent messages, as
//     unread bytes may belong to the message that failed to be read rather than the next message. Callers
//     should therefore consider closing the connection after a read failure.
//   - This function may also fail if the connection is unexpectedly closed or reset, for example, by the
//     remote endpoint when it cannot consume the entire message. On Windows, the corresponding error message
//     may be "An existing connection was forcibly closed by the remote host" or
//     "An established connection was aborted by the software in your host machine". As a result, callers
//     should ensure that no trailing bytes remain unread after this function returns.
func ReadFromConnection(
	conn net.Conn,
	maxMsgBytes int,
	readTimeout time.Duration,
) (retData []byte, retErr error) {
	if maxMsgBytes < 0 {
		return nil, fmt.Errorf("%w: max bytes per message must be non-negative", ErrInvalidArgument)
	}

	// allow reasonably large maxMsgBytes values while preventing overflow on both 32-bit and 64-bit systems
	if maxMsgBytes > MaxReadMessageBytes {
		return nil, fmt.Errorf(
			"%w: max bytes per message must be at most %d", ErrInvalidArgument, MaxReadMessageBytes,
		)
	}

	if readTimeout < 0 {
		return nil, fmt.Errorf("%w: read timeout must be non-negative", ErrInvalidArgument)
	}

	// set a read deadline to prevent IndefiniteWait
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, fmt.Errorf("%w: failed to set read deadline: %v", ErrReadFailed, err)
	}
	defer func() {
		// clear the read deadline before returning control to caller
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			retData = nil
			retErr = errors.Join(
				retErr,
				fmt.Errorf("%w: failed to unset read deadline: %v", ErrReadFailed, err),
			)
		}
	}()

	// use a reader with a fixed byte limit to prevent UnboundedRead
	//     - this reader never reads beyond its limit and returns EOF if the available data exceeds it
	// the read limit includes two extra bytes:
	//     - the first is for MessageEndByte
	//     - the second is for detecting messages exceeding maxMsgBytes; otherwise, if the read limit were
	//       maxMsgBytes+1, an oversized message whose first maxMsgBytes+1 bytes end with MessageEndByte would
	//       be silently truncated
	limitReader := io.LimitReader(conn, int64(maxMsgBytes)+2)

	// read until and including MessageEndByte, or fail with an error (possibly EOF)
	// note: we observed that bufReader.ReadBytes() reads beyond the first occurrence of its input delimiter
	//       when additional bytes are available after the delimiter, potentially reading up to the limit
	//       configured for limitReader
	bufReader := bufio.NewReader(limitReader)
	bufData, err := bufReader.ReadBytes(MessageEndByte)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w: end marker not found", ErrMessageIncomplete)
		}

		if errors.Is(err, os.ErrDeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrTimeout, err)
		}

		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	bufLen := len(bufData)
	if bufLen == maxMsgBytes+2 {
		return nil, fmt.Errorf("%w: exceeds the allowed %d-byte limit", ErrMessageTooLarge, maxMsgBytes)
	}

	if bufLen == maxMsgBytes+1 && bufReader.Buffered() == 1 {
		return nil, fmt.Errorf("%w: exceeds the allowed %d-byte limit", ErrMessageTooLarge, maxMsgBytes)
	}

	bufData = bufData[:bufLen-1] // trim MessageEndByte
	return bufData, nil
}

// WriteToConnection is a convenience helper that writes data to a connection, followed by MessageEndByte
// which ReadFromConnection uses as message delimiter. It is not intended to be called concurrently on the
// same connection, but concurrent calls on different connections are safe.
//
// The parameters are:
//   - conn: the connection to write to.
//   - data: the data to write to the connection.
//
// The return values are:
//   - On failure, the function returns an error wrapping the sentinel error ErrWriteFailed.
//   - On success, the function returns a nil error.
func WriteToConnection(conn net.Conn, data []byte) error {
	// we write MessageEndByte separately to avoid allocating and copying a potentially large buffer or
	// modifying the original data

	if err := stream.WriteExactBytes(conn, data); err != nil {
		return fmt.Errorf("%w for data: %v", ErrWriteFailed, err)
	}

	if err := stream.WriteExactBytes(conn, []byte{MessageEndByte}); err != nil {
		return fmt.Errorf("%w for end marker: %v", ErrWriteFailed, err)
	}

	return nil
}
