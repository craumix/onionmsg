package types

import (
	"context"
	"fmt"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"
)

const (
	queueTimeout = time.Second * 15
)

type MessagingPeer struct {
	RIdentity     Identity `json:"identity"`
	LastSyncState SyncMap  `json:"lastSync"`

	ctx         context.Context
	stop        context.CancelFunc
	skipTimeout context.CancelFunc

	Room *Room `json:"-"`
}

func NewMessagingPeer(rid Identity) *MessagingPeer {
	return &MessagingPeer{
		RIdentity: rid,
	}
}

// RunMessageQueue creates a cancellable context for the MessagingPeer
// and starts a loop that will try to send queued messages every so often.
func (mp *MessagingPeer) RunMessageQueue(ctx context.Context, room *Room) {
	mp.Room = room

	mp.ctx, mp.stop = context.WithCancel(ctx)

	lf := log.Fields{
		"room": mp.Room.ID,
		"peer": mp.RIdentity.Fingerprint(),
	}

	log.WithFields(lf).Debug("starting message queue...")

	for {
		select {
		case <-mp.ctx.Done():
			log.WithFields(lf).Debug("queue terminated")
			return
		default:
			if SyncMapsEqual(mp.Room.SyncState, mp.LastSyncState) {
				break
			}

			startSync := time.Now()

			log.WithFields(lf).Debug("running message sync")

			err := mp.syncMsgs()
			if err != nil {
				log.WithError(err).WithFields(lf).Debug("message sync failed")
			} else {
				mp.LastSyncState = CopySyncMap(mp.Room.SyncState)
				log.WithField("time", time.Since(startSync)).WithFields(lf).Debug("message sync done")
			}
		}

		var skip context.Context
		skip, mp.skipTimeout = context.WithCancel(context.Background())

		select {
		case <-skip.Done(): //used to skip a single wait period
		case <-mp.ctx.Done(): //context cancelled
		case <-time.After(queueTimeout): //timeout
		}
	}
}

func (mp *MessagingPeer) BumpQueue() {
	if mp.skipTimeout != nil {
		mp.skipTimeout()
	}
}

func (mp *MessagingPeer) syncMsgs() error {
	if mp.Room == nil {
		return fmt.Errorf("Room not set")
	}

	conn, err := connection.GetConnFunc("tcp", mp.RIdentity.URL()+":"+strconv.Itoa(PubConvPort))
	if err != nil {
		return err
	}
	defer conn.Close()

	err = fingerprintChallenge(conn, mp.Room.Self)
	if err != nil {
		return err
	}

	conn.WriteBytes(mp.Room.ID[:])
	conn.Flush()

	err = expectResponse(conn, "auth_ok")
	if err != nil {
		return err
	}

	remoteSyncTimes := make(SyncMap)
	err = conn.ReadStruct(&remoteSyncTimes)
	if err != nil {
		return err
	}

	msgsToSync := mp.findMessagesToSync(remoteSyncTimes)
	conn.WriteStruct(msgsToSync)
	conn.Flush()

	err = expectResponse(conn, "messages_ok")
	if err != nil {
		return err
	}

	blobIDs := blobIDsFromMessages(msgsToSync...)
	err = sendBlobs(conn, blobIDs)
	if err != nil {
		return err
	}

	err = expectResponse(conn, "sync_ok")
	if err != nil {
		return err
	}
	return nil
}

func (mp *MessagingPeer) Stop() {
	if mp.stop != nil {
		mp.stop()
	}
}

func sendBlobs(conn connection.ConnWrapper, ids []uuid.UUID) error {
	conn.WriteStruct(ids)
	conn.Flush()

	for _, id := range ids {
		stat, err := blobmngr.StatFromID(id)
		if err != nil {
			return err
		}

		blockCount := int(stat.Size() / blocksize)
		if stat.Size()%blocksize != 0 {
			blockCount++
		}

		conn.WriteInt(blockCount)
		conn.Flush()

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

			conn.WriteBytes(buf[:n])
			conn.Flush()

			err = expectResponse(conn, "block_ok")
			if err != nil {
				return err
			}
		}

		err = expectResponse(conn, "blob_ok")
		if err != nil {
			return err
		}

		log.WithField("blob", id.String()).Debug("transferred blob")
	}

	return nil
}

func (mp *MessagingPeer) findMessagesToSync(remoteSyncTimes SyncMap) []Message {
	msgs := make([]Message, 0)

	for _, msg := range mp.Room.Messages {
		if last, ok := remoteSyncTimes[msg.Meta.Sender]; !ok || msg.Meta.Time.After(last) {
			msgs = append(msgs, msg)
		}
	}

	return msgs
}
