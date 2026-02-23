package pcap

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

// mockConn implements network.Conn for testing.
type mockConn struct {
	readData [][]byte
	readIdx  int
	sent     [][]byte
	mu       sync.Mutex
}

func (m *mockConn) ReadPacket() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readIdx >= len(m.readData) {
		return nil, io.EOF
	}
	data := m.readData[m.readIdx]
	m.readIdx++
	return data, nil
}

func (m *mockConn) SendPacket(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.sent = append(m.sent, cp)
	return nil
}

func TestRecordingConnBasic(t *testing.T) {
	mock := &mockConn{
		readData: [][]byte{
			{0x00, 0x13, 0xDE, 0xAD}, // opcode 0x0013
		},
	}

	var buf bytes.Buffer
	hdr := FileHeader{
		Version:        FormatVersion,
		ServerType:     ServerTypeChannel,
		ClientMode:     40,
		SessionStartNs: 1000,
	}
	w, err := NewWriter(&buf, hdr, SessionMetadata{})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	rc := NewRecordingConn(mock, w, 1000)

	// Read a packet (C→S).
	data, err := rc.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if !bytes.Equal(data, []byte{0x00, 0x13, 0xDE, 0xAD}) {
		t.Errorf("ReadPacket data mismatch")
	}

	// Send a packet (S→C).
	sendData := []byte{0x00, 0x12, 0xBE, 0xEF}
	if err := rc.SendPacket(sendData); err != nil {
		t.Fatalf("SendPacket: %v", err)
	}

	// Flush and read back.
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	// First record: C→S.
	rec, err := r.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket[0]: %v", err)
	}
	if rec.Direction != DirClientToServer {
		t.Errorf("rec[0] direction = %v, want C→S", rec.Direction)
	}
	if rec.Opcode != 0x0013 {
		t.Errorf("rec[0] opcode = 0x%04X, want 0x0013", rec.Opcode)
	}

	// Second record: S→C.
	rec, err = r.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket[1]: %v", err)
	}
	if rec.Direction != DirServerToClient {
		t.Errorf("rec[1] direction = %v, want S→C", rec.Direction)
	}
	if rec.Opcode != 0x0012 {
		t.Errorf("rec[1] opcode = 0x%04X, want 0x0012", rec.Opcode)
	}

	// EOF.
	_, err = r.ReadPacket()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestRecordingConnConcurrent(t *testing.T) {
	// Generate enough packets for concurrent stress.
	const numPackets = 100
	readData := make([][]byte, numPackets)
	for i := range readData {
		readData[i] = []byte{byte(i >> 8), byte(i), 0xAA}
	}

	mock := &mockConn{readData: readData}

	var buf bytes.Buffer
	hdr := FileHeader{
		Version:        FormatVersion,
		ServerType:     ServerTypeChannel,
		ClientMode:     40,
		SessionStartNs: 1000,
	}
	w, err := NewWriter(&buf, hdr, SessionMetadata{})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	rc := NewRecordingConn(mock, w, 1000)

	// Concurrent reads and sends.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < numPackets; i++ {
			_, _ = rc.ReadPacket()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numPackets; i++ {
			_ = rc.SendPacket([]byte{byte(i >> 8), byte(i), 0xBB})
		}
	}()

	wg.Wait()

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// Verify all 200 records can be read back.
	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	count := 0
	for {
		_, err := r.ReadPacket()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadPacket: %v", err)
		}
		count++
	}
	if count != 2*numPackets {
		t.Errorf("got %d records, want %d", count, 2*numPackets)
	}
}
