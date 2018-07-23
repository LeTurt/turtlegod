package parser

func UnpackCNVarIntUint8(data []byte) (uint8, int) {
	value, bytesRead := unpackCNVarInt(data, 1)
	return uint8(value), bytesRead
}

func UnpackCNVarIntUint16(data []byte) (uint16, int) {
	value, bytesRead := unpackCNVarInt(data, 2)
	return uint16(value), bytesRead
}

func UnpackCNVarIntUint32(data []byte) (uint32, int) {
	value, bytesRead := unpackCNVarInt(data, 4)
	return uint32(value), bytesRead
}

func UnpackCNVarIntUint64(data []byte) (uint64, int) {
	value, bytesRead := unpackCNVarInt(data, 8)
	return uint64(value), bytesRead
}

func unpackCNVarInt(data []byte, valuesize int) (uint64, int) {
	var value uint64
	bytesRead := 0

	for i := 0; ; i++ {
		shift := i * 7
		piece := uint64(data[0])
		bytesRead++
		maxShift := valuesize * 8 - 1
		maxValue := uint64(1) << uint8(maxShift+1)
		if valuesize == 8 {
			//need this because above shift will wrap for uint64
			maxValue = 0xFFFFFFFFFFFFFFFF
		}
		rightSide := uint64(piece & 0x7f)
		rightSide = rightSide << uint8(shift)
		value = value | rightSide
		if shift > maxShift || value > maxValue {
			panic("readVarint, value overflow");
		}
		bigger := piece & 0x80
		if bigger == 0 {
			if piece == 0 && shift != 0 {
				panic("readVarint, invalid value representation")
			}
			break
		}
		data = data[1:]
	}

	return value, bytesRead
}
