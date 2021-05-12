package sio

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
)

const (
	//16K
	maxBufSize = 2 ^ ^14
)

//DataConn is a helper struct to simplify communication over a net.Conn.
//Also tries to save bandwith by using a manually flushed bufio.ReadWriter.
//Has a artifical limit of 16K for message size.
type DataConn struct {
	buffer *bufio.ReadWriter
	conn   net.Conn
}

//NewDataIO creates a new DataConn from a net.Conn
func NewDataIO(conn net.Conn) *DataConn {
	return &DataConn{
		buffer: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		conn:   conn,
	}
}

//WriteBytes writes a byte slice to the connection.
//It returns the number of bytes written.
//If n < len(msg), it also returns an error explaining why the write is short.
func (d *DataConn) WriteBytes(msg []byte) (int, error) {
	if len(msg) > maxBufSize {
		return 0, fmt.Errorf("data cannot be larger %d to be sent", maxBufSize)
	}
	n, err := d.buffer.Write(append(intToBytes(len(msg)), msg...))
	return n, err
}

//ReadBytes reads a byte slice from the underlying connection
func (d *DataConn) ReadBytes() ([]byte, error) {
	l := make([]byte, 4)
	_, err := d.buffer.Read(l)
	if err != nil {
		return nil, err
	}

	bufSize := bytesToInt(l)
	if bufSize > maxBufSize {
		return nil, fmt.Errorf("%d exceeds buffer size limit", bufSize)
	}

	msg := make([]byte, bytesToInt(l))
	_, err = d.buffer.Read(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

//WriteString writes the specified string to the underlying connectrion
//It returns the number of bytes written.
//If n < len(msg), it also returns an error explaining why the write is short.
func (d *DataConn) WriteString(msg string) (int, error) {
	n, err := d.WriteBytes([]byte(msg))
	return n, err
}

//ReadString reads a string from the underlying connection
func (d *DataConn) ReadString() (string, error) {
	msg, err := d.ReadBytes()
	if err != nil {
		return "", err
	}

	return string(msg), nil
}

//WriteInt writes the specified int to the underlying connection
//It returns the number of bytes written.
//If n < 4, it also returns an error explaining why the write is short.
func (d *DataConn) WriteInt(msg int) (int, error) {
	n, err := d.WriteBytes(intToBytes(msg))
	return n, err
}

//ReadInt reades an int from the underlying connection
func (d *DataConn) ReadInt() (int, error) {
	msg, err := d.ReadBytes()
	if err != nil {
		return 0, err
	}

	return bytesToInt(msg), nil
}

//WriteStruct serializes and then writes the specified struct to the underlying connection
//It returns the number of bytes written.
//If n < len(json.Marshal(msg)), it also returns an error explaining why the write is short.
func (d *DataConn) WriteStruct(msg interface{}) (int, error) {
	m, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	return d.WriteBytes(m)
}

//ReadStruct reades an serialzed struct from the underlying connection and unmarshals it into the provided struct
func (d *DataConn) ReadStruct(target interface{}) error {
	raw, err := d.ReadBytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(raw, target)
}

//Flush writes any buffered data to the underlying io.Writer.
func (d *DataConn) Flush() error {
	//log.Printf("Buffered %d bytes before flushing\n", d.Buffered())
	return d.buffer.Flush()
}

//Close closes the connection. Any blocked Read or Write operations will be unblocked and return errors.
func (d *DataConn) Close() error {
	return d.conn.Close()
}

//Buffered returns the number of bytes that have been written into the current buffer.
func (d *DataConn) Buffered() int {
	return d.buffer.Writer.Buffered()
}

func bytesToInt(d []byte) int {
	return int(binary.LittleEndian.Uint32(d))
}

func intToBytes(i int) []byte {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(i))
	return bs
}
