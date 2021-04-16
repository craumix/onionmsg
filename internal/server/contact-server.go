package server

import (
	"log"
	"net"
	"strconv"

	"github.com/Craumix/tormsg/internal/types"
	"github.com/google/uuid"
)

func StartContactServer(port int, identities map[string]*types.Identity) (error) {
	server, err := net.Listen("tcp", "localhost:" + strconv.Itoa(port))
	if err != nil {
		return err
	}

	defer server.Close()

	for {
		c, err := server.Accept()
		if err != nil {
			log.Println(err)
		}

		go func() {
			dconn := types.NewDataIO(c)

			contactFingerprint, err := dconn.ReadString()
			if err != nil {
				log.Println(err.Error())
				return
			}

			if identities[contactFingerprint] == nil {
				log.Printf("Contact id %s unknown\n", contactFingerprint)
				return
			}
			
			msg, err := dconn.ReadBytes()
			if err != nil {
				log.Println(err.Error())
				return
			}
			id, _ := uuid.FromBytes(msg)

			convID := types.NewIdentity()
			_, err = dconn.WriteString(convID.Fingerprint())
			if err != nil {
				log.Println(err.Error())
				return
			}

			_, err = dconn.WriteBytes(identities[contactFingerprint].Sign(append([]byte(convID.Fingerprint()), id[:]...)))
			if err != nil {
				log.Println(err.Error())
				return
			}

			dconn.Flush()
			dconn.Close()

			log.Printf("Exchange succesfull uuid %s sent id %s", id, convID.Fingerprint())
		}()
	}
}