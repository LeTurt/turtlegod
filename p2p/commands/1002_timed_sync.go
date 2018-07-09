package commands

import (
	"strconv"
	"encoding/hex"
	"encoding/binary"
)

const BIN_KV_SERIALIZE_TYPE_INT64 uint8 = 1;
const BIN_KV_SERIALIZE_TYPE_INT32 uint8 = 2;
const BIN_KV_SERIALIZE_TYPE_INT16 uint8 = 3;
const BIN_KV_SERIALIZE_TYPE_INT8 uint8 = 4;
const BIN_KV_SERIALIZE_TYPE_UINT64 uint8 = 5;
const BIN_KV_SERIALIZE_TYPE_UINT32 uint8 = 6;
const BIN_KV_SERIALIZE_TYPE_UINT16 uint8 = 7;
const BIN_KV_SERIALIZE_TYPE_UINT8 uint8 = 8;
const BIN_KV_SERIALIZE_TYPE_DOUBLE uint8 = 9;
const BIN_KV_SERIALIZE_TYPE_STRING uint8 = 10;
const BIN_KV_SERIALIZE_TYPE_BOOL uint8 = 11;
const BIN_KV_SERIALIZE_TYPE_OBJECT uint8 = 12;
const BIN_KV_SERIALIZE_TYPE_ARRAY uint8 = 13;
const BIN_KV_SERIALIZE_FLAG_ARRAY uint8 = 0x80;

type KeyValue struct {
	key   string
	value interface{}
}

type CmdTimedSync struct {
	currentHeight uint32
	hash          []uint8
	hashStr       string
}

func unpackVarInt(data []byte) (uint64, int) {
	size := data[0] & 0x03
	switch size {
	case 0:
		value := data[0] >> 2
		return uint64(value), 1
	case 1:
		value := data[0]
		value |= data[1] << 8;
		value = value >> 2
		return uint64(value), 2
	case 2:
		value := data[0]
		value |= data[1] << 8;
		value |= data[2] << 16;
		value |= data[3] << 24;
		value = value >> 2
		return uint64(value), 4
	default:
		value := data[0]
		value |= data[1] << 8;
		value |= data[2] << 16;
		value |= data[3] << 24;
		value |= data[4] << 32;
		value |= data[5] << 40;
		value |= data[6] << 48;
		value |= data[7] << 56;
		value = value >> 2
		return uint64(value), 8
	}
	//number of consumed bytes = second return value
}

//name is a special case, where it always has a single byte for size (number of chars to follow)
func readName(data []byte) (string, int) {
	size := uint8(data[0])
	name := string(data[1 : size+1])
	return name, int(size) + 1
}

func readValue(data []byte) (interface{}, int) {
	typeId := data[0]
	switch typeId {
	case BIN_KV_SERIALIZE_TYPE_OBJECT:
		kvs, bytesRead := readSection(data[1:])
		return kvs, bytesRead + 1
	case BIN_KV_SERIALIZE_TYPE_UINT32:
		value := binary.LittleEndian.Uint32(data[1:5])
		return value, 5
	case BIN_KV_SERIALIZE_TYPE_STRING:
		size, bytesRead := unpackVarInt(data[1:])
		//again assume string fits in positive integer
		sizeI := int(size)
		start := 1+bytesRead
		end := start + sizeI
		hash := data[start:end]
		value := string(hash)
		return value, end
	}
	return nil, 0
}

//a section has N objects, and the N is always the first value as varInt, followed by section name as "Name" type.
//followed by the N objects, each with a type byte, followed by their specific bytes identified by the type byte
//the protocol seems to always have a single root object, so this expects the root to be a section of size 1
func readSection(data []byte) (map[string]interface{}, int) {
	totalBytes := 0
	count, bytesRead := unpackVarInt(data)
	totalBytes += bytesRead
	//move slice forward by number of consumed bytes
	data = data[bytesRead:]
	items := make(map[string]interface{})
	//assuming there will be no more values in a section than range of positive integer
	for i := 0; i < int(count); i++ {
		name, bytesRead := readName(data)
		data = data[bytesRead:]
		totalBytes += bytesRead

		i, bytesRead := readValue(data)
		data = data[bytesRead:]
		totalBytes += bytesRead

		items[name] = i
	}
	return items, totalBytes
}

func parse1002(data []byte) CmdTimedSync {
	//the protocol is parsed in KVBinaryInputStreamSerializer.parseBinary() in TC code
	//it seems to always start with a single "section". So read the section size and name first
	kvs, _ := readSection(data)
	if len(kvs) != 1 {
		panic("Expected 1 root object, got " + strconv.Itoa(len(kvs)))
	}

	currentHeight := kvs["current_height"].(uint32)
	topId := kvs["top_id"].([]byte)
	hashStr := hex.EncodeToString(topId)

	return CmdTimedSync{currentHeight, topId, hashStr}
}
