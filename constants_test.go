package main

import (
	"bytes"
	"testing"
)

var opCodeTests = []struct {
	bytes    []byte
	expected uint16
}{
	{
		bytes:    []byte{0, 1},
		expected: 1,
	},
	{
		bytes:    []byte{0, 2},
		expected: 2,
	},
	{
		bytes:    []byte{0, 3},
		expected: 3,
	},
	{
		bytes:    []byte{0, 4},
		expected: 4,
	},
	{
		bytes:    []byte{0, 5},
		expected: 5,
	},
	{
		bytes:    []byte{0, 1, 0},
		expected: 0,
	},
}

func TestDecodeOpCode(t *testing.T) {
	for _, test := range opCodeTests {
		got := decodeUInt16(test.bytes)
		if got != test.expected {
			t.Errorf("Opcode decode: Expected %d, got %d", test.expected, got)
		}
	}
}

func TestEncodeUInt(t *testing.T) {
	test := uint16(16)

	out := encodeUInt16(test)
	if !bytes.Equal(out, []byte{0, 16}) {
		t.Errorf("Expected []byte{0, 16} got %v", out)
	}
}
