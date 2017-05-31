package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

var rootDir = ""

func main() {
	log.Println("Start TFTP server...")

	rootDir = os.Args[1]
	listenAndServe()
}

type requestConn struct {
	conn net.PacketConn
	addr net.Addr
}

func listenAndServe() {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", tftpPort))
	if err != nil {
		log.Println(err)
		return
	}

	buffer := make([]byte, 512)
	for {
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Println(err)
			return
		}

		req := buffer[:n]

		opcode := decodeUInt16(req[:2])
		reqFields := bytes.Split(req[2:], []byte{0})
		reqFields = reqFields[:len(reqFields)-1] // Remove empty split

		reqConn := &requestConn{
			conn: conn,
			addr: addr,
		}

		switch opcode {
		case opRead:
			processReadRequest(reqConn, reqFields)
			// case opWrite:
			// 	c.processWriteRequest(reqFields)
		}
	}
}

func processReadRequest(conn *requestConn, req [][]byte) {
	if len(req) < 2 {
		writeError(conn, errNotDefined, "")
		return
	}

	filename := string(req[0])
	mode := string(req[1])
	filepath, _ := filepath.Abs(filepath.Join(rootDir, filename))

	log.Printf("Read request for %s with mode %s from %s\n", filename, mode, conn.addr.String())

	if !fileExists(filepath) {
		log.Printf("File %s not found\n", filepath)
		writeError(conn, errFileNotFound, "File not found")
		return
	}

	file, err := os.Open(filepath)
	if err != nil {
		log.Println(err)
		writeError(conn, errAccessViolation, "Failed to open file")
		return
	}

	newConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Println(err)
		file.Close()
		return
	}

	client := &connection{
		op:     opRead,
		client: &requestConn{conn: newConn, addr: conn.addr},
		file:   file,
		options: &options{
			blockSize:  512,
			timeout:    1 * time.Second,
			windowSize: 1,
		},
	}

	go client.start()
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
