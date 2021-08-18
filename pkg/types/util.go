package types

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"
)

const (
	PubContPort = 10050
	PubConvPort = 10051

	blocksize = 1 << 19 // 512K
)

type ContactRequest struct {
	RemoteFP string
	LocalFP  string
	ID       uuid.UUID
}

type ContactResponse struct {
	ConvFP string
	Sig    []byte
}

func RandomString(size int) string {
	r := make([]byte, size)
	rand.Read(r)
	return base64.RawStdEncoding.EncodeToString(r)
}

// SendMessage signs the Message with the Identity and sends it via the connection.ConnWrapper
func SendMessage(dataConnP *connection.ConnWrapper, identity Identity, msg Message) error {
	dataConn := *dataConnP

	sigSalt, err := dataConn.ReadBytes()
	if err != nil {
		return err
	}

	msgMarshal, _ := json.Marshal(msg)
	_, err = sendDataWithSig(&dataConn, identity, msgMarshal, sigSalt)
	if err != nil {
		return nil
	}

	if msg.ContainsBlob() {
		id := msg.Content.Meta.BlobUUID

		stat, err := blobmngr.StatFromID(id)
		if err != nil {
			return err
		}

		blockCount := int(stat.Size() / blocksize)
		if stat.Size()%blocksize != 0 {
			blockCount++
		}

		_, err = dataConn.WriteInt(blockCount)
		if err != nil {
			return err
		}

		file, err := blobmngr.FileFromID(id)
		if err != nil {
			return err
		}

		defer file.Close()

		buf := make([]byte, blocksize)
		for c := 0; c < blockCount; c++ {
			n, err := file.Read(buf)
			if err != nil {
				return err
			}

			_, err = sendDataWithSig(&dataConn, identity, buf[:n], sigSalt)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func sendDataWithSig(dataConnP *connection.ConnWrapper, identity Identity, data, sigSalt []byte) (int, error) {
	dataConn := *dataConnP

	n, err := dataConn.WriteBytes(data)
	if err != nil {
		return 0, err
	}

	m, err := dataConn.WriteBytes(identity.Sign(append(sigSalt, data...)))
	if err != nil {
		return n, err
	}

	err = dataConn.Flush()
	if err != nil {
		return 0, err
	}

	resp, err := dataConn.ReadString()
	if err != nil {
		return m + n, err
	} else if resp != "ok" {
		return m + n, fmt.Errorf("received response \"%s\" for msg meta", resp)
	}

	return m + n, nil
}
