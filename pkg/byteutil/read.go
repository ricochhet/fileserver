package byteutil

import "encoding/binary"

// ReadU32 reads a uint32 at the specified offset.
func ReadU32(b []byte, off *int) uint32 {
	v := binary.LittleEndian.Uint32(b[*off:])
	*off += 4

	return v
}

// ReadI32 reads an int32 at the specified offset.
func ReadI32(b []byte, off *int) int32 {
	return int32(ReadU32(b, off))
}

// ReadU64 reads a uint64 at the specified offset.
func ReadU64(b []byte, off *int) uint64 {
	v := binary.LittleEndian.Uint64(b[*off:])
	*off += 8

	return v
}

// ReadI64 reads an int64 at the specified offset.
func ReadI64(b []byte, off *int) int64 {
	return int64(ReadU64(b, off))
}

// ReadU16 reads a uint16 at the specified offset.
func ReadU16(b []byte, off *int) uint16 {
	v := binary.LittleEndian.Uint16(b[*off:])
	*off += 2

	return v
}

// ReadI16 reads an int16 at the specified offset.
func ReadI16(b []byte, off *int) int16 {
	return int16(ReadU16(b, off))
}

// ReadU8 reads a uint8 at the specified offset.
func ReadU8(b []byte, off *int) uint8 {
	v := b[*off]
	*off++

	return v
}

// ReadI8 reads an int8 at the specified offset.
func ReadI8(b []byte, off *int) int8 {
	return int8(ReadU8(b, off))
}

// ReadStringUnicode reads a UTF-16LE encoded string of the specified character length.
func ReadStringUnicode(b []byte, off *int, chars int) string {
	bytes := chars * 2
	raw := b[*off : *off+bytes]
	*off += bytes

	runes := make([]rune, chars)
	for i := range chars {
		runes[i] = rune(binary.LittleEndian.Uint16(raw[i*2:]))
	}

	return string(runes)
}
