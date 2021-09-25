package tor

import (
	"crypto/rand"
	"encoding/base64"
)

func prngString(size int) string {
	r := make([]byte, size)
	rand.Read(r)
	return base64.RawStdEncoding.EncodeToString(r)
}