package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type Message struct {
	Sender    string    `json:"sender"`
	Time      time.Time `json:"time"`
	Type      byte      `json:"type"`
	Content   []byte    `json:"content"`
	Signature []byte    `json:"signature"`
}

type statusResponse struct {
	Status string `json:"status"`
}

type torlogResponse struct {
	Log string `json:"log"`
}

type listContactIDsResponse []string

type listRoomsResponse []string

type addContactIDResponse struct {
	Fingerprint string `json:"fingerprint"`
}

var (
	socketType string
	client     *http.Client
	address    string
)

func Init(connectionType, location string) error {
	switch connectionType {
	case "tcp":
		address = "http://" + location
		socketType = connectionType
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial(socketType, location)
				},
			},
		}
	case "unix":
		address = "http://unix"
		socketType = connectionType
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial(socketType, location)
				},
			},
		}
	default:
		return fmt.Errorf("invalid socket type %s\nmust be either tcp or unix (default tcp)", socketType)
	}
	return nil
}

func Status() (bool, error) {
	var resp statusResponse
	err := getRequest("/v1/status", &resp)
	if err != nil {
		return false, err
	}
	return resp.Status == "ok", nil
}

func Torlog() (string, error) {
	var resp torlogResponse
	err := getRequest("/v1/torlog", &resp)
	if err != nil {
		return "", err
	}
	return resp.Log, nil
}

func ListContactIDs() ([]string, error) {
	var resp listContactIDsResponse
	err := getRequest("/v1/contact/list", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func ListRooms() ([]string, error) {
	var resp listRoomsResponse
	err := getRequest("/v1/room/list", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func CreateContactID() (string, error) {
	var resp addContactIDResponse
	err := getRequest("/v1/contact/create", &resp)
	if err != nil {
		return "", err
	}
	return resp.Fingerprint, nil
}

func DeleteContactID(fingerprint string) error {
	return getRequest(fmt.Sprintf("/v1/contact/delete?fingerprint=%s", fingerprint), nil)
}

func CreateRoom(fingerprints []string) error {
	req, err := json.Marshal(fingerprints)
	if err != nil {
		return err
	}
	return postRequest("/v1/room/create", req, nil)
}

func DeleteRoom(uuid string) error {
	return getRequest(fmt.Sprintf("/v1/room/delete?uuid=%s", uuid), nil)
}

func SendMessage(uuid string, mtype int, msg []byte) error {
	return postRequest(fmt.Sprintf("/v1/room/send?uuid=%s&mtype=%d", uuid, mtype), msg, nil)
}

func ListMessages(uuid string) ([]Message, error) {
	var resp []Message
	err := getRequest(fmt.Sprintf("/v1/room/messages?uuid=%s", uuid), &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func getRequest(path string, v interface{}) error {
	resp, err := client.Get(address + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if v != nil {
		return json.Unmarshal(body, v)
	}
	return nil
}

func postRequest(path string, body []byte, v interface{}) error {
	resp, err := client.Post(address+path, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	} else if v != nil {
		return json.Unmarshal(respBody, v)
	}
	return nil
}
