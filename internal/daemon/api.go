package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/Craumix/tormsg/internal/types"
	"github.com/google/uuid"
)

func startAPIServer() {
	log.Printf("Starting API-Server %s\n", apiSocket.Addr())

	http.HandleFunc("/v1/status", statusRoute)

	http.HandleFunc("/v1/contact/list", listContactIDsRoute)
	http.HandleFunc("/v1/contact/add", addContactIDRoute)
	http.HandleFunc("/v1/contact/remove", rmContactIDRoute)

	http.HandleFunc("/v1/room/list", listRoomsRoute)
	http.HandleFunc("/v1/room/create", createRoomRoute)
	http.HandleFunc("/v1/room/send", sendMessageRoute)
	http.HandleFunc("/v1/room/messages", listRoomMessagesRoute)

	err := http.Serve(apiSocket, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func statusRoute(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("{\"status\":\"ok\"}"))
	return
}

func listContactIDsRoute(w http.ResponseWriter, req *http.Request) {
	contIDs := make([]string, 0)
	for _, id := range data.ContactIdentities {
		contIDs = append(contIDs, id.Fingerprint())
	}

	raw, _ := json.Marshal(&contIDs)

	w.Write(raw)
	return
}

func listRoomsRoute(w http.ResponseWriter, req *http.Request) {
	rooms := make([]string, 0)
	for key := range data.Rooms {
		rooms = append(rooms, key.String())
	}

	raw, _ := json.Marshal(&rooms)

	w.Write(raw)
	return
}

func addContactIDRoute(w http.ResponseWriter, req *http.Request) {
	id := types.NewIdentity()
	err := registerContactIdentity(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("{\"fingerprint\":\"%s\"}", id.Fingerprint())))
	return
}

func rmContactIDRoute(w http.ResponseWriter, req *http.Request) {
	fp := req.FormValue("fingerprint")
	if fp == "" {
		http.Error(w, "Missing parameter \"fingerprint\"", http.StatusBadRequest)
		return
	}

	err := deregisterContactIdentity(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	return
}

func createRoomRoute(w http.ResponseWriter, req *http.Request) {
	var fingerprints []string

	body, err := io.ReadAll(req.Body)
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

	ids := make([]*types.RemoteIdentity, 0)
	for _, f := range fingerprints {
		p, err := types.NewRemoteIdentity(f)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		ids = append(ids, p)
	}

	go func() {
		room, err := types.NewRoom(ids, torInstance.Proxy, contactPort, conversationPort)
		if err != nil {
			log.Println(err.Error())
			return
		}

		err = registerRoom(room)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}()

	return
}

func sendMessageRoute(w http.ResponseWriter, req *http.Request) {
	uuid, err := uuid.Parse(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var mtype int
	if req.FormValue("type") == "" {
		mtype = types.MTYPE_TEXT;
	}else {
		mtype, err = strconv.Atoi(req.FormValue("type"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	room := data.Rooms[uuid]
	if room == nil {
		http.Error(w, "No such room " + uuid.String(), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	room.SendMessage(byte(mtype), body)

	return
}

func listRoomMessagesRoute(w http.ResponseWriter, req *http.Request) {
	uuid, err := uuid.Parse(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	room := data.Rooms[uuid]
	if room == nil {
		http.Error(w, "No such room " + uuid.String(), http.StatusBadRequest)
	}

	raw, _ := json.Marshal(room.Messages)

	w.Write(raw)
	return
}
