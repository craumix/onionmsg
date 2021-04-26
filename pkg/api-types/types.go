package types

type StatusResponse struct {
	Status string `json:"status"`
}

type TorlogResponse struct {
	Log string `json:"log"`
}

type ListContactIDsResponse []string

type ListRoomsResponse []string

type AddContactIDResponse struct {
	Fingerprint string `json:"fingerprint"`
}

type CreateRoomRequest []string

type SendMessageRequest []byte
