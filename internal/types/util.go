package types

import (
	"time"

	"github.com/google/uuid"
)

const (
	PubContPort = 10050
	PubConvPort = 10051
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

func (r ContactRequest) GenerateResponse(cID Identity) (ContactResponse, RoomRequest, error) {
	remoteID, _ := NewIdentity(Remote, r.LocalFP)
	remoteID.Meta.Admin = true

	convID, _ := NewIdentity(Self, "")

	sig, err := cID.Sign(append([]byte(convID.Fingerprint()), r.ID[:]...))
	if err != nil {
		return ContactResponse{}, RoomRequest{}, err
	}

	return ContactResponse{
			ConvFP: convID.Fingerprint(),
			Sig:    sig,
		}, RoomRequest{
			Room: Room{
				Self:      convID,
				Peers:     []*MessagingPeer{NewMessagingPeer(remoteID)},
				ID:        r.ID,
				SyncState: make(SyncMap),
			},
			ViaFingerprint: cID.Fingerprint(),
			ID:             uuid.New(),
		}, nil
}
