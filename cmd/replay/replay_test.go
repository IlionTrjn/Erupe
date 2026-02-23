package main

import (
	"bytes"
	"os"
	"testing"

	"erupe-ce/network/pcap"
)

func createTestCapture(t *testing.T, records []pcap.PacketRecord) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.mhfr")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()

	hdr := pcap.FileHeader{
		Version:        pcap.FormatVersion,
		ServerType:     pcap.ServerTypeChannel,
		ClientMode:     40,
		SessionStartNs: 1000000000,
	}
	meta := pcap.SessionMetadata{Host: "127.0.0.1", Port: 54001}

	w, err := pcap.NewWriter(f, hdr, meta)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for _, r := range records {
		if err := w.WritePacket(r); err != nil {
			t.Fatalf("WritePacket: %v", err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	return f.Name()
}

func TestRunDump(t *testing.T) {
	path := createTestCapture(t, []pcap.PacketRecord{
		{TimestampNs: 1000000100, Direction: pcap.DirClientToServer, Opcode: 0x0013, Payload: []byte{0x00, 0x13}},
		{TimestampNs: 1000000200, Direction: pcap.DirServerToClient, Opcode: 0x0012, Payload: []byte{0x00, 0x12, 0xFF}},
	})
	// Just verify it doesn't error.
	if err := runDump(path); err != nil {
		t.Fatalf("runDump: %v", err)
	}
}

func TestRunStats(t *testing.T) {
	path := createTestCapture(t, []pcap.PacketRecord{
		{TimestampNs: 1000000100, Direction: pcap.DirClientToServer, Opcode: 0x0013, Payload: []byte{0x00, 0x13}},
		{TimestampNs: 1000000200, Direction: pcap.DirServerToClient, Opcode: 0x0012, Payload: []byte{0x00, 0x12, 0xFF}},
		{TimestampNs: 1000000300, Direction: pcap.DirClientToServer, Opcode: 0x0013, Payload: []byte{0x00, 0x13, 0xAA}},
	})
	if err := runStats(path); err != nil {
		t.Fatalf("runStats: %v", err)
	}
}

func TestRunStatsEmpty(t *testing.T) {
	path := createTestCapture(t, nil)
	if err := runStats(path); err != nil {
		t.Fatalf("runStats empty: %v", err)
	}
}

func TestRunJSON(t *testing.T) {
	path := createTestCapture(t, []pcap.PacketRecord{
		{TimestampNs: 1000000100, Direction: pcap.DirClientToServer, Opcode: 0x0013, Payload: []byte{0x00, 0x13}},
	})
	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := runJSON(path); err != nil {
		os.Stdout = old
		t.Fatalf("runJSON: %v", err)
	}

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if buf.Len() == 0 {
		t.Error("runJSON produced no output")
	}
	// Should be valid JSON containing "packets".
	if !bytes.Contains(buf.Bytes(), []byte(`"packets"`)) {
		t.Error("runJSON output missing 'packets' key")
	}
}

func TestComparePackets(t *testing.T) {
	expected := []pcap.PacketRecord{
		{Direction: pcap.DirClientToServer, Opcode: 0x0013, Payload: []byte{0x00, 0x13}},
		{Direction: pcap.DirServerToClient, Opcode: 0x0012, Payload: []byte{0x00, 0x12, 0xAA}},
		{Direction: pcap.DirServerToClient, Opcode: 0x0061, Payload: []byte{0x00, 0x61}},
	}
	actual := []pcap.PacketRecord{
		{Direction: pcap.DirServerToClient, Opcode: 0x0012, Payload: []byte{0x00, 0x12, 0xBB, 0xCC}}, // size diff
		{Direction: pcap.DirServerToClient, Opcode: 0x0099, Payload: []byte{0x00, 0x99}},             // opcode mismatch
	}

	diffs := ComparePackets(expected, actual)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	// First diff: size delta.
	if diffs[0].SizeDelta != 1 {
		t.Errorf("diffs[0] SizeDelta = %d, want 1", diffs[0].SizeDelta)
	}

	// Second diff: opcode mismatch.
	if !diffs[1].OpcodeMismatch {
		t.Error("diffs[1] expected OpcodeMismatch=true")
	}
}

func TestComparePacketsMissingResponse(t *testing.T) {
	expected := []pcap.PacketRecord{
		{Direction: pcap.DirServerToClient, Opcode: 0x0012, Payload: []byte{0x00, 0x12}},
		{Direction: pcap.DirServerToClient, Opcode: 0x0061, Payload: []byte{0x00, 0x61}},
	}
	actual := []pcap.PacketRecord{
		{Direction: pcap.DirServerToClient, Opcode: 0x0012, Payload: []byte{0x00, 0x12}},
	}

	diffs := ComparePackets(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Actual != nil {
		t.Error("expected nil Actual for missing response")
	}
}

func TestPacketDiffString(t *testing.T) {
	d := PacketDiff{
		Index:    0,
		Expected: pcap.PacketRecord{Opcode: 0x0012},
		Actual:   nil,
	}
	s := d.String()
	if s == "" {
		t.Error("PacketDiff.String() returned empty")
	}
}
