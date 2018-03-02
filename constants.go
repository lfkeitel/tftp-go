package main

import (
	"time"
)

const (
	tftpPort       = 69
	maxRetransmits = 5
)

type opCode uint16

// TFTP op codes
const (
	opRetransmit opCode = 0
	opRead       opCode = 1
	opWrite      opCode = 2
	opData       opCode = 3
	opAck        opCode = 4
	opError      opCode = 5
	opOAck       opCode = 6
)

func (op opCode) String() string {
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

type tftpError uint16

// TFTP error codes
const (
	errNotDefined       tftpError = 0
	errFileNotFound     tftpError = 1
	errAccessViolation  tftpError = 2
	errDiskFull         tftpError = 3
	errIllegalOperation tftpError = 4
	errUnknownTID       tftpError = 5
	errFileExists       tftpError = 6
	errNoSuchUser       tftpError = 7
	errOptionsDenied    tftpError = 8
)

// TFTP transfer modes, netascii is not acually implemented. All connections are assumed to be octet mode.
const (
	modeNetascii = "netascii"
	modeOctet    = "octet"
)

// TFTP options
const (
	optionBlockSize    = "blksize"
	optionTimeout      = "timeout"
	optionTransferSize = "tsize"
)

// defaultOptions should never be changed at runtime. These settings comply
// with RFC 1350 and will act as if no options were given if used as is.
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

func (o *tftpOptions) copy() *tftpOptions {
	return &tftpOptions{
		oackSent:   o.oackSent,
		blockSize:  o.blockSize,
		timeout:    o.timeout,
		windowSize: o.windowSize,
		tsize:      o.tsize,
	}
}
