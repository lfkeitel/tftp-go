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

const (
	modeNetascii = "netascii"
	modeOctet    = "octet"
)

type options struct {
	blockSize  int
	timeout    time.Duration
	windowSize int
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

func sendReadWriteRequest(conn *requestConn, op uint16, filename, mode string) {
	if op != opWrite && op != opRead {
		return
	}

	filenameBytes := []byte(filename)
	modeBytes := []byte(mode)
	resp := make([]byte, 4+len(filenameBytes)+len(modeBytes))
	// Op code
	resp[0] = byte(op >> 8)
	resp[1] = byte(op)
	// Filename
	copy(resp[2:len(filenameBytes)+2], filenameBytes)
	resp[len(filenameBytes)+2] = 0 // Null terminator
	// Mode
	copy(resp[len(filenameBytes)+3:len(resp)], modeBytes)
	resp[len(resp)-1] = 0 // Null terminator

	conn.conn.WriteTo(resp, conn.addr)
}

func sendData(conn *requestConn, blockID uint16, data []byte) {
	resp := make([]byte, 4+len(data))
	// Op code
	resp[0] = byte(opData >> 8)
	resp[1] = byte(opData)
	// Block #
	resp[2] = byte(blockID >> 8)
	resp[3] = byte(blockID)
	// Data
	copy(resp[4:], data)

	conn.conn.WriteTo(resp, conn.addr)
}

func sendAck(conn *requestConn, blockID uint16) {
	resp := make([]byte, 4)
	// Op code
	resp[0] = byte(opAck >> 8)
	resp[1] = byte(opAck)
	// Block #
	resp[2] = byte(blockID >> 8)
	resp[3] = byte(blockID)

	conn.conn.WriteTo(resp, conn.addr)
}

func writeError(conn *requestConn, code int, msg string) {
	msgBytes := []byte(msg)

	resp := make([]byte, 5+len(msgBytes))
	// Op code
	resp[0] = byte(opError >> 8)
	resp[1] = byte(opError)
	// Error code
	resp[2] = byte(code >> 8)
	resp[3] = byte(code)
	// Human-readable message
	copy(resp[4:len(resp)-1], msgBytes)
	// Null terminator
	resp[len(resp)-1] = 0

	conn.conn.WriteTo(resp, conn.addr)
}
