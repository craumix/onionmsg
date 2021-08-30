package types

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"
)

const (
	PubContPort = 10050
	PubConvPort = 10051

	blocksize = 1 << 19 // 512K
)

type SyncMap map[string]time.Time

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

func CopySyncMap(m SyncMap) SyncMap {
	cp := make(SyncMap)
	for k, v := range m {
		cp[k] = v
	}

	return cp
}

func SyncMapsEqual(map1, map2 SyncMap) bool {
	for k, v := range map1 {
		if t, ok := map2[k]; !ok || !t.Equal(v) {
			return false
		}
	}

	return true
}

func blobIDsFromMessages(msgs ...Message) []uuid.UUID {
	ids := make([]uuid.UUID, 0)

	for _, msg := range msgs {
		if msg.ContainsBlob() {
			ids = append(ids, msg.Content.Meta.BlobUUID)
		}
	}

	return ids
}

func expectResponse(conn connection.ConnWrapper, expResp string) error {
	resp, err := conn.ReadString()
	if err != nil {
		return err
	} else if resp != expResp {
		return fmt.Errorf("received response \"%s\" wanted \"%s\"", resp, expResp)
	}

	return nil
}

func fingerprintChallenge(conn connection.ConnWrapper, id Identity) error {
	challenge, err := conn.ReadBytes(false)
	if err != nil {
		return err
	}

	conn.WriteString(id.Fingerprint())
	conn.WriteBytes(id.Sign(challenge), false)
	conn.Flush()

	return nil
}
