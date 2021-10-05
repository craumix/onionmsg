package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

type ContentType string

const (
	ContentTypeText    ContentType = "mtype.text"
	ContentTypeCmd     ContentType = "mtype.cmd"
	ContentTypeFile    ContentType = "mtype.file"
	ContentTypeSticker ContentType = "mtype.sticker"
)

type BlobMeta struct {
	ID   uuid.UUID `json:"uuid"`
	Name string    `json:"name,omitempty"`
	Type string    `json:"type,omitempty"`
	Size int       `json:"size,omitempty"`
}

type MessageMeta struct {
	Sender string    `json:"sender"`
	Time   time.Time `json:"time"`
}

type MessageContent struct {
	Type    ContentType `json:"type"`
	ReplyTo *Message    `json:"replyto,omitempty"`
	Blob    *BlobMeta   `json:"blob,omitempty"`
	Data    []byte      `json:"data,omitempty"`
}

type Message struct {
	Meta    MessageMeta    `json:"meta"`
	Content MessageContent `json:"content"`
	Sig     []byte         `json:"sig"`
}

func (m *Message) ContainsBlob() bool {
	return m.Content.Blob != nil
}

func (m *Message) Sign(key ed25519.PrivateKey) {
	m.Sig = ed25519.Sign(key, m.signData())
}

func (m *Message) SigIsValid() bool {
	rawKey, err := base64.RawURLEncoding.DecodeString(m.Meta.Sender)
	if err != nil {
		log.Debugf("unable to decode %s as message sender!", m.Meta.Sender)
		return false
	} else if len(rawKey) != ed25519.PublicKeySize {
		log.Debugf("invalid length for public key, %d instead of %d!", len(rawKey), ed25519.PublicKeySize)
		return false
	}

	if m.Content.ReplyTo != nil && !m.Content.ReplyTo.SigIsValid() {
		log.Warn("recursive signature check failed for message")
		return false
	}

	pubKey := ed25519.PublicKey(rawKey)

	return ed25519.Verify(pubKey, m.signData(), m.Sig)
}

func (m *Message) signData() []byte {
	const (
		sigFieldName = "Sig"
	)

	ref := reflect.ValueOf(m).Elem()
	typeOf := ref.Type()

	signData := make([]byte, 0)

	if _, containsSig := typeOf.FieldByName(sigFieldName); !containsSig {
		log.Panicf("Message struct is missing a signature field called %s!", sigFieldName)
	}

	for i := 0; i < ref.NumField(); i++ {
		if typeOf.Field(i).Name != sigFieldName {
			v, _ := json.Marshal(ref.Field(i).Interface())
			signData = append(signData, v...)
		}
	}

	return signData
}

func NewMessage(content MessageContent, sender Identity) Message {
	msg := Message{
		Meta: MessageMeta{
			Sender: sender.Fingerprint(),
			Time:   time.Now().UTC(),
		},
		Content: content,
	}

	msg.Sign(*sender.Priv)

	return msg
}

func (m *Message) isCommand() (bool, string) {
	return m.Content.Type == ContentTypeCmd, strings.Split(string(m.Content.Data), CommandDelimiter)[0]
}
