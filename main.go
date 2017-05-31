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
	rootDir = os.Args[1]

	stat, err := os.Stat(rootDir)
	if err != nil {
		log.Fatalln(err)
	}
	if !stat.IsDir() {
		log.Fatalln("Server root is not a directory")
	}

	fullpath, _ := filepath.Abs(rootDir)
	log.Printf("Start TFTP server serving %s", fullpath)

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
		case opRead, opWrite:
			processRequest(reqConn, opcode, reqFields)
		}
	}
}

func processRequest(conn *requestConn, op uint16, req [][]byte) {
	if len(req) < 2 {
		writeError(conn, errNotDefined, "")
		return
	}

	filename := string(req[0])
	mode := string(req[1])
	filepath, _ := filepath.Abs(filepath.Join(rootDir, filename))

	log.Printf("%s request for %s with mode %s from %s\n", optionToString(op), filename, mode, conn.addr.String())

	if op == opRead && !fileExists(filepath) {
		log.Printf("File %s not found\n", filepath)
		writeError(conn, errFileNotFound, "File not found")
		return
	}

	var file *os.File
	var err error

	if op == opRead {
		file, err = os.Open(filepath)
	} else {
		file, err = os.Create(filepath)
	}

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
		op:     op,
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
