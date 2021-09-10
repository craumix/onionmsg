package types

import "github.com/google/uuid"

type RoomRequest struct {
	Room           Room      `json:"room"`
	ViaFingerprint string    `json:"via"`
	ID             uuid.UUID `json:"uuid"`
}
