package parser

import (
	"encoding/binary"
	"math"
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

func UnpackP2PVarInt(data []byte) (uint64, int) {
	size := data[0] & 0x03
	switch size {
	case 0:
		value := data[0] >> 2
		return uint64(value), 1
	case 1:
		value := uint64(data[0])
		value |= uint64(data[1]) << 8;
		value = value >> 2
		return uint64(value), 2
	case 2:
		value := uint64(data[0])
		value |= uint64(data[1]) << 8;
		value |= uint64(data[2]) << 16;
		value |= uint64(data[3]) << 24;
		value = value >> 2
		return uint64(value), 4
	default:
		value := uint64(data[0])
		value |= uint64(data[1]) << 8;
		value |= uint64(data[2]) << 16;
		value |= uint64(data[3]) << 24;
		value |= uint64(data[4]) << 32;
		value |= uint64(data[5]) << 40;
		value |= uint64(data[6]) << 48;
		value |= uint64(data[7]) << 56;
		value = value >> 2
		return uint64(value), 8
	}
	//number of consumed bytes = second return value
}

//name is a special case, where it always has a single byte for size (number of chars to follow)
func ReadName(data []byte) (string, int) {
	size := uint8(data[0])
	name := string(data[1 : size+1])
	return name, int(size) + 1
}

func ReadValue(data []byte, typeId uint8) (interface{}, int) {
	if typeId & BIN_KV_SERIALIZE_FLAG_ARRAY != 0 {
		typeId = typeId &^ BIN_KV_SERIALIZE_FLAG_ARRAY
		items, readBytes := ReadArray(data, typeId)
		return items, readBytes
	}
	switch typeId {
	case BIN_KV_SERIALIZE_TYPE_BOOL:
		value := int8(data[0])
		return value > 0, 1
	case BIN_KV_SERIALIZE_TYPE_INT8:
		value := int8(data[0])
		return value, 1
	case BIN_KV_SERIALIZE_TYPE_INT16:
		value := binary.LittleEndian.Uint16(data[0:2])
		//TODO: add tests to see signed ints are ok for all sizes
		return int16(value), 2
	case BIN_KV_SERIALIZE_TYPE_INT32:
		value := binary.LittleEndian.Uint32(data[0:4])
		return int32(value), 4
	case BIN_KV_SERIALIZE_TYPE_INT64:
		value := binary.LittleEndian.Uint64(data[0:8])
		return int64(value), 8
	case BIN_KV_SERIALIZE_TYPE_UINT8:
		value := uint8(data[1])
		return value, 1
	case BIN_KV_SERIALIZE_TYPE_UINT16:
		value := binary.LittleEndian.Uint16(data[0:2])
		return value, 2
	case BIN_KV_SERIALIZE_TYPE_UINT32:
		value := binary.LittleEndian.Uint32(data[0:4])
		return value, 4
	case BIN_KV_SERIALIZE_TYPE_UINT64:
		value := binary.LittleEndian.Uint64(data[0:8])
		return value, 8
	case BIN_KV_SERIALIZE_TYPE_DOUBLE:
		//double in cryptonote is 8 bytes, checked with compiling and running sizeof(double)
		//parsing: https://stackoverflow.com/questions/22491876/convert-byte-array-uint8-to-float64-in-golang#22492518
		bits := binary.LittleEndian.Uint64(data[0:8])
		value := math.Float64frombits(bits)
		return value, 8
	case BIN_KV_SERIALIZE_TYPE_OBJECT:
		kvs, bytesRead := ReadSection(data)
		return kvs, bytesRead
	case BIN_KV_SERIALIZE_TYPE_STRING:
		hash, bytesRead := ReadString(data)
		//		value := hex.EncodeToString(hash)
		return hash, bytesRead
	case BIN_KV_SERIALIZE_TYPE_ARRAY:
		items, readBytes := ReadArray(data, typeId)
		return items, readBytes
	}
	//TODO: panic
	return nil, 0
}

func ReadString(data []byte) ([]byte, int) {
	size, bytesRead := UnpackP2PVarInt(data)
	//again assume string fits in positive integer
	sizeI := int(size)
	start := bytesRead
	end := start + sizeI
	hash := data[start:end]
	return hash, end
}

func ReadArray(data []byte, typeId uint8) (interface{}, int) {
	totalBytes := 0
	size, readBytes := UnpackP2PVarInt(data);
	totalBytes += readBytes
	items := make([]interface{}, size)
	data = data[readBytes:]
	//assume array size is less than max positive integer
	for i := 0 ; i < int(size) ; i++ {
		value, readBytes := ReadValue(data, typeId)
		items = append(items, value)
		totalBytes += readBytes
		data = data[readBytes:]
	}
	return items, totalBytes
}

//a section has N objects, and the N is always the first value as varInt, followed by section name as "Name" type.
//followed by the N objects, each with a type byte, followed by their specific bytes identified by the type byte
//the protocol seems to always have a single root object, so this expects the root to be a section of size 1
func ReadSection(data []byte) (map[string]interface{}, int) {
	totalBytes := 0
	count, bytesRead := UnpackP2PVarInt(data)
	totalBytes += bytesRead
	//move slice forward by number of consumed bytes
	data = data[bytesRead:]
	items := make(map[string]interface{})
	//assuming there will be no more values in a section than range of positive integer
	for i := 0; i < int(count); i++ {
		name, bytesRead := ReadName(data)
		data = data[bytesRead:]
		totalBytes += bytesRead

		typeId := data[0]
		data = data[1:]
		i, bytesRead := ReadValue(data, typeId)
		data = data[bytesRead:]
		totalBytes += bytesRead

		items[name] = i
	}
	return items, totalBytes
}

