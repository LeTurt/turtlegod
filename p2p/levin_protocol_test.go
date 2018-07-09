package p2p

import (
	"testing"
	"encoding/hex"
	"os"
	"github.com/leturt/turtlegod/p2p/commands"
)

func TestHeaderSerialize(t *testing.T) {
	var commandId uint32 = 1
	data := make([]byte, 5)
	headerBytes := createHeader(commandId, data, false)
	headerStr := hex.EncodeToString(headerBytes)
	println(headerStr)
}

func TestCmd1002(t *testing.T) {
	dataFile, err := os.Open("commands/1002.bin")
	if err != nil {
		panic(err)
	}
	defer dataFile.Close()
	cmd := parseLevinHeader(dataFile)
	print(cmd.Command)
	commands.ParseCmd(cmd)
}