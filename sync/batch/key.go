package batch

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
)

// defaultShardKey hashes k using FNV-32a.
// For the common types (string, int family) it avoids fmt allocations.
func defaultShardKey[K comparable](k K) int {
	var b8 [8]byte
	h := fnv.New32a()
	switch v := any(k).(type) {
	case string:
		_, _ = io.WriteString(h, v)
	case int:
		binary.LittleEndian.PutUint64(b8[:], uint64(v))
		_, _ = h.Write(b8[:])
	case int8:
		b8[0] = byte(v)
		_, _ = h.Write(b8[:1])
	case int16:
		binary.LittleEndian.PutUint16(b8[:], uint16(v))
		_, _ = h.Write(b8[:2])
	case int32:
		binary.LittleEndian.PutUint32(b8[:], uint32(v))
		_, _ = h.Write(b8[:4])
	case int64:
		binary.LittleEndian.PutUint64(b8[:], uint64(v))
		_, _ = h.Write(b8[:])
	case uint:
		binary.LittleEndian.PutUint64(b8[:], uint64(v))
		_, _ = h.Write(b8[:])
	case uint8:
		b8[0] = v
		_, _ = h.Write(b8[:1])
	case uint16:
		binary.LittleEndian.PutUint16(b8[:], v)
		_, _ = h.Write(b8[:2])
	case uint32:
		binary.LittleEndian.PutUint32(b8[:], v)
		_, _ = h.Write(b8[:4])
	case uint64:
		binary.LittleEndian.PutUint64(b8[:], v)
		_, _ = h.Write(b8[:])
	default:
		_, _ = fmt.Fprintf(h, "%v", v)
	}
	return int(h.Sum32())
}
