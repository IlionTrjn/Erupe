// replay is a CLI tool for inspecting and replaying .mhfr packet capture files.
//
// Usage:
//
//	replay --capture file.mhfr --mode dump     # Human-readable text output
//	replay --capture file.mhfr --mode json     # JSON export
//	replay --capture file.mhfr --mode stats    # Opcode histogram, duration, counts
//	replay --capture file.mhfr --mode replay --target 127.0.0.1:54001  # Replay against live server
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"erupe-ce/network"
	"erupe-ce/network/pcap"
)

func main() {
	capturePath := flag.String("capture", "", "Path to .mhfr capture file (required)")
	mode := flag.String("mode", "dump", "Mode: dump, json, stats, replay")
	target := flag.String("target", "", "Target server address for replay mode (host:port)")
	speed := flag.Float64("speed", 1.0, "Replay speed multiplier (e.g. 2.0 = 2x faster)")
	_ = target // used in replay mode
	_ = speed
	flag.Parse()

	if *capturePath == "" {
		fmt.Fprintln(os.Stderr, "error: --capture is required")
		flag.Usage()
		os.Exit(1)
	}

	switch *mode {
	case "dump":
		if err := runDump(*capturePath); err != nil {
			fmt.Fprintf(os.Stderr, "dump failed: %v\n", err)
			os.Exit(1)
		}
	case "json":
		if err := runJSON(*capturePath); err != nil {
			fmt.Fprintf(os.Stderr, "json failed: %v\n", err)
			os.Exit(1)
		}
	case "stats":
		if err := runStats(*capturePath); err != nil {
			fmt.Fprintf(os.Stderr, "stats failed: %v\n", err)
			os.Exit(1)
		}
	case "replay":
		if *target == "" {
			fmt.Fprintln(os.Stderr, "error: --target is required for replay mode")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "replay mode not yet implemented (requires live server connection)")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func openCapture(path string) (*pcap.Reader, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open capture: %w", err)
	}
	r, err := pcap.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, nil, fmt.Errorf("read capture: %w", err)
	}
	return r, f, nil
}

func readAllPackets(r *pcap.Reader) ([]pcap.PacketRecord, error) {
	var records []pcap.PacketRecord
	for {
		rec, err := r.ReadPacket()
		if err == io.EOF {
			break
		}
		if err != nil {
			return records, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func runDump(path string) error {
	r, f, err := openCapture(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Print header info.
	startTime := time.Unix(0, r.Header.SessionStartNs)
	fmt.Printf("=== MHFR Capture: %s ===\n", path)
	fmt.Printf("Server: %s  ClientMode: %d  Start: %s\n",
		r.Header.ServerType, r.Header.ClientMode, startTime.Format(time.RFC3339Nano))
	if r.Meta.Host != "" {
		fmt.Printf("Host: %s  Port: %d  Remote: %s\n", r.Meta.Host, r.Meta.Port, r.Meta.RemoteAddr)
	}
	if r.Meta.CharID != 0 {
		fmt.Printf("CharID: %d  UserID: %d\n", r.Meta.CharID, r.Meta.UserID)
	}
	fmt.Println()

	records, err := readAllPackets(r)
	if err != nil {
		return err
	}

	for i, rec := range records {
		elapsed := time.Duration(rec.TimestampNs - r.Header.SessionStartNs)
		opcodeName := network.PacketID(rec.Opcode).String()
		fmt.Printf("#%04d  +%-12s  %s  0x%04X %-30s  %d bytes\n",
			i, elapsed, rec.Direction, rec.Opcode, opcodeName, len(rec.Payload))
	}

	fmt.Printf("\nTotal: %d packets\n", len(records))
	return nil
}

type jsonCapture struct {
	Header  jsonHeader           `json:"header"`
	Meta    pcap.SessionMetadata `json:"metadata"`
	Packets []jsonPacket         `json:"packets"`
}

type jsonHeader struct {
	Version    uint16 `json:"version"`
	ServerType string `json:"server_type"`
	ClientMode int    `json:"client_mode"`
	StartTime  string `json:"start_time"`
}

type jsonPacket struct {
	Index      int    `json:"index"`
	Timestamp  string `json:"timestamp"`
	ElapsedNs  int64  `json:"elapsed_ns"`
	Direction  string `json:"direction"`
	Opcode     uint16 `json:"opcode"`
	OpcodeName string `json:"opcode_name"`
	PayloadLen int    `json:"payload_len"`
}

func runJSON(path string) error {
	r, f, err := openCapture(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	records, err := readAllPackets(r)
	if err != nil {
		return err
	}

	out := jsonCapture{
		Header: jsonHeader{
			Version:    r.Header.Version,
			ServerType: r.Header.ServerType.String(),
			ClientMode: int(r.Header.ClientMode),
			StartTime:  time.Unix(0, r.Header.SessionStartNs).Format(time.RFC3339Nano),
		},
		Meta:    r.Meta,
		Packets: make([]jsonPacket, len(records)),
	}

	for i, rec := range records {
		out.Packets[i] = jsonPacket{
			Index:      i,
			Timestamp:  time.Unix(0, rec.TimestampNs).Format(time.RFC3339Nano),
			ElapsedNs:  rec.TimestampNs - r.Header.SessionStartNs,
			Direction:  rec.Direction.String(),
			Opcode:     rec.Opcode,
			OpcodeName: network.PacketID(rec.Opcode).String(),
			PayloadLen: len(rec.Payload),
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func runStats(path string) error {
	r, f, err := openCapture(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	records, err := readAllPackets(r)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("Empty capture (0 packets)")
		return nil
	}

	// Compute stats.
	type opcodeStats struct {
		opcode uint16
		count  int
		bytes  int
	}
	statsMap := make(map[uint16]*opcodeStats)
	var totalC2S, totalS2C int
	var bytesC2S, bytesS2C int

	for _, rec := range records {
		s, ok := statsMap[rec.Opcode]
		if !ok {
			s = &opcodeStats{opcode: rec.Opcode}
			statsMap[rec.Opcode] = s
		}
		s.count++
		s.bytes += len(rec.Payload)

		switch rec.Direction {
		case pcap.DirClientToServer:
			totalC2S++
			bytesC2S += len(rec.Payload)
		case pcap.DirServerToClient:
			totalS2C++
			bytesS2C += len(rec.Payload)
		}
	}

	// Sort by count descending.
	sorted := make([]*opcodeStats, 0, len(statsMap))
	for _, s := range statsMap {
		sorted = append(sorted, s)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	duration := time.Duration(records[len(records)-1].TimestampNs - records[0].TimestampNs)

	fmt.Printf("=== Capture Stats: %s ===\n", path)
	fmt.Printf("Server: %s  Duration: %s  Packets: %d\n",
		r.Header.ServerType, duration, len(records))
	fmt.Printf("C→S: %d packets (%d bytes)  S→C: %d packets (%d bytes)\n\n",
		totalC2S, bytesC2S, totalS2C, bytesS2C)

	fmt.Printf("%-8s %-35s %8s %10s\n", "Opcode", "Name", "Count", "Bytes")
	fmt.Printf("%-8s %-35s %8s %10s\n", "------", "----", "-----", "-----")
	for _, s := range sorted {
		name := network.PacketID(s.opcode).String()
		fmt.Printf("0x%04X   %-35s %8d %10d\n", s.opcode, name, s.count, s.bytes)
	}

	return nil
}
