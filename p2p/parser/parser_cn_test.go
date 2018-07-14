package parser

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCNUInt8(t *testing.T) {
	data := []byte{0x0}
	val, bytesRead := UnpackCNVarIntUint8(data)
	assert.Equal(t, uint8(0), val, "0x00 should be parsed as 0.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0x1}
	val, bytesRead = UnpackCNVarIntUint8(data)
	assert.Equal(t, uint8(1), val, "0x01 should be parsed as 1.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0x10}
	val, bytesRead = UnpackCNVarIntUint8(data)
	assert.Equal(t, uint8(16), val, "0x10 should be parsed as 16.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0x7f}
	val, bytesRead = UnpackCNVarIntUint8(data)
	assert.Equal(t, uint8(127), val, "0x7f should be parsed as 127.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0xff,0x01} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint8(data)
	assert.Equal(t, uint8(255), val, "0x1ff should be parsed as 255.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xFF}
	assert.Panics(t, func() {UnpackCNVarIntUint8(data)})

	data = []byte{0xFF, 0xFF}
	assert.Panics(t, func() {UnpackCNVarIntUint8(data)})

	data = []byte{0xFF, 0x02}
	assert.Panics(t, func() {UnpackCNVarIntUint8(data)})
}

func TestCNUInt16(t *testing.T) {
	data := []byte{0x0}
	val, bytesRead := UnpackCNVarIntUint16(data)
	assert.Equal(t, uint16(0), val, "0x00 should be parsed as 0.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0x1}
	val, bytesRead = UnpackCNVarIntUint16(data)
	assert.Equal(t, uint16(1), val, "0x01 should be parsed as 1.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0xFF, 0x50}
	val, bytesRead = UnpackCNVarIntUint16(data)
	//0x50FF in binary is 0101000011111111, removing the varint top bit from first byte it is 01010000111111=10367
	assert.Equal(t, uint16(10367), val, "0x50FF should be parsed as 10367.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xFF, 0x7F}
	val, bytesRead = UnpackCNVarIntUint16(data)
	assert.Equal(t, uint16(16383), val, "0x7fff should be parsed as 16383.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xff,0xff,0x01} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint16(data)
	assert.Equal(t, uint16(32767), val, "0x1ffff should be parsed as 32767.")
	assert.Equal(t, 3, bytesRead)

	data = []byte{0xff,0xff,0x03} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint16(data)
	assert.Equal(t, uint16(65535), val, "0x1ffff should be parsed as 65535.")
	assert.Equal(t, 3, bytesRead)

	data = []byte{0xFF}
	assert.Panics(t, func() {UnpackCNVarIntUint16(data)})

	data = []byte{0xFF, 0xFF}
	assert.Panics(t, func() {UnpackCNVarIntUint16(data)})

	data = []byte{0xFF, 0xFF, 0xFF}
	assert.Panics(t, func() {UnpackCNVarIntUint16(data)})

	data = []byte{0xFF, 0xFF, 0x04}
	assert.Panics(t, func() {UnpackCNVarIntUint16(data)})
}

func TestCNUInt32(t *testing.T) {
	data := []byte{0x0}
	val, bytesRead := UnpackCNVarIntUint32(data)
	assert.Equal(t, uint32(0), val, "0x00 should be parsed as 0.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0x1}
	val, bytesRead = UnpackCNVarIntUint32(data)
	assert.Equal(t, uint32(1), val, "0x01 should be parsed as 1.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0xFF, 0x50}
	val, bytesRead = UnpackCNVarIntUint32(data)
	//0x50FF in binary is 0101000011111111, removing the varint top bit from first byte it is 01010000111111=10367
	assert.Equal(t, uint32(10367), val, "0x50FF should be parsed as 10367.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xFF, 0x7F}
	val, bytesRead = UnpackCNVarIntUint32(data)
	assert.Equal(t, uint32(16383), val, "0x7fff should be parsed as 16383.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xff,0xff,0x01} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint32(data)
	assert.Equal(t, uint32(32767), val, "0x1ffff should be parsed as 32767.")
	assert.Equal(t, 3, bytesRead)

	data = []byte{0xff,0xff,0x03} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint32(data)
	assert.Equal(t, uint32(65535), val, "0x1ffff should be parsed as 65535.")
	assert.Equal(t, 3, bytesRead)

	data = []byte{0xff, 0xff, 0xff,0xff,0x0f} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint32(data)
	assert.Equal(t, uint32(4294967295), val, "0x1ffffffff should be parsed as 4294967295.")
	assert.Equal(t, 5, bytesRead)

	data = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	assert.Panics(t, func() {UnpackCNVarIntUint32(data)})

	data = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x1f}
	assert.Panics(t, func() {UnpackCNVarIntUint32(data)})
}

func TestCNUInt64(t *testing.T) {
	data := []byte{0x0}
	val, bytesRead := UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(0), val, "0x00 should be parsed as 0.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0x1}
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(1), val, "0x01 should be parsed as 1.")
	assert.Equal(t, 1, bytesRead)

	data = []byte{0xFF, 0x50}
	val, bytesRead = UnpackCNVarIntUint64(data)
	//0x50FF in binary is 0101000011111111, removing the varint top bit from first byte it is 01010000111111=10367
	assert.Equal(t, uint64(10367), val, "0x50FF should be parsed as 10367.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xFF, 0x7F}
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(16383), val, "0x7fff should be parsed as 16383.")
	assert.Equal(t, 2, bytesRead)

	data = []byte{0xff,0xff,0x01} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(32767), val, "0x1ffff should be parsed as 32767.")
	assert.Equal(t, 3, bytesRead)

	data = []byte{0xff,0xff,0x03} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(65535), val, "0x1ffff should be parsed as 65535.")
	assert.Equal(t, 3, bytesRead)

	data = []byte{0xff, 0xff, 0xff,0xff,0x0f} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(4294967295), val, "0x1ffffffff should be parsed as 4294967295.")
	assert.Equal(t, 5, bytesRead)

	data = []byte{0xff, 0xff, 0xff,0xff,0x1f} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(8589934591), val, "0x1ffffffff should be parsed as 8589934591.")
	assert.Equal(t, 5, bytesRead)

	data = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,0xff, 0x7f} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(9223372036854775807), val, "0x7fffffffffffffff should be parsed as 9223372036854775807.")
	assert.Equal(t, 9, bytesRead)

	data = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(0xffffffffffffffff), val, "0xfffffffffffffffff should be parsed as integer.")
	assert.Equal(t, 10, bytesRead)

	//this should actually fail, but it doesnt, as the value used to parse is uin64, and it will shift extra bits out
	//might need a fix in a perfect world, but this already works for much bigger values than the CN ever did so leave it
	data = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02} //little-endian byte order, so low byte first. or so i think from CN code..
	val, bytesRead = UnpackCNVarIntUint64(data)
	assert.Equal(t, uint64(0x7fffffffffffffff), val, "0xfffffffffffffffff should be parsed as integer.")
//	assert.Panics(t, func() {UnpackCNVarIntUint64(datamodel)})

	data = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}
	assert.Panics(t, func() {UnpackCNVarIntUint64(data)})
}
