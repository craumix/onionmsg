package types

import (
	"fmt"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"net"
	"strconv"
)

type StatusMessage string

const (
	AuthOK     = "auth_ok"
	MessagesOK = "messages_ok"
	SyncOK     = "sync_ok"
	BlockOK    = "block_ok"
	BlobOK     = "blob_ok"

	blocksize = 1 << 19 // 512K
)

type ConnectionManager struct {
	Proxy proxy.Dialer
}

type MessageConnection struct {
	conn connection.ConnWrapper
}

func NewConnectionManager(proxy proxy.Dialer) ConnectionManager {
	return ConnectionManager{
		Proxy: proxy,
	}
}

func (m ConnectionManager) dialConn(network, address string) (MessageConnection, error) {
	var err error
	var conn net.Conn

	if m.Proxy != nil {
		conn, err = m.Proxy.Dial(network, address)
	} else {
		conn, err = net.Dial(network, address)
	}

	if err != nil {
		return MessageConnection{}, err
	}

	wrappedConn := connection.WrapConnection(conn)

	if wrappedConn == nil {
		log.Print("OH NO")
	}

	return MessageConnection{
		wrappedConn,
	}, nil
}

func (mc MessageConnection) expectResponse(expected string) error {
	resp, err := mc.conn.ReadString()
	if err != nil {
		return err
	} else if resp != expected {
		return fmt.Errorf("received response \"%s\" wanted \"%s\"", resp, expected)
	}

	return nil
}

func (mc MessageConnection) ExpectStatusMessage(expected StatusMessage) error {
	return mc.expectResponse(string(expected))
}

func (mc MessageConnection) SendMessages(messages ...Message) error {
	_, err := mc.conn.WriteStruct(messages)
	if err != nil {
		return err
	}

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) SendUUIDs(ids ...uuid.UUID) error {
	_, err := mc.conn.WriteStruct(ids)
	if err != nil {
		return err
	}

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) SolveFingerprintChallenge(sID Identity, roomID uuid.UUID) error {
	if mc.conn == nil {
		log.Print("OOF")
	}

	challenge, err := mc.conn.ReadBytes()
	if err != nil {
		return err
	}

	signed, err := sID.Sign(challenge)
	if err != nil {
		fmt.Print(err.Error())
	}

	mc.conn.WriteString(sID.Fingerprint())
	mc.conn.WriteBytes(signed)
	mc.conn.WriteBytes(roomID[:])

	mc.conn.Flush()

	return nil
}

func (mc MessageConnection) SendBlobs(blobIds ...uuid.UUID) error {
	mc.SendUUIDs(blobIds...)

	for _, id := range blobIds {
		stat, err := blobmngr.StatFromID(id)
		if err != nil {
			return err
		}

		blockCount := int(stat.Size() / blocksize)
		if stat.Size()%blocksize != 0 {
			blockCount++
		}

		mc.conn.WriteInt(blockCount)
		mc.conn.Flush()

		file, err := blobmngr.FileFromID(id)
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

func (mc MessageConnection) ReadRemoteSyncMap() (SyncMap, error) {
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

func (mc MessageConnection) ReadContactResponse() (ContactResponse, error) {
	resp := ContactResponse{}
	err := mc.conn.ReadStruct(&resp)
	if err != nil {
		return ContactResponse{}, err
	}

	return resp, nil
}

func (mc MessageConnection) Close() error {
	return mc.conn.Close()
}

func (m ConnectionManager) ContactPeer(room *Room, peerCID Identity) (ContactResponse, error) {
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

func (m ConnectionManager) syncMsgs(room *Room, peerRID Identity) error {
	if room == nil {
		return fmt.Errorf("room not set")
	}

	conn, err := m.dialConn("tcp", peerRID.URL()+":"+strconv.Itoa(PubConvPort))
	if err != nil {
		return err
	}
	//defer conn.Close()

	err = conn.SolveFingerprintChallenge(room.Self, room.ID)
	if err != nil {
		return err
	}

	err = conn.ExpectStatusMessage(AuthOK)
	if err != nil {
		return err
	}

	remoteSyncTimes, err := conn.ReadRemoteSyncMap()
	if err != nil {
		return err
	}

	msgsToSync := room.findMessagesToSync(remoteSyncTimes)
	conn.SendMessages(msgsToSync...)

	err = conn.ExpectStatusMessage(MessagesOK)
	if err != nil {
		return err
	}

	err = conn.SendBlobs(blobIDsFromMessages(msgsToSync...)...)
	if err != nil {
		return err
	}

	err = conn.ExpectStatusMessage(SyncOK)
	if err != nil {
		return err
	}

	return nil
}
