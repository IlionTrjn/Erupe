package pcap

import (
	"encoding/binary"
	"erupe-ce/network"
	"sync"
	"time"
)

// RecordingConn wraps a network.Conn and records all packets to a Writer.
// It is safe for concurrent use from separate send/recv goroutines.
type RecordingConn struct {
	inner   network.Conn
	writer  *Writer
	startNs int64
	mu      sync.Mutex
}

// NewRecordingConn wraps inner, recording all packets to w.
// startNs is the session start time in nanoseconds (used as the time base).
func NewRecordingConn(inner network.Conn, w *Writer, startNs int64) *RecordingConn {
	return &RecordingConn{
		inner:   inner,
		writer:  w,
		startNs: startNs,
	}
}

// ReadPacket reads from the inner connection and records the packet as client-to-server.
func (rc *RecordingConn) ReadPacket() ([]byte, error) {
	data, err := rc.inner.ReadPacket()
	if err != nil {
		return data, err
	}
	rc.record(DirClientToServer, data)
	return data, nil
}

// SendPacket sends via the inner connection and records the packet as server-to-client.
func (rc *RecordingConn) SendPacket(data []byte) error {
	err := rc.inner.SendPacket(data)
	if err != nil {
		return err
	}
	rc.record(DirServerToClient, data)
	return nil
}

func (rc *RecordingConn) record(dir Direction, data []byte) {
	var opcode uint16
	if len(data) >= 2 {
		opcode = binary.BigEndian.Uint16(data[:2])
	}

	rec := PacketRecord{
		TimestampNs: time.Now().UnixNano(),
		Direction:   dir,
		Opcode:      opcode,
		Payload:     data,
	}

	rc.mu.Lock()
	_ = rc.writer.WritePacket(rec)
	rc.mu.Unlock()
}
