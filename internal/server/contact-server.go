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
			con := c

			msg, err := types.ReadCon(con)
			if err != nil {
				log.Println(err.Error())
				return
			}

			sentFingerprint := string(msg)
			if identities[string(msg)] == nil {
				log.Printf("Contact id %s unknown\n", string(msg))
				return
			}
			
			msg, err = types.ReadCon(con)
			if err != nil {
				log.Println(err.Error())
				return
			}
			id, _ := uuid.FromBytes(msg)

			convID := types.NewIdentity()

			_, err = types.WriteCon(con, []byte(convID.Fingerprint()))
			if err != nil {
				log.Println(err.Error())
				return
			}

			_, err = types.WriteCon(con, identities[sentFingerprint].Sign(append([]byte(convID.Fingerprint()), id[:]...)))
			if err != nil {
				log.Println(err.Error())
				return
			}

			con.Close()

			log.Printf("Exchange succesfull uuid %s sent id %s", id, convID.Fingerprint())
		}()
	}
}