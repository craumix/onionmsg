package types

import (
	"time"

	"github.com/google/uuid"
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

func (r ContactRequest) GenerateResponse(cID ContactIdentity) (ContactResponse, RoomRequest, error) {
	remoteID, _ := NewRemoteIdentity(r.LocalFP)
	remoteID.SetAdmin(true)

	convID := NewSelfIdentity()

	sig := cID.Sign(append([]byte(convID.Fingerprint()), r.ID[:]...))

	return ContactResponse{
			ConvFP: convID.Fingerprint(),
			Sig:    sig,
		}, RoomRequest{
			Room: Room{
				Self:      &convID,
				Peers:     []*MessagingPeer{NewMessagingPeer(remoteID)},
				ID:        r.ID,
				SyncState: make(SyncMap),
			},
			ViaFingerprint: cID.Fingerprint(),
			ID:             uuid.New(),
		}, nil
}
