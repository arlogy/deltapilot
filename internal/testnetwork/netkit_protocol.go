package testnetwork

import (
	"bytes"
	"testing"

	"github.com/arlogy/deltapilot/internal/random"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/netkit/protocol"
)

func GenerateSizedNetProtocolMessage(t *testing.T, n int) []byte {
	t.Helper()

	data := testutils.GenerateNBytes(t, n)

	// remove message end marker; only when n != 0, to avoid bytes.ReplaceAll() returning nil
	if n != 0 {
		data = bytes.ReplaceAll(data, []byte{protocol.MessageEndByte}, []byte{0x05})
	}

	return data
}

func GenerateNetProtocolMessage(t *testing.T) []byte {
	t.Helper()

	return GenerateSizedNetProtocolMessage(t, random.RandomSequenceLen)
}
