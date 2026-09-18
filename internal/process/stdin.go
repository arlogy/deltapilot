package process

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode"
)

const EndOfTransmission byte = 0x04 // ASCII EOT; Ctrl+D keyboard shortcut

func ReadFromStdin() ([]byte, error) {
	// read message from standard input
	//     Ctrl+D allows reading up to EndOfTransmission; Ctrl+Z produces io.EOF
	fmt.Println("Enter a message; press Ctrl+D or Ctrl+Z to finish entering, or Ctrl+C to abandon")
	reader := bufio.NewReader(os.Stdin)
	msgData, err := reader.ReadBytes(EndOfTransmission)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	msgData = bytes.TrimSuffix(msgData, []byte{EndOfTransmission})

	// detect empty message, including on Ctrl+C
	if len(msgData) == 0 {
		return nil, errors.New("no data available")
	}

	// normalize the message read from standard input
	msgData = bytes.ReplaceAll(msgData, []byte("\r\n"), []byte("\n"))
	msgData = bytes.ReplaceAll(msgData, []byte("\r"), []byte("\n"))
	msgData = bytes.TrimRightFunc(msgData, unicode.IsSpace)

	return msgData, nil
}
