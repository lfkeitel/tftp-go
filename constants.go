package main

import (
	"os"
	"time"
)

const (
	tftpPort       = 69
	maxRetransmits = 5
)

// TFTP op codes
const (
	opRetransmit = 0
	opRead       = 1
	opWrite      = 2
	opData       = 3
	opAck        = 4
	opError      = 5
)

// TFTP error codes
const (
	errNotDefined       = 0
	errFileNotFound     = 1
	errAccessViolation  = 2
	errDiskFull         = 3
	errIllegalOperation = 4
	errUnknownTID       = 5
	errFileExists       = 6
	errNoSuchUser       = 7
)

type options struct {
	blockSize  int
	timeout    time.Duration
	windowSize int
}

func decodeUInt16(op []byte) uint16 {
	var code uint16

	switch len(op) {
	case 1:
		code = uint16(op[0])
	case 2:
		code = uint16(op[0]) << 8
		code = code + uint16(op[1])
	}

	return code
}

func encodeUInt16(in uint16) []byte {
	out := make([]byte, 2)
	out[0] = byte(in >> 8)
	out[1] = byte(in)
	return out
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
