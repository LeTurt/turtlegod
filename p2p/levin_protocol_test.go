package p2p

import (
	"testing"
	"encoding/hex"
)

func TestHeaderSerialize(t *testing.T) {
	var commandId uint32 = 1
	data := make([]byte, 5)
	headerBytes := createHeader(commandId, data, false)
	headerStr := hex.EncodeToString(headerBytes)
	println(headerStr)
}
