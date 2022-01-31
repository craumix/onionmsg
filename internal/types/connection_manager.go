package types

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/net/proxy"
	"net"
	"strconv"
)

type StatusMessage string

const (
	AuthOK     = "auth_ok"
	AuthFailed = "auth_failed"

	MessagesOK = "messages_ok"
	SyncOK     = "sync_ok"
	BlockOK    = "block_ok"
	BlobOK     = "blob_ok"

	MessageSigInvalid = "message_sig_invalid"
	MalformedUUID     = "malformed_uuid"

	blocksize = 1 << 19 // 512K
)

type ConnectionManager struct {
	proxy       proxy.Dialer
	blobManager blobmngr.ManagesBlobs
}

type MessageConnection struct {
	conn sio.ConnWrapper
}

func NewConnectionManager(proxy proxy.Dialer, blobManager blobmngr.ManagesBlobs) ConnectionManager {
	return ConnectionManager{
		proxy:       proxy,
		blobManager: blobManager,
	}
}

func (m ConnectionManager) UseConnection(conn net.Conn) MessageConnection {
	return MessageConnection{
		conn: sio.WrapConnection(conn),
	}
}

func (m ConnectionManager) dialConn(network, address string) (MessageConnection, error) {
	var err error
	var conn net.Conn

	if m.proxy != nil {
		conn, err = m.proxy.Dial(network, address)
	} else {
		conn, err = net.Dial(network, address)
	}

	if err != nil {
		return MessageConnection{}, err
	}

	return MessageConnection{
		sio.WrapConnection(conn),
	}, nil
}

func (mc MessageConnection) expectString(expected string) error {
	resp, err := mc.conn.ReadString()
	if err != nil {
		return err
	} else if resp != expected {
		return fmt.Errorf("received response \"%s\" wanted \"%s\"", resp, expected)
	}

	return nil
}

func (mc MessageConnection) ExpectStatusMessage(expected StatusMessage) error {
	return mc.expectString(string(expected))
}

func (mc MessageConnection) SendStatusMessage(msg StatusMessage) error {
	_, err := mc.conn.WriteString(string(msg))
	mc.conn.Flush()
	return err
}

func (mc MessageConnection) SendMessages(messages ...Message) error {
	_, err := mc.conn.WriteStruct(messages)
	if err != nil {
		return err
	}

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) ReadMessages() ([]Message, error) {
	msgs := make([]Message, 0)
	err := mc.conn.ReadStruct(&msgs)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func (mc MessageConnection) SendUUID(id uuid.UUID) error {
	_, err := mc.conn.WriteBytes(id[:])
	if err != nil {
		return err
	}

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) ReadUUID() (uuid.UUID, error) {
	raw, err := mc.conn.ReadBytes()
	if err != nil {
		return uuid.UUID{}, err
	}

	id, err := uuid.FromBytes(raw)
	if err != nil {
		return uuid.UUID{}, err
	}

	return id, nil
}

func (mc MessageConnection) SendUUIDs(ids ...uuid.UUID) error {
	_, err := mc.conn.WriteStruct(ids)
	if err != nil {
		return err
	}

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) ReadUUIDs() ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	err := mc.conn.ReadStruct(&ids)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (mc MessageConnection) SendSyncMap(syncMap SyncMap) error {
	_, err := mc.conn.WriteStruct(syncMap)
	if err != nil {
		return err
	}
	mc.conn.Flush()
	return nil
}

func (mc MessageConnection) ReadSyncMap() (SyncMap, error) {
	remoteSyncTimes := make(SyncMap)
	err := mc.conn.ReadStruct(&remoteSyncTimes)
	if err != nil {
		return nil, err
	}

	return remoteSyncTimes, nil
}

func (mc MessageConnection) SendContactRequest(request ContactRequest) error {
	_, err := mc.conn.WriteStruct(&request)
	if err != nil {
		return err
	}
	mc.conn.Flush()
	return nil
}

func (mc MessageConnection) ReadContactRequest() (*ContactRequest, error) {
	cReq := &ContactRequest{}
	err := mc.conn.ReadStruct(cReq)
	if err != nil {
		return nil, err
	}

	return cReq, err
}

func (mc MessageConnection) SendContactResponse(response ContactResponse) error {
	_, err := mc.conn.WriteStruct(&response)
	if err != nil {
		return err
	}

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) ReadContactResponse() (ContactResponse, error) {
	resp := ContactResponse{}
	err := mc.conn.ReadStruct(&resp)
	if err != nil {
		return ContactResponse{}, err
	}

	return resp, nil
}

func (mc MessageConnection) SolveFingerprintChallenge(sID Identity) error {
	challenge, err := mc.conn.ReadBytes()
	if err != nil {
		return err
	}

	signed, err := sID.Sign(challenge)
	if err != nil {
		log.WithError(err).Debug()
	}

	mc.conn.WriteString(sID.Fingerprint())
	mc.conn.WriteBytes(signed)

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) ReadFingerprintWithChallenge() (string, error) {
	challenge, _ := mc.writeRandom(32)

	fingerprint, err := mc.conn.ReadString()
	if err != nil {
		return "", err
	}
	sig, err := mc.conn.ReadBytes()
	if err != nil {
		return "", err
	}

	keyBytes, err := base64.RawURLEncoding.DecodeString(fingerprint)
	if err != nil {
		return "", err
	}

	key := ed25519.PublicKey(keyBytes)
	if !ed25519.Verify(key, challenge, sig) {
		return "", fmt.Errorf("remote failed challenge")
	}

	return fingerprint, nil
}

func (mc MessageConnection) SendBlobs(blobManager blobmngr.ManagesBlobs, blobIds ...uuid.UUID) error {
	mc.SendUUIDs(blobIds...)

	for _, id := range blobIds {
		stat, err := blobManager.StatFromID(id)
		if err != nil {
			return err
		}

		blockCount := int(stat.Size() / blocksize)
		if stat.Size()%blocksize != 0 {
			blockCount++
		}

		mc.conn.WriteInt(blockCount)
		mc.conn.Flush()

		file, err := blobManager.FileFromID(id)
		if err != nil {
			return err
		}
		defer file.Close()

		buf := make([]byte, blocksize)
		for c := 0; c < blockCount; c++ {
			n, err := file.Read(buf)
			if err != nil {
				return err
			}

			mc.conn.WriteBytes(buf[:n])
			mc.conn.Flush()

			err = mc.ExpectStatusMessage(BlockOK)
			if err != nil {
				return err
			}
		}

		err = mc.ExpectStatusMessage(BlobOK)
		if err != nil {
			return err
		}

		log.WithField("blob", id.String()).Debug("transferred blob")
	}

	return nil
}

func (mc MessageConnection) ReadAndCreateBlobs(blobManager blobmngr.BlobManager) error {
	ids, _ := mc.ReadUUIDs()

	for _, id := range ids {
		blockcount, err := mc.conn.ReadInt()
		if err != nil {
			return err
		}

		file, err := blobManager.FileFromID(id)
		if err != nil {
			return err
		}

		rcvOK := false
		defer func() {
			file.Close()
			if !rcvOK {
				blobManager.RemoveBlob(id)
			}
		}()

		for i := 0; i < blockcount; i++ {
			buf, err := mc.conn.ReadBytes()
			if err != nil {
				return err
			}

			_, err = file.Write(buf)
			if err != nil {
				return err
			}

			mc.SendStatusMessage(BlockOK)
		}

		mc.SendStatusMessage(BlobOK)

		rcvOK = true
	}

	return nil
}

func (m ConnectionManager) contactPeer(room *Room, peerCID Identity) (ContactResponse, error) {
	conn, err := m.dialConn("tcp", peerCID.URL()+":"+strconv.Itoa(PubContPort))
	if err != nil {
		return ContactResponse{}, err
	}
	defer conn.Close()

	req := ContactRequest{
		RemoteFP: peerCID.Fingerprint(),
		LocalFP:  room.Self.Fingerprint(),
		ID:       room.ID,
	}
	err = conn.SendContactRequest(req)
	if err != nil {
		return ContactResponse{}, err
	}

	resp, err := conn.ReadContactResponse()
	if err != nil {
		return ContactResponse{}, err
	}

	return resp, nil
}

func (mc MessageConnection) writeRandom(length int) ([]byte, error) {
	r := make([]byte, length)
	rand.Read(r)

	_, err := mc.conn.WriteBytes(r)
	if err != nil {
		return nil, err
	}

	mc.conn.Flush()

	return r, nil
}

func (m ConnectionManager) syncMsgs(room *Room, peerRID Identity) error {
	if room == nil {
		return fmt.Errorf("room not set")
	}

	conn, err := m.dialConn("tcp", peerRID.URL()+":"+strconv.Itoa(PubConvPort))
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.SolveFingerprintChallenge(room.Self)
	if err != nil {
		return err
	}

	conn.SendUUID(room.ID)

	err = conn.ExpectStatusMessage(AuthOK)
	if err != nil {
		return err
	}

	remoteSyncTimes, err := conn.ReadSyncMap()
	if err != nil {
		return err
	}

	msgsToSync := room.findMessagesToSync(remoteSyncTimes)
	conn.SendMessages(msgsToSync...)

	err = conn.ExpectStatusMessage(MessagesOK)
	if err != nil {
		return err
	}

	err = conn.SendBlobs(m.blobManager, blobIDsFromMessages(msgsToSync...)...)
	if err != nil {
		return err
	}

	err = conn.ExpectStatusMessage(SyncOK)
	if err != nil {
		return err
	}

	return nil
}

func (mc MessageConnection) Close() error {
	return mc.conn.Close()
}
