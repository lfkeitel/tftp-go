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
	opRetransmit uint16 = 0
	opRead       uint16 = 1
	opWrite      uint16 = 2
	opData       uint16 = 3
	opAck        uint16 = 4
	opError      uint16 = 5
	opOAck       uint16 = 6
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
	errOptionsDenied    = 8
)

// TFTP transfer modes
const (
	modeNetascii = "netascii"
	modeOctet    = "octet"
)

// TFTP options
const (
	optionBulkSize     = "blksize"
	optionTimeout      = "timeout"
	optionTransferSize = "tsize"
)

var defaultOptions = &tftpOptions{
	blockSize:  512,
	timeout:    5 * time.Second,
	windowSize: 1,
	tsize:      -1,
}

type tftpOptions struct {
	oackSent   bool
	blockSize  int
	timeout    time.Duration
	windowSize int
	tsize      int64
}

func optionToString(op uint16) string {
	switch op {
	case opRetransmit:
		return "Retransmit"
	case opRead:
		return "Read"
	case opWrite:
		return "Write"
	case opData:
		return "Data"
	case opAck:
		return "Ack"
	case opError:
		return "Error"
	}
	return ""
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
