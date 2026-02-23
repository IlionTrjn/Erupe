package main

import (
	"fmt"

	"erupe-ce/network"
	"erupe-ce/network/pcap"
)

// PacketDiff describes a difference between an expected and actual packet.
type PacketDiff struct {
	Index          int
	Expected       pcap.PacketRecord
	Actual         *pcap.PacketRecord // nil if no response received
	OpcodeMismatch bool
	SizeDelta      int
}

func (d PacketDiff) String() string {
	if d.Actual == nil {
		return fmt.Sprintf("#%d: expected 0x%04X (%s), got no response",
			d.Index, d.Expected.Opcode, network.PacketID(d.Expected.Opcode))
	}
	if d.OpcodeMismatch {
		return fmt.Sprintf("#%d: opcode mismatch: expected 0x%04X (%s), got 0x%04X (%s)",
			d.Index,
			d.Expected.Opcode, network.PacketID(d.Expected.Opcode),
			d.Actual.Opcode, network.PacketID(d.Actual.Opcode))
	}
	return fmt.Sprintf("#%d: 0x%04X (%s) size delta %+d bytes",
		d.Index, d.Expected.Opcode, network.PacketID(d.Expected.Opcode), d.SizeDelta)
}

// ComparePackets compares expected server responses against actual responses.
// Only compares S→C packets (server responses).
func ComparePackets(expected, actual []pcap.PacketRecord) []PacketDiff {
	expectedS2C := pcap.FilterByDirection(expected, pcap.DirServerToClient)
	actualS2C := pcap.FilterByDirection(actual, pcap.DirServerToClient)

	var diffs []PacketDiff
	for i, exp := range expectedS2C {
		if i >= len(actualS2C) {
			diffs = append(diffs, PacketDiff{
				Index:    i,
				Expected: exp,
				Actual:   nil,
			})
			continue
		}
		act := actualS2C[i]
		if exp.Opcode != act.Opcode {
			diffs = append(diffs, PacketDiff{
				Index:          i,
				Expected:       exp,
				Actual:         &act,
				OpcodeMismatch: true,
			})
		} else if len(exp.Payload) != len(act.Payload) {
			diffs = append(diffs, PacketDiff{
				Index:     i,
				Expected:  exp,
				Actual:    &act,
				SizeDelta: len(act.Payload) - len(exp.Payload),
			})
		}
	}

	// Extra actual packets beyond expected.
	for i := len(expectedS2C); i < len(actualS2C); i++ {
		act := actualS2C[i]
		diffs = append(diffs, PacketDiff{
			Index:    i,
			Expected: pcap.PacketRecord{},
			Actual:   &act,
		})
	}

	return diffs
}
