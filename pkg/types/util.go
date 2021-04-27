package types

import (
	"crypto/rand"
	"encoding/base64"
)

func RandomString(size int) string {
	r := make([]byte, size)
	rand.Read(r)
	return base64.RawStdEncoding.EncodeToString(r)
}
