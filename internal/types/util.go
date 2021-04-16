package types

import (
	"encoding/binary"
)

func int64ToBytes(i int64) []byte {
	bs := make([]byte, 8)
    binary.LittleEndian.PutUint64(bs, uint64(i))
    return bs
}