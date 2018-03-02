package main

import (
	"log"
	"net"
	"time"
)

type requestConn struct {
	conn net.PacketConn
	addr net.Addr
}

func (conn *requestConn) sendReadRequest(filename, mode string) {
	conn.sendRWRequest(opRead, filename, mode)
}

func (conn *requestConn) sendWriteRequest(filename, mode string) {
	conn.sendRWRequest(opWrite, filename, mode)
}

func (conn *requestConn) sendRWRequest(op opCode, filename, mode string) {
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
	copy(resp[len(filenameBytes)+3:], modeBytes)
	resp[len(resp)-1] = 0 // Null terminator

	conn.conn.WriteTo(resp, conn.addr)
}

func (conn *requestConn) sendData(blockID uint16, data []byte) {
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

func (conn *requestConn) sendAck(blockID uint16) {
	resp := make([]byte, 4)
	// Op code
	resp[0] = byte(opAck >> 8)
	resp[1] = byte(opAck)
	// Block #
	resp[2] = byte(blockID >> 8)
	resp[3] = byte(blockID)

	conn.conn.WriteTo(resp, conn.addr)
}

func (conn *requestConn) sendOAck(options map[string]string) {
	resp := make([]byte, 2, 2+totalStringMapLen(options))
	// Op code
	resp[0] = byte(opOAck >> 8)
	resp[1] = byte(opOAck)
	// Options
	for k, v := range options {
		resp = append(resp, []byte(k)...)
		resp = append(resp, 0)
		resp = append(resp, []byte(v)...)
		resp = append(resp, 0)
	}

	conn.conn.WriteTo(resp, conn.addr)
}

func (conn *requestConn) sendError(code tftpError, msg string) {
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

func (conn *requestConn) readNextMessage(op opCode, options *tftpOptions) *response {
	var buffer []byte
	if op == opRead {
		buffer = make([]byte, defaultOptions.blockSize)
	} else {
		buffer = make([]byte, options.blockSize+4)
	}

	conn.conn.SetReadDeadline(time.Now().Add(options.timeout))

	n, addr, err := conn.conn.ReadFrom(buffer)
	if err != nil {
		netErr := err.(net.Error)
		if netErr.Timeout() {
			return &response{op: opRetransmit}
		}
		log.Println(err)
		return nil
	}

	conn.addr = addr

	if n < 4 {
		conn.sendError(errNotDefined, "Malformatted message")
		return nil
	}

	recv := buffer[:n]

	opcode := opCode(decodeUInt16(recv[:2]))

	switch opcode {
	case opAck:
		return &response{
			op:      opAck,
			blockID: decodeUInt16(recv[2:4]),
		}
	case opError:
		errorMsg := ""
		if len(recv) > 4 {
			errorMsg = string(recv[4 : len(recv)-2]) // Strip null terminator
		}

		return &response{
			op:        opError,
			errorCode: decodeUInt16(recv[2:4]),
			errorMsg:  errorMsg,
		}
	case opData:
		return &response{
			op:      opData,
			blockID: decodeUInt16(recv[2:4]),
			data:    recv[4:],
		}
	default:
		conn.sendError(errIllegalOperation, "")
		return nil
	}
}
