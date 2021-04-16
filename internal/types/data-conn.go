package types

import (
	"bufio"
	"encoding/binary"
	"net"
)

type DataConn struct {
	buffer	*bufio.ReadWriter
	conn	net.Conn
}

func NewDataIO(conn net.Conn) *DataConn {
	return &DataConn{
		buffer: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		conn: conn,
	}
}

func (d *DataConn) WriteBytes(msg []byte) (int, error) {
	n, err := d.buffer.Write(append(intToBytes(len(msg)), msg...))
	return n, err
}

func (d *DataConn) ReadBytes() ([]byte, error) {
	l := make([]byte, 4)
	_, err := d.buffer.Read(l)
	if err != nil {
		return nil, err
	}

	msg := make([]byte, bytesToInt(l))
	_, err = d.buffer.Read(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (d *DataConn) WriteString(msg string) (int, error) {
	n, err := d.WriteBytes([]byte(msg))
	return n, err
}

func (d *DataConn) ReadString() (string, error) {
	msg, err := d.ReadBytes()
	if err != nil {
		return "", err
	}

	return string(msg), nil
}

func (d *DataConn) Flush() {
	d.buffer.Flush()
}

func (d *DataConn) Close() error {
	return d.conn.Close()
}

func bytesToInt(d []byte) int {
	return int(binary.LittleEndian.Uint32(d))
}

func intToBytes(i int) []byte {
	bs := make([]byte, 4)
    binary.LittleEndian.PutUint32(bs, uint32(i))
    return bs
}