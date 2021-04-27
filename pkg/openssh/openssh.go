package openssh

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
)

func FromFile(filename string) (ed25519.PrivateKey, error) {
	pemBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return FromPem(pemBytes)
}

func FromPem(pemBytes []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("unable to get pem block")
	}

	return FromBytes(block.Bytes)
}

func FromBytes(raw []byte) (ed25519.PrivateKey, error) {
	magicNullterm := bytes.Index(raw, []byte{0x00})
	authMagic := string(raw[:magicNullterm])
	raw = raw[magicNullterm+1:]
	if authMagic != "openssh-key-v1" {
		return nil, fmt.Errorf("invalid auth magic %s", authMagic)
	}

	raw, ciphername := readNextString(raw)
	_ = ciphername
	raw, kdfname := readNextString(raw)
	raw, kdfopts := readNextBytes(raw)
	raw, num := readNextInt(raw)
	if num != 0x01 {
		return nil, fmt.Errorf("invalid value for number of keys %d", num)
	}
	
	if kdfname == "bcrypt" && len(kdfopts) == 24 {
		foo, salt := readNextBytes(kdfopts)
		log.Println(hex.EncodeToString(salt))
		_, work := readNextInt(foo)
		log.Println(work)
	}

	//Skip Public-Key length
	raw = raw[4:]

	//Skip Public Key
	raw, _ = readNextString(raw)
	raw, _ = readNextBytes(raw)

	raw, payloadSize := readNextInt(raw)

	if ciphername == "none" && kdfname == "none" {
		return keyFromPayload(raw[:payloadSize])
	}else {
		return nil, nil
	}
}

func keyFromPayload(payload []byte) (ed25519.PrivateKey, error) {
	//Skip weird 8 bytes
	payload = payload[8:]

	payload, keytype := readNextString(payload)
	if keytype != "ssh-ed25519" {
		return nil, fmt.Errorf("key is not of type ssh-ed25519 but of %s", keytype)
	}

	//Skip what is probably a duplicate of the public key
	payload, _ = readNextBytes(payload)

	payload, privatekey := readNextBytes(payload)
	if len(privatekey) != 64 {
		return nil, fmt.Errorf("length of private key is not 64 but %d", len(privatekey))
	}

	//Comment
	payload, _ = readNextString(payload)

	return ed25519.PrivateKey(privatekey), nil
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
