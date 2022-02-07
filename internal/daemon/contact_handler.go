package daemon

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
)

func (d *Daemon) handleContact(conn net.Conn) {
	mConn := d.ConnectionManager.UseConnection(conn)
	defer mConn.Close()

	request, err := mConn.ReadContactRequest()
	if err != nil {
		log.WithError(err).Debug()
		return
	}

	cID, found := d.getContactID(request.RemoteFP)
	if !found {
		log.WithError(fmt.Errorf("contact handler was addressed by unknown name")).Debug()
		return
	}

	response, roomRequest, err := request.GenerateResponse(cID)
	if err != nil {
		log.WithError(err).Debug()
		return
	}

	err = mConn.SendContactResponse(response)
	if err != nil {
		log.WithError(err).Debug()
		return
	}

	d.addRoomRequest(&roomRequest)

	if d.Config.AutoAccept {
		d.AcceptRoomRequest(roomRequest.ID)
	} else {
		d.Notifier.NotifyNewRequest(&roomRequest)
	}
}
