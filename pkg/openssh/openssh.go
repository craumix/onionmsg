package openssh

import (
	"bytes"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

type OpenSSHKeyfile struct {
	AuthMagic  string
	Ciphername string
	KDFName    string
	KDFOpts    []byte
	Comment    string

	PublicKey  OpenSSHKey
	PrivateKey OpenSSHKey
}

type OpenSSHKey struct {
	Type    string
	Content []byte
}

func FromFile(filename string) (*OpenSSHKeyfile, error) {
	pemBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return FromPem(pemBytes)
}

func FromPem(pemBytes []byte) (*OpenSSHKeyfile, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("unable to get pem block")
	}

	return FromBytes(block.Bytes)
}

func FromBytes(raw []byte) (*OpenSSHKeyfile, error) {
	keyfile := &OpenSSHKeyfile{}

	magicNullterm := bytes.Index(raw, []byte{0x00})
	keyfile.AuthMagic = string(raw[:magicNullterm])
	raw = raw[magicNullterm+1:]

	raw, keyfile.Ciphername = readNextString(raw)
	raw, keyfile.KDFName = readNextString(raw)
	raw, keyfile.KDFOpts = readNextBytes(raw)
	raw, num := readNextInt(raw)
	if num != 0x01 {
		return nil, fmt.Errorf("invalid value for number of keys %d", num)
	}

	//Skip Public-Key length
	raw = raw[4:]

	raw, publicType := readNextString(raw)
	raw, publicContent := readNextBytes(raw)
	keyfile.PublicKey = OpenSSHKey{
		Type:    publicType,
		Content: publicContent,
	}

	//Skip payload size
	raw = raw[4:]
	//Skip weird 8 bytes
	raw = raw[8:]

	raw, privateType := readNextString(raw)
	//Skip what is probably a duplicate of the public key
	raw, _ = readNextBytes(raw)
	raw, privateContent := readNextBytes(raw)
	keyfile.PrivateKey = OpenSSHKey{
		Type:    privateType,
		Content: privateContent,
	}

	raw, keyfile.Comment = readNextString(raw)

	return keyfile, nil
}

func readNextInt(raw []byte) (newRaw []byte, length int) {
	length = int(binary.BigEndian.Uint32(raw[:4]))
	newRaw = raw[4:]
	return
}

func readNextBytes(raw []byte) (newRaw, value []byte) {
	raw, length := readNextInt(raw)
	value = raw[:length]
	newRaw = raw[length:]
	return
}

func readNextString(raw []byte) (newRaw []byte, value string) {
	newRaw, tmp := readNextBytes(raw)
	value = string(tmp)
	return
}
