package cryptoutil

import "encoding/base64"

// EncodeB64 encodes the byte slice into a base64 string.
func EncodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeB64 decodes the string into a byte slice.
func DecodeB64(s string) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(s)
}
