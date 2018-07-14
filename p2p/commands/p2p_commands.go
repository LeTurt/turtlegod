package commands

import (
	"strconv"
	"encoding/hex"
	"encoding/binary"
	"github.com/leturt/turtlegod/p2p/parser"
)

type CmdTimedSync struct {
	currentHeight uint32
	hash          []uint8
	hashStr       string
}

func parse1002(data []byte) CmdTimedSync {
	//the protocol is parsed in KVBinaryInputStreamSerializer.parseBinary() in TC code
	//it seems to always start with a single "section". So read the section size and name first
	kvs, _ := parser.ReadSection(data)
	if len(kvs) != 1 {
		panic("Expected 1 root object, got " + strconv.Itoa(len(kvs)))
	}

	payloadMap := kvs["payload_data"].(map[string]interface{})
	currentHeight := payloadMap["current_height"].(uint32)
	topId := payloadMap["top_id"].([]byte)
	hashStr := hex.EncodeToString(topId)

	cmd1002 := CmdTimedSync{currentHeight, topId, hashStr}
	return cmd1002
}

func parsePeerList(data []uint8) []string {
	count := len(data)/24
	peerlist := []string{} //todo: set capacity to count
	for i := 0 ; i < count ; i++ {
		ip1 := strconv.FormatUint(uint64(data[0]), 10)
		ip2 := strconv.FormatUint(uint64(data[1]), 10)
		ip3 := strconv.FormatUint(uint64(data[2]), 10)
		ip4 := strconv.FormatUint(uint64(data[3]), 10)
		ip := ip1 + "." + ip2 + "." + ip3 + "." + ip4
		port := binary.LittleEndian.Uint32(data[4:8])
		peerIdType := binary.LittleEndian.Uint64(data[8:16])
		lastSeen := binary.LittleEndian.Uint64(data[16:24])
		peerInfo := ip + ":" + strconv.FormatUint(uint64(port), 10) + " type: " + strconv.FormatUint(peerIdType, 10) +
			"seen: "+strconv.FormatUint(lastSeen, 10)
		peerlist = append(peerlist, peerInfo)
		data = data[24:]
	}
	return peerlist
}

func parse1001(data []byte) {
	kvs, _ := parser.ReadSection(data)
	peerList := kvs["local_peerlist"].([]uint8)
	peers := parsePeerList(peerList)
	print(peers)

}