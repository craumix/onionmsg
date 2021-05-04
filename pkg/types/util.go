package types

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
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
