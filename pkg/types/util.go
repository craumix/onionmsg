package types

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

const (
	PubContPort = 10050
	PubConvPort = 10051

	blocksize = 1 << 19 // 512K
)

type ContactRequest struct {
	RemoteFP string
	LocalFP  string
	ID       uuid.UUID
}

type ContactResponse struct {
	ConvFP string
	Sig    []byte
}

func RandomString(size int) string {
	r := make([]byte, size)
	rand.Read(r)
	return base64.RawStdEncoding.EncodeToString(r)
}

/*https://stackoverflow.com/questions/23057785/how-to-copy-a-map/23058707#23058707*/
func CopyMap(m map[string]time.Time) map[string]time.Time {
    cp := make(map[string]time.Time)
    for k, v := range m {
        cp[k] = v
    }

    return cp
}