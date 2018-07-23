package commands

import (
	"strconv"
	"encoding/binary"
	"github.com/leturt/turtlegod/legacy/p2p/parser"
	"encoding/hex"
)

type PeerInfo struct {
	peerType uint64
	lastSeen uint64
	ipStr string
	ip []uint8
	port uint32
}

type CmdTimedSync struct {
	currentHeight uint32
	hash          []uint8
	hashStr       string
}

type CmdHandshake struct {
	networkId []uint8
	version uint8
	localTime uint64
	myPort uint32
	peerId uint64
	currentHeight uint32
	topId []uint8
	topIdStr string
}

type CmdHandshakeReply struct {
	networkId []uint8
	version uint8
	localTime uint64
	myPort uint32
	peerId uint64
	currentHeight uint32
	topId []uint8
	topIdStr string
	peers []PeerInfo
}

type CmdPing struct {
	//ping is just an empty request body
}

type CmdPingReply struct {
	status string
	peerId uint64
}

func parsePeerList(data []uint8) []PeerInfo {
	count := len(data)/24
	peerlist := []PeerInfo{} //todo: set capacity to count
	for i := 0 ; i < count ; i++ {
		ipBytes := []uint8{data[0], data[1], data[2], data[3]}
		ip1 := strconv.FormatUint(uint64(data[0]), 10)
		ip2 := strconv.FormatUint(uint64(data[1]), 10)
		ip3 := strconv.FormatUint(uint64(data[2]), 10)
		ip4 := strconv.FormatUint(uint64(data[3]), 10)
		ipStr := ip1 + "." + ip2 + "." + ip3 + "." + ip4
		port := binary.LittleEndian.Uint32(data[4:8])
		peerIdType := binary.LittleEndian.Uint64(data[8:16])
		lastSeen := binary.LittleEndian.Uint64(data[16:24])
		peerInfo := ipStr + ":" + strconv.FormatUint(uint64(port), 10) + " type: " + strconv.FormatUint(peerIdType, 10) +
			"seen: "+strconv.FormatUint(lastSeen, 10)
		println(peerInfo)
		pi := PeerInfo{peerIdType, lastSeen, ipStr, ipBytes, port}
		peerlist = append(peerlist, pi)
		data = data[24:]
	}
	return peerlist
}

func parse1001(data []byte) CmdHandshake {
	kvs, _ := parser.ReadSection(data)
	shake := CmdHandshake{}
	shake.currentHeight = 0

	nodeDataMap := kvs["node_data"].(map[string]interface{})
	networkId := nodeDataMap["network_id"].([]uint8)
	shake.networkId = networkId
	version := nodeDataMap["version"].(uint8)
	shake.version = version
	peerId := nodeDataMap["peer_id"].(uint64)
	shake.peerId = peerId
	localTime := nodeDataMap["local_time"].(uint64)
	shake.localTime = localTime
	myPort := nodeDataMap["my_port"].(uint32)
	shake.myPort = myPort

	payloadMap := kvs["payload_data"].(map[string]interface{})
	currentHeight := payloadMap["current_height"].(uint32)
	shake.currentHeight = currentHeight
	topId := payloadMap["top_id"].([]byte)
	shake.topId = topId
	hashStr := hex.EncodeToString(topId)
	shake.topIdStr = hashStr

	return CmdHandshake{}
}

func parse1001Reply(data []byte) CmdHandshakeReply {
	//TODO: delete this duplicate method, merge into parse1001 with check for reply
	kvs, _ := parser.ReadSection(data)
	shake := CmdHandshakeReply{}
	shake.currentHeight = 0

	nodeDataMap := kvs["node_data"].(map[string]interface{})
	networkId := nodeDataMap["network_id"].([]uint8)
	shake.networkId = networkId
	version := nodeDataMap["version"].(uint8)
	shake.version = version
	peerId := nodeDataMap["peer_id"].(uint64)
	shake.peerId = peerId
	localTime := nodeDataMap["local_time"].(uint64)
	shake.localTime = localTime
	myPort := nodeDataMap["my_port"].(uint32)
	shake.myPort = myPort

	payloadMap := kvs["payload_data"].(map[string]interface{})
	currentHeight := payloadMap["current_height"].(uint32)
	shake.currentHeight = currentHeight
	topId := payloadMap["top_id"].([]byte)
	shake.topId = topId
	hashStr := hex.EncodeToString(topId)
	shake.topIdStr = hashStr

	peerList := kvs["local_peerlist"].([]uint8)
	peers := parsePeerList(peerList)
	shake.peers = peers
	return shake
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

func parse1003(data []byte) CmdPing {
	kvs, _ := parser.ReadSection(data)
	if len(kvs) > 0 {
		panic("Expected 0 root object, got " + strconv.Itoa(len(kvs)))
	}
	return CmdPing{}
}

//TODO: test
func parse1003Reply(data []byte) CmdPingReply {
	kvs, _ := parser.ReadSection(data)
	if len(kvs) > 0 {
		panic("Expected 0 root object, got " + strconv.Itoa(len(kvs)))
	}
	status := kvs["status"].(string)
	peerId := kvs["status"].(uint64)
	return CmdPingReply{status, peerId}

}
