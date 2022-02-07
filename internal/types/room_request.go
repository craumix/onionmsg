package types

import "github.com/google/uuid"

type RoomRequest struct {
	Room           Room        `json:"room"`
	ViaFingerprint Fingerprint `json:"via"`
	ID             uuid.UUID   `json:"uuid"`
}
