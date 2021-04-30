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
)

func Start(listener net.Listener) {
	log.Printf("Starting API-Server %s\n", listener.Addr())

	http.HandleFunc("/v1/status", statusRoute)
	http.HandleFunc("/v1/torlog", torlogRoute)

	http.HandleFunc("/v1/blob", getBlob)

	http.HandleFunc("/v1/contact/list", listContactIDsRoute)
	http.HandleFunc("/v1/contact/create", createContactIDRoute)
	http.HandleFunc("/v1/contact/delete", deleteContactIDRoute)

	http.HandleFunc("/v1/room/list", listRoomsRoute)
	http.HandleFunc("/v1/room/create", createRoomRoute)
	http.HandleFunc("/v1/room/delete", deleteRoomRoute)
	http.HandleFunc("/v1/room/send", sendMessageRoute)
	http.HandleFunc("/v1/room/messages", listMessagesRoute)
	http.HandleFunc("/v1/room/useradd", addUserToRoom)

	err := http.Serve(listener, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func statusRoute(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func torlogRoute(w http.ResponseWriter, req *http.Request) {
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

func getBlob(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	blob, err := blobmngr.GetRessource(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(blob)
}

func listContactIDsRoute(w http.ResponseWriter, req *http.Request) {

	contIDs := daemon.ListContactIDs()
	raw, _ := json.Marshal(&contIDs)

	w.Write(raw)
}

func listRoomsRoute(w http.ResponseWriter, req *http.Request) {

	rooms := daemon.ListRooms()
	raw, _ := json.Marshal(&rooms)

	w.Write(raw)
}

func createContactIDRoute(w http.ResponseWriter, req *http.Request) {
	fp, err := daemon.CreateContactID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("{\"fingerprint\":\"%s\"}", fp)))
}

func deleteContactIDRoute(w http.ResponseWriter, req *http.Request) {
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

func createRoomRoute(w http.ResponseWriter, req *http.Request) {
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
		http.Error(w, "Must provide at least one contactID", http.StatusBadRequest)
		return
	}
}

func deleteRoomRoute(w http.ResponseWriter, req *http.Request) {
	err := daemon.DeleteRoom(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sendMessageRoute(w http.ResponseWriter, req *http.Request) {
	msgType := byte(types.MTYPE_TEXT)

	if req.FormValue("type") != "" {
		mtype, err := strconv.Atoi(req.FormValue("type"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mtype == types.MTYPE_CMD {
			http.Error(w, "Raw commands are not allowed", http.StatusForbidden)
			return
		}
		if types.MTYPE_TEXT <= mtype && mtype <= types.MTYPE_BLOB {
			msgType = byte(mtype)
		}
	}

	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = daemon.SendMessage(req.FormValue("uuid"), msgType, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func listMessagesRoute(w http.ResponseWriter, req *http.Request) {
	messages, err := daemon.ListMessages(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	raw, _ := json.Marshal(messages)

	w.Write(raw)
}

func addUserToRoom(w http.ResponseWriter, req *http.Request) {
	
}
