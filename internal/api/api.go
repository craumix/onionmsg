package api

import (
	"encoding/json"
	"fmt"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 2 << 14 //8K
	maxFileSize    = 2 << 30 //2G

	unixSocketName = "onionmsg.sock"
	defaultApiPort = 10052
)

func defaultUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		//TODO Fixme
		CheckOrigin: func(r *http.Request) bool { return true },
	}
}

type Backend interface {
	TorInfo() interface{}
	GetNotifier() types.Notifier
	GetBlobManager() blobmngr.ManagesBlobs
	GetContactIDsAsStrings() []string
	CreateAndRegisterNewContactID() (types.Identity, error)
	DeregisterAndRemoveContactIDByFingerprint(fingerprint string) error
	GetRoomRequests() []*types.RoomRequest
	AcceptRoomRequest(id string) error
	RemoveRoomRequestByID(toRemove string)
	GetRoomInfoByID(roomId string) (*types.RoomInfo, error)
	GetInfoForAllRooms() []*types.RoomInfo
	DeregisterAndDeleteRoomByID(roomID string) error
	CreateRoom(fingerprints []string) error
	SendMessageInRoom(roomID string, content types.MessageContent) error
	ListMessagesInRoom(roomID string, count int) ([]types.Message, error)
	AddNewPeerToRoom(roomID string, newPeerFingerprint string) error
}

type Config struct {
	UseUnixSocket bool
	PortOffset    int
}

type API struct {
	config Config

	port       int
	backend    Backend
	wsUpgrader websocket.Upgrader
}

func NewAPI(config Config, backend Backend) *API {
	return &API{
		config: config,

		port:       defaultApiPort + config.PortOffset,
		backend:    backend,
		wsUpgrader: defaultUpgrader(),
	}
}

func (api *API) Start() error {
	var (
		listener net.Listener
		err      error
	)

	if api.config.UseUnixSocket {
		listener, err = sio.CreateUnixSocket(unixSocketName)
	} else {
		listener, err = sio.CreateTCPSocket(defaultApiPort)
	}

	if err != nil {
		return err
	}

	log.WithField("address", listener.Addr()).Info("Starting API-Server")

	api.setupHandleFuncs()

	return http.Serve(listener, nil)
}

func (api *API) setupHandleFuncs() {
	http.HandleFunc("/v1/ws", api.routeOpenWS)

	http.HandleFunc("/v1/status", api.RouteStatus)
	http.HandleFunc("/v1/tor", api.RouteTorInfo)

	http.HandleFunc("/v1/blob", api.RouteBlob)

	http.HandleFunc("/v1/contact/list", api.RouteContactList)
	http.HandleFunc("/v1/contact/create", api.RouteContactCreate)
	http.HandleFunc("/v1/contact/delete", api.RouteContactDelete)

	http.HandleFunc("/v1/request/list", api.RouteRequestList)
	http.HandleFunc("/v1/request/accept", api.RouteRequestAccept)
	http.HandleFunc("/v1/request/delete", api.RouteRequestDelete)

	http.HandleFunc("/v1/room/info", api.RouteRoomInfo)
	http.HandleFunc("/v1/room/list", api.RouteRoomList)
	http.HandleFunc("/v1/room/create", api.RouteRoomCreate)
	http.HandleFunc("/v1/room/delete", api.RouteRoomDelete)
	http.HandleFunc("/v1/room/send/message", api.RouteRoomSendMessage)
	http.HandleFunc("/v1/room/send/file", api.RouteRoomSendFile)
	http.HandleFunc("/v1/room/messages", api.RouteRoomMessages)

	http.HandleFunc("/v1/room/command/useradd", api.RouteRoomCommandUseradd)
	http.HandleFunc("/v1/room/command/nameroom", api.RouteRoomCommandNameRoom)
	http.HandleFunc("/v1/room/command/setnick", api.RouteRoomCommandSetNick)
	http.HandleFunc("/v1/room/command/promote", api.RouteRoomCommandPromote)
	http.HandleFunc("/v1/room/command/removepeer", api.RouteRoomCommandRemovePeer)
}

func (api *API) routeOpenWS(w http.ResponseWriter, req *http.Request) {
	conn, err := api.wsUpgrader.Upgrade(w, req, nil)
	if err != nil {
		log.WithError(err).Warn("error when upgrading connection")
	}

	notifier := api.backend.GetNotifier()
	notifier.AddObserver(conn)
}

func (api *API) RouteStatus(w http.ResponseWriter, _ *http.Request) {
	setJSONContentHeader(w)
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func (api *API) RouteTorInfo(w http.ResponseWriter, _ *http.Request) {
	sendSerialized(w, api.backend.TorInfo())
}

func (api *API) RouteBlob(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = api.backend.GetBlobManager().StatFromID(id)
	if os.IsNotExist(err) {
		http.Error(w, "Blob not found!", http.StatusNotFound)
		return
	}

	//To set correct filename for downloads
	respFilename := req.FormValue("filename")
	if respFilename != "" {
		w.Header().Add("Content-Disposition", "attachment; filename=\""+respFilename+"\"")
	}

	//If the blob exists, it will never change
	w.Header().Add("Cache-Control", "public, max-age=604800, immutable")
	w.Header().Add("Content-Type", "application/octet-stream")

	err = api.backend.GetBlobManager().StreamTo(id, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (api *API) RouteContactList(w http.ResponseWriter, _ *http.Request) {
	sendSerialized(w, api.backend.GetContactIDsAsStrings())
}

func (api *API) RouteContactCreate(w http.ResponseWriter, _ *http.Request) {
	cID, err := api.backend.CreateAndRegisterNewContactID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setJSONContentHeader(w)
	w.Write([]byte(fmt.Sprintf("{\"id\":\"%s\"}", cID.Fingerprint())))
}

func (api *API) RouteContactDelete(w http.ResponseWriter, req *http.Request) {
	fp := req.FormValue("fingerprint")
	if fp == "" {
		http.Error(w, "Missing parameter \"fingerprint\"", http.StatusBadRequest)
		return
	}

	err := api.backend.DeregisterAndRemoveContactIDByFingerprint(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (api *API) RouteRequestList(w http.ResponseWriter, _ *http.Request) {
	sendSerialized(w, api.backend.GetRoomRequests())
}

func (api *API) RouteRequestAccept(w http.ResponseWriter, req *http.Request) {
	sid := req.FormValue("uuid")
	id, err := uuid.Parse(sid)
	if err != nil {
		http.Error(w, "Malformed uuid", http.StatusBadRequest)
		return
	}

	err = api.backend.AcceptRoomRequest(id.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) RouteRequestDelete(w http.ResponseWriter, req *http.Request) {
	sid := req.FormValue("uuid")
	id, err := uuid.Parse(sid)
	if err != nil {
		http.Error(w, "Malformed uuid", http.StatusBadRequest)
		return
	}

	api.backend.RemoveRoomRequestByID(id.String())
}

func (api *API) RouteRoomInfo(w http.ResponseWriter, req *http.Request) {
	sid := req.FormValue("uuid")
	id, err := uuid.Parse(sid)
	if err != nil {
		http.Error(w, "Malformed uuid", http.StatusBadRequest)
		return
	}

	info, err := api.backend.GetRoomInfoByID(id.String())
	if err != nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	sendSerialized(w, info)
}

func (api *API) RouteRoomList(w http.ResponseWriter, _ *http.Request) {
	sendSerialized(w, api.backend.GetInfoForAllRooms())
}

func (api *API) RouteRoomCreate(w http.ResponseWriter, req *http.Request) {
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

	err = api.backend.CreateRoom(ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) RouteRoomDelete(w http.ResponseWriter, req *http.Request) {
	err := api.backend.DeregisterAndDeleteRoomByID(req.FormValue("uuid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (api *API) RouteRoomSendMessage(w http.ResponseWriter, req *http.Request) {
	//TODO Modify this to only send messages and create extra endpoint for blobs
	errCode, err := api.sendMessage(req, "")
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func (api *API) RouteRoomSendFile(w http.ResponseWriter, req *http.Request) {
	id, err := api.backend.GetBlobManager().MakeBlob()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = api.backend.GetBlobManager().WriteIntoBlob(req.Body, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := req.Header.Get(FilenameHeader)

	mimetype := req.Header.Get(MimetypeHeader)
	if mimetype == "" {
		mimetype = mime.TypeByExtension(filepath.Ext(filename))
	}

	filesize := 0
	fileStat, err := api.backend.GetBlobManager().StatFromID(id)
	if err == nil {
		filesize = int(fileStat.Size())
	}

	replyto, err := replyFromHeader(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = api.backend.SendMessageInRoom(req.FormValue("uuid"), types.MessageContent{
		Type:    types.ContentTypeFile,
		ReplyTo: replyto,
		Blob: &types.BlobMeta{
			ID:   id,
			Name: filename,
			Type: mimetype,
			Size: filesize,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (api *API) RouteRoomMessages(w http.ResponseWriter, req *http.Request) {
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

	messages, err := api.backend.ListMessagesInRoom(req.FormValue("uuid"), count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendSerialized(w, messages)
}

func (api *API) RouteRoomCommandUseradd(w http.ResponseWriter, req *http.Request) {
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

	if err := api.backend.AddNewPeerToRoom(roomID.String(), fingerprint); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) RouteRoomCommandNameRoom(w http.ResponseWriter, req *http.Request) {
	errCode, err := api.sendMessage(req, types.RoomCommandNameRoom)
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func (api *API) RouteRoomCommandSetNick(w http.ResponseWriter, req *http.Request) {
	errCode, err := api.sendMessage(req, types.RoomCommandNick)
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func (api *API) RouteRoomCommandPromote(w http.ResponseWriter, req *http.Request) {
	errCode, err := api.sendMessage(req, types.RoomCommandPromote)
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func (api *API) RouteRoomCommandRemovePeer(w http.ResponseWriter, req *http.Request) {
	errCode, err := api.sendMessage(req, types.RoomCommandRemovePeer)
	if err != nil {
		http.Error(w, err.Error(), errCode)
	}
}

func (api *API) sendMessage(req *http.Request, roomCommand types.Command) (int, error) {
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if len(content) > maxMessageSize {
		return http.StatusBadRequest, fmt.Errorf("message too big, cannot be greater %d", maxMessageSize)
	}

	msgType := types.ContentTypeText
	if roomCommand != "" {
		msgType = types.ContentTypeCmd
	}

	replyto, err := replyFromHeader(req)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = api.backend.SendMessageInRoom(req.FormValue("uuid"), types.MessageContent{
		Type:    msgType,
		ReplyTo: replyto,
		Data:    types.ConstructCommand(content, roomCommand),
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil
}
