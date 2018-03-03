package main

import (
	"strconv"
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

// TFTP transfer modes, for completeness. All connections are treated as octet mode.
const (
	modeNetascii = "netascii"
	modeOctet    = "octet"
	modeMail     = "mail"
)

// TFTP options
const (
	optionBlockSize    = "blksize"
	optionTimeout      = "timeout"
	optionTransferSize = "tsize"
	optionWindowSize   = "windowsize"
)

type tftpOptions struct {
	oackSent   bool
	blockSize  int
	timeout    time.Duration
	windowSize int
	tsize      int64
}

// defaultOptions should never be changed at runtime. These settings comply
// with RFC 1350 and will act as if no options were given if used as is.
var defaultOptions = &tftpOptions{
	blockSize:  512,
	timeout:    5 * time.Second,
	windowSize: 1,
	tsize:      -1,
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

func (o *tftpOptions) toMap() map[string]string {
	r := make(map[string]string)

	if o.blockSize != defaultOptions.blockSize {
		r[optionBlockSize] = strconv.Itoa(o.blockSize)
	}
	if o.timeout != defaultOptions.timeout {
		r[optionTimeout] = strconv.Itoa(int(o.timeout / time.Second))
	}
	if o.tsize > -1 {
		r[optionTransferSize] = strconv.FormatInt(o.tsize, 10)
	}
	if o.windowSize > -1 {
		r[optionWindowSize] = strconv.Itoa(o.windowSize)
	}

	return r
}
