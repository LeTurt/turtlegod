package commands

func parse2002(data []byte) {
	kvs, _ := readSection(data)
	peerList := kvs["local_peerlist"].([]uint8)
	peers := parsePeerList(peerList)
	print(peers)
}
