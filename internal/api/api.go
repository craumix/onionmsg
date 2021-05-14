package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	"github.com/ipsn/go-adorable"
)

const (
	//8K
	maxMessageSize = 2 << 14
	//2G
	maxFileSize = 2 << 30
)

func Start(listener net.Listener) {
	log.Printf("Starting API-Server %s\n", listener.Addr())

	http.HandleFunc("/v1/status", routeStatus)
	http.HandleFunc("/v1/torlog", routeTorlog)

	http.HandleFunc("/v1/blob", routeBlob)
	http.HandleFunc("/v1/avatar", routeAvatar)

	http.HandleFunc("/v1/contact/list", routeContactList)
	http.HandleFunc("/v1/contact/create", routeContactCreate)
	http.HandleFunc("/v1/contact/delete", routeContactDelete)

	http.HandleFunc("/v1/room/list", routeRoomList)
	http.HandleFunc("/v1/room/create", routeRoomCreate)
	http.HandleFunc("/v1/room/delete", routeRoomDelete)
	http.HandleFunc("/v1/room/send/message", routeRoomSendMessage)
	http.HandleFunc("/v1/room/send/file", routeRoomSendFile)
	http.HandleFunc("/v1/room/messages", routeRoomMessages)
	http.HandleFunc("/v1/room/command/useradd", routeRoomCommandUseradd)
	http.HandleFunc("/v1/room/command/nameroom", routeRoomCommandNameroom)
	http.HandleFunc("/v1/room/command/setnick", routeRoomCommandSetnick)

	err := http.Serve(listener, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func routeStatus(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func routeTorlog(w http.ResponseWriter, req *http.Request) {
	logs, err := daemon.GetTorlog()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var torlogResp struct {
		Log string `json:"log"`
	}
	torlogResp.Log = string(logs)
	msg, err := json.Marshal(torlogResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(msg))
}

func routeBlob(w http.ResponseWriter, req *http.Request) {
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

func routeAvatar(w http.ResponseWriter, req *http.Request) {
	w.Write(adorable.PseudoRandom([]byte(req.FormValue("seed"))))
}

func routeContactList(w http.ResponseWriter, req *http.Request) {

	contIDs := daemon.ListContactIDs()
	raw, _ := json.Marshal(&contIDs)

	w.Write(raw)
}

func routeContactCreate(w http.ResponseWriter, req *http.Request) {
	fp, err := daemon.CreateContactID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("{\"fingerprint\":\"%s\"}", fp)))
}

func routeContactDelete(w http.ResponseWriter, req *http.Request) {
	fp := req.FormValue("fingerprint")
	if fp == "" {
		http.Error(w, "Missing parameter \"fingerprint\"", http.StatusBadRequest)
		return
	}

	err := daemon.DeleteContactID(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func routeRoomList(w http.ResponseWriter, req *http.Request) {

	rooms := daemon.ListRooms()
	raw, _ := json.Marshal(&rooms)

	w.Write(raw)
}

func routeRoomCreate(w http.ResponseWriter, req *http.Request) {
	var fingerprints []string

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &fingerprints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(fingerprints) == 0 {
		http.Error(w, "Must provide at least one contactID", http.StatusBadRequest)
		return
	}
	if err := daemon.CreateRoom(fingerprints); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func routeRoomDelete(w http.ResponseWriter, req *http.Request) {
	err := daemon.DeleteRoom(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//Modify this to only send messages and create extra endpoint for blobs
func routeRoomSendMessage(w http.ResponseWriter, req *http.Request) {
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(content) > maxMessageSize {
		http.Error(w, fmt.Sprintf("message to big, cannot be greater %d", maxMessageSize), http.StatusBadRequest)
		return
	}

	err = daemon.SendMessage(req.FormValue("uuid"), types.MTYPE_TEXT, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func routeRoomSendFile(w http.ResponseWriter, req *http.Request) {
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

	buf := make([]byte, 4096)
	var n int
	for {
		n, _ = req.Body.Read(buf)
		if n == 0 {
			break
		}

		_, err = file.Write(buf[:n])
		if err != nil {
			break
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = daemon.SendMessage(req.FormValue("uuid"), types.MTYPE_BLOB, id[:])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func routeRoomMessages(w http.ResponseWriter, req *http.Request) {
	messages, err := daemon.ListMessages(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	raw, _ := json.Marshal(messages)

	w.Write(raw)
}

func routeRoomCommandUseradd(w http.ResponseWriter, req *http.Request) {
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

	if err := daemon.AddUserToRoom(roomID, fingerprint); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func routeRoomCommandNameroom(w http.ResponseWriter, req *http.Request) {
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := "name_room " + string(content)
	err = daemon.SendMessage(req.FormValue("uuid"), types.MTYPE_CMD, []byte(msg))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func routeRoomCommandSetnick(w http.ResponseWriter, req *http.Request) {
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := "nick " + string(content)

	err = daemon.SendMessage(req.FormValue("uuid"), types.MTYPE_CMD, []byte(msg))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
