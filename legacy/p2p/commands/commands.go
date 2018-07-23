package commands

import "encoding/binary"

type LevinCommand struct {
	Command        uint32
	IsNotification bool
	IsResponse     bool
	Data           []byte
}

var SIGA uint32 = 0x01011101
var SIGB uint32 = 0x01020101
var VER uint8 = 0x01

func ParseCmd(cmd LevinCommand) {
	data := cmd.Data
	//TODO: size check
	sigA := binary.LittleEndian.Uint32(data[:4])
	if sigA != SIGA {
		panic("Invalid storage signature A")
	}
	sigB := binary.LittleEndian.Uint32(data[4:8])
	if sigB != SIGB {
		panic("Invalid storage signature B")
	}
	ver := data[8]
	if ver != VER {
		panic("Invalid storage version")
	}

	switch cmd.Command {
	case 1001:
//		parse1001(cmd.Data[9:])
		parse1001Reply(cmd.Data[9:])
	case 1002:
		parse1002(cmd.Data[9:])
	case 1003:
		parse1003(cmd.Data[9:])
	case 2002:
		parse2002(cmd.Data[9:])
	case 2003:
		parse2003(cmd.Data[9:])
	}
}
