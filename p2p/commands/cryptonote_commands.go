package commands

import "github.com/leturt/turtlegod/p2p/parser"

func parse2002(data []byte) {
	kvs, _ := parser.ReadSection(data)
	peerList := kvs["local_peerlist"].([]uint8)
	peers := parsePeerList(peerList)
	print(peers)
}
