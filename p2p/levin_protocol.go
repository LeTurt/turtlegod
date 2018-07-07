package p2p

import (
	"encoding/binary"
	"net"
	"io"
	"encoding/hex"
)

//first 8 bytes of protocol header, identifying the protocol messages
//named Bender's nightmare in original code, apparently due to some futurama episode with the 2 in binary string
const LEVIN_SIGNATURE uint64 = 0x0101010101012101;
const LEVIN_PACKET_REQUEST uint32 = 0x00000001;
const LEVIN_PACKET_RESPONSE uint32 = 0x00000002;
//this is to indicate a protocol packet max size of 100MB (according to original comments)
const LEVIN_DEFAULT_MAX_PACKET_SIZE = 100000000;
const LEVIN_PROTOCOL_VER_1 uint32 = 1;

var conn net.Conn

type LevinHeader struct {
	Signature        uint64
	BodySize         uint64
	HaveToReturnData bool
	Command          uint32
	ReturnCode       int32
	Flags            uint32
	Version          uint32
}

type LevinCommand struct {
	Command        uint32
	IsNotification bool
	IsResponse     bool
	data           []byte
}

func headerSize() int {
	return 8 + 8 + 1 + 4 + 4 + 4 + 4 //all header fields as bytes
}

func createHeader(command uint32, data []byte, needResponse bool) []byte {
	header := LevinHeader{}
	header.Signature = LEVIN_SIGNATURE
	header.BodySize = uint64(len(data))
	header.HaveToReturnData = needResponse
	header.Command = command
	header.Version = LEVIN_PROTOCOL_VER_1
	header.Flags = LEVIN_PACKET_REQUEST

	headerBytes := make([]byte, headerSize())

	//TODO: just use index numbers instead of counting index
	index := 0
	b := make([]byte, 8)

	//index 0-7 is
	binary.LittleEndian.PutUint64(b, header.Signature)
	//	binary.BigEndian.PutUint64(b, header.Signature)
	println("header sig:", hex.EncodeToString(b))
	//https://stackoverflow.com/questions/37884361/concat-multiple-slices-in-golang
	index += copy(headerBytes[index:], b)
	//index 8-15
	binary.LittleEndian.PutUint64(b, header.BodySize)
	index += copy(headerBytes[index:], b)

	//index 16 = true if other end needs to return something
	if header.HaveToReturnData {
		//default value is 0, so if value is false do nothing
		headerBytes[index] = 1
	}
	index++

	//index 17-20 protocol command id
	binary.LittleEndian.PutUint32(b, header.Command)
	index += copy(headerBytes[index:], b[:4])
	//index 21-24 is return code
	//TODO: check uint32() conversion works ok here and does not lose sign or anything else
	binary.LittleEndian.PutUint32(b, uint32(header.ReturnCode))
	index += copy(headerBytes[index:], b[:4])
	//index 25-28 is protocol flags
	binary.LittleEndian.PutUint32(b, header.Flags)
	index += copy(headerBytes[index:], b[:4])
	//index 29->32 is protocol version
	binary.LittleEndian.PutUint32(b, header.Version)
	index += copy(headerBytes[index:], b[:4])

	return headerBytes
}

func SendMessage(command uint32, data []byte, needResponse bool) {
	headerBytes := createHeader(command, data, needResponse)
	writeStrict(headerBytes)
	writeStrict(data)
}

func ReceiveMessage() {
	headerBytes := make([]byte, headerSize())
	readStrict(headerBytes)

	header := LevinHeader{}
	header.Signature = binary.LittleEndian.Uint64(headerBytes[:8])
	header.BodySize = binary.LittleEndian.Uint64(headerBytes[8:16])
	header.HaveToReturnData = (headerBytes[16] != 0)
	header.Command = binary.LittleEndian.Uint32(headerBytes[17:21])
	//TODO: check int32() conversion does not lose anything here
	header.ReturnCode = int32(binary.LittleEndian.Uint32(headerBytes[21:25]))
	header.Flags = binary.LittleEndian.Uint32(headerBytes[25:29])
	header.Version = binary.LittleEndian.Uint32(headerBytes[29:33])

	//TODO: check max body size
	data := make([]byte, header.BodySize)
	readStrict(data)

	cmd := LevinCommand{}
	cmd.Command = header.Command
	cmd.data = data
	cmd.IsNotification = !header.HaveToReturnData
	cmd.IsResponse = (header.Flags & LEVIN_PACKET_RESPONSE) == LEVIN_PACKET_RESPONSE
}

func writeStrict(data []byte) {
	n, err := conn.Write(data)
	println("wrote:", n, "bytes")
	if err != nil {
		println("error writing data", err)
	}
}

func readStrict(data []byte) bool {
	//connection is built in p2pnode.cpp
	//https://stackoverflow.com/questions/24339660/read-whole-data-with-golang-net-conn-read#24343240
	//	b := make([]byte, headerSize())
	n, err := io.ReadFull(conn, data)
	print("read", n, "bytes")
	//ReadFull also gives ErrUnexpectedEOF error in case of too few bytes to read, so err != nil should cover all cases
	if err != nil {
		//TODO:
		return false
	}
	return true
}
