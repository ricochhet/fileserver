package byteutil

import (
	"encoding/binary"
	"unicode/utf16"
)

// WriteU32 writes a uint32 at the specified offset.
func WriteU32(b []byte, off *int, v uint32) {
	binary.LittleEndian.PutUint32(b[*off:], v)
	*off += 4
}

// WriteI32 writes an int32 at the specified offset.
func WriteI32(b []byte, off *int, v int32) {
	WriteU32(b, off, uint32(v))
}

// WriteU64 writes a uint64 at the specified offset.
func WriteU64(b []byte, off *int, v uint64) {
	binary.LittleEndian.PutUint64(b[*off:], v)
	*off += 8
}

// WriteI64 writes an int64 at the specified offset.
func WriteI64(b []byte, off *int, v int64) {
	WriteU64(b, off, uint64(v))
}

// WriteU16 writes a uint16 at the specified offset.
func WriteU16(b []byte, off *int, v uint16) {
	binary.LittleEndian.PutUint16(b[*off:], v)
	*off += 2
}

// WriteI16 writes an int16 at the specified offset.
func WriteI16(b []byte, off *int, v int16) {
	WriteU16(b, off, uint16(v))
}

// WriteU8 writes a uint8 at the specified offset.
func WriteU8(b []byte, off *int, v uint8) {
	b[*off] = v
	*off++
}

// WriteI8 writes an int8 at the specified offset.
func WriteI8(b []byte, off *int, v int8) {
	WriteU8(b, off, uint8(v))
}

// WriteStringUnicode writes a UTF-16LE encoded string of the specified length.
func WriteStringUnicode(b []byte, off *int, s string) {
	u16 := utf16.Encode([]rune(s))
	for _, v := range u16 {
		binary.LittleEndian.PutUint16(b[*off:], v)
		*off += 2
	}
}
