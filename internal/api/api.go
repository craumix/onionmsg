package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	//8K
	maxMessageSize = 2 << 14
	//2G
	maxFileSize = 2 << 30

	apiPort        = 10052
	unixSocketName = "onionmsg.sock"
)

var (
	wsUpgrader = websocket.Upgrader{
		//TODO Fixme
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func Start(unixSocket bool) {
	var (
		listener net.Listener
		err      error
	)

	if unixSocket {
		listener, err = sio.CreateUnixSocket(unixSocketName)
	} else {
		listener, err = sio.CreateTCPSocket(apiPort)
	}
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Starting API-Server %s\n", listener.Addr())

	http.HandleFunc("/v1/ws", routeOpenWS)

	http.HandleFunc("/v1/status", RouteStatus)
	http.HandleFunc("/v1/tor", RouteTorInfo)

	http.HandleFunc("/v1/blob", RouteBlob)

	http.HandleFunc("/v1/contact/list", RouteContactList)
	http.HandleFunc("/v1/contact/create", RouteContactCreate)
	http.HandleFunc("/v1/contact/delete", RouteContactDelete)

	http.HandleFunc("/v1/room/list", RouteRoomList)
	http.HandleFunc("/v1/room/create", RouteRoomCreate)
	http.HandleFunc("/v1/room/delete", RouteRoomDelete)
	http.HandleFunc("/v1/room/send/message", RouteRoomSendMessage)
	http.HandleFunc("/v1/room/send/file", RouteRoomSendFile)
	http.HandleFunc("/v1/room/messages", RouteRoomMessages)
	http.HandleFunc("/v1/room/command/useradd", RouteRoomCommandUseradd)
	http.HandleFunc("/v1/room/command/nameroom", RouteRoomCommandNameRoom)
	http.HandleFunc("/v1/room/command/setnick", RouteRoomCommandSetNick)

	err = http.Serve(listener, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func routeOpenWS(w http.ResponseWriter, req *http.Request) {
	c, err := wsUpgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("error upgrading connection %s", err)
	}

	observerList = append(observerList, c)
}

func RouteStatus(w http.ResponseWriter, req *http.Request) {
	setJSONContentHeader(w)
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func RouteTorInfo(w http.ResponseWriter, req *http.Request) {
	sendSerialized(w, daemon.TorInfo())
}

func RouteBlob(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = blobmngr.StreamTo(id, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func RouteContactList(w http.ResponseWriter, req *http.Request) {
	sendSerialized(w, daemon.ListContactIDs())
}

func RouteContactCreate(w http.ResponseWriter, req *http.Request) {
	fp, err := daemon.CreateContactID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setJSONContentHeader(w)
	w.Write([]byte(fmt.Sprintf("{\"id\":\"%s\"}", fp)))
}

func RouteContactDelete(w http.ResponseWriter, req *http.Request) {
	fp := req.FormValue("id")
	if fp == "" {
		http.Error(w, "Missing parameter \"id\"", http.StatusBadRequest)
		return
	}

	err := daemon.DeleteContact(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func RouteRoomList(w http.ResponseWriter, req *http.Request) {
	sendSerialized(w, daemon.Rooms())
}

func RouteRoomCreate(w http.ResponseWriter, req *http.Request) {
	var ids []string

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(ids) == 0 {
		http.Error(w, "Must provide at least one contactID", http.StatusBadRequest)
		return
	}

	err = daemon.CreateRoom(ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RouteRoomDelete(w http.ResponseWriter, req *http.Request) {
	err := daemon.DeleteRoom(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Modify this to only send messages and create extra endpoint for blobs
func RouteRoomSendMessage(w http.ResponseWriter, req *http.Request) {
	errCode, err := sendMessage(req, "")
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func RouteRoomSendFile(w http.ResponseWriter, req *http.Request) {
	id, err := blobmngr.MakeBlob()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	file, err := blobmngr.FileFromID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	/*
		lengthStr := req.Header.Get("Content-Length")
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if length > maxFileSize {
			http.Error(w, fmt.Sprintf("file to large, cannot be larger than %d", maxFileSize), http.StatusBadRequest)
			return
		}
	*/

	err = blobmngr.WriteIntoFile(req.Body, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := req.FormValue("Content-Filename")

	mimetype := req.FormValue("Content-Mimetype")
	if mimetype == "" {
		mimetype = mime.TypeByExtension(filepath.Ext(filename))
	}

	err = daemon.SendMessage(req.FormValue("uuid"), types.MessageContent{
		Type: types.ContentTypeFile,
		Meta: types.ContentMeta{
			BlobUUID: id,
			Filename: filename,
			Mimetype: mimetype,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func RouteRoomMessages(w http.ResponseWriter, req *http.Request) {
	var (
		count = 0
		err   error
	)

	if req.FormValue("count") != "" {
		count, err = strconv.Atoi(req.FormValue("count"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	messages, err := daemon.ListMessages(req.FormValue("uuid"), count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendSerialized(w, messages)
}

func RouteRoomCommandUseradd(w http.ResponseWriter, req *http.Request) {
	roomID, err := uuid.Parse(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fingerprint := string(content)

	if err := daemon.AddPeerToRoom(roomID, fingerprint); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RouteRoomCommandNameRoom(w http.ResponseWriter, req *http.Request) {
	errCode, err := sendMessage(req, types.RoomCommandNameRoom)
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func RouteRoomCommandSetNick(w http.ResponseWriter, req *http.Request) {
	errCode, err := sendMessage(req, types.RoomCommandNick)
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func sendMessage(req *http.Request, roomCommand types.RoomCommand) (int, error) {
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if len(content) > maxMessageSize {
		return http.StatusBadRequest, fmt.Errorf("message too big, cannot be greater %d", maxMessageSize)
	}

	msgType := types.ContentTypeText
	msg := ""
	if roomCommand != "" {
		msgType = types.ContentTypeCmd
		msg += string(roomCommand) + " "
	}
	msg += string(content)

	err = daemon.SendMessage(req.FormValue("uuid"), types.MessageContent{
		Type: msgType,
		Data: []byte(msg),
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil
}
