package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

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
			ids = append(ids, msg.Content.Blob.ID)
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
	challenge, err := conn.ReadBytes()
	if err != nil {
		return err
	}

	signed, err := id.Sign(challenge)
	if err != nil {
		fmt.Print(err.Error())
	}

	conn.WriteString(id.Fingerprint())
	conn.WriteBytes(signed)
	conn.Flush()

	return nil
}

func Sign(key ed25519.PrivateKey, data []byte) []byte {
	return ed25519.Sign(key, data)
}

func Fingerprint(key ed25519.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(key)
}

func init() {
	err := RegisterRoomCommands()
	if err != nil {
		log.WithError(err).Warn()
	}
}
