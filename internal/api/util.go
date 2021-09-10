package api

import (
	"encoding/json"
	"net/http"

	"github.com/craumix/onionmsg/pkg/types"
)

const (
	//FIX maybe (use appropriate existing headers)
	//https://datatracker.ietf.org/doc/html/rfc6648
	replyToHeader  = "X-ReplyTo"
	filenameHeader = "X-Filename"
	mimetypeHeader = "X-Mimetype"
)

func setJSONContentHeader(w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json")
}

func sendSerialized(w http.ResponseWriter, v interface{}) {
	raw, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setJSONContentHeader(w)
	w.Write(raw)
}

func replyFromHeader(req *http.Request) (*types.Message, error) {
	rawReply := req.Header.Get(replyToHeader)
	if rawReply == "" {
		return nil, nil
	}

	msg := &types.Message{}
	err := json.Unmarshal([]byte(rawReply), msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
