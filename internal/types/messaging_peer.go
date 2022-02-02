package types

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	queueTimeout = time.Second * 15
)

type MessagingPeer struct {
	RIdentity     RemoteIdentity `json:"identity"`
	LastSyncState SyncMap        `json:"lastSync"`

	ctx         context.Context
	stop        context.CancelFunc
	skipTimeout context.CancelFunc

	Room *Room `json:"-"`
}

func NewMessagingPeer(rid RemoteIdentity) *MessagingPeer {
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

			err := mp.Room.syncMsgs(mp.RIdentity)
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

func (mp *MessagingPeer) Stop() {
	if mp.stop != nil {
		mp.stop()
	}
}
