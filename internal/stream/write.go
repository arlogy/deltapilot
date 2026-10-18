package stream

import (
	"io"
	"net"
)

// WriteExactBytes wraps conn.Write() and enforces the io.Writer contract by returning an error if fewer than
// len(data) bytes are written without an error. The error returned for a short write is io.ErrShortWrite.
func WriteExactBytes(conn net.Conn, data []byte) error {
	n, err := conn.Write(data)
	if err != nil {
		return err
	}
	if n < len(data) {
		return io.ErrShortWrite
	}

	return nil
}
