package types

import (
	"net"
	"encoding/binary"
)

func WriteCon(con net.Conn, msg []byte) (int, error) {
	n, err := con.Write(append(intToBytes(len(msg)), msg...))
	return n, err
}

func ReadCon(con net.Conn) ([]byte, error) {
	l := make([]byte, 4)
	_, err := con.Read(l)
	if err != nil {
		return nil, err
	}

	msg := make([]byte, bytesToInt(l))
	_, err = con.Read(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func bytesToInt(d []byte) int {
	return int(binary.LittleEndian.Uint32(d))
}

func intToBytes(i int) []byte {
	bs := make([]byte, 4)
    binary.LittleEndian.PutUint32(bs, uint32(i))
    return bs
}

func int64ToBytes(i int64) []byte {
	bs := make([]byte, 8)
    binary.LittleEndian.PutUint64(bs, uint64(i))
    return bs
}