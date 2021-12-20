package main

import (
	"bytes"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type serverOption func(*server)

type server struct {
	conn           net.PacketConn
	rootDir        string
	disableCreate  bool
	disableWrite   bool
	allowOverwrite bool
}

func newServer(options ...serverOption) *server {
	s := &server{}
	for _, option := range options {
		option(s)
	}
	return s
}

func withRootDir(dir string) serverOption {
	return func(s *server) {
		s.rootDir = dir
	}
}

func withDisableCreate(s *server) {
	s.disableCreate = true
}

func withDisableWrite(s *server) {
	s.disableWrite = true
}

func withAllowOverwrite(s *server) {
	s.allowOverwrite = true
}

func (s *server) listenAndServe(address string) {
	stat, err := os.Stat(s.rootDir)
	if err != nil {
		log.Fatalln(err)
	}
	if !stat.IsDir() {
		log.Fatalln("Server root is not a directory")
	}

	fullpath, _ := filepath.Abs(s.rootDir)
	log.Printf("Start TFTP server serving %s", fullpath)

	s.conn, err = net.ListenPacket("udp", address)
	if err != nil {
		log.Println(err)
		return
	}

	buffer := make([]byte, defaultOptions.blockSize)
	for {
		n, addr, err := s.conn.ReadFrom(buffer)
		if err != nil {
			log.Println(err)
			return
		}

		req := buffer[:n]

		opcode := opCode(decodeUInt16(req[:2]))
		reqFields := bytes.Split(req[2:], []byte{0})
		reqFields = reqFields[:len(reqFields)-1] // Remove empty split

		conn := &requestConn{conn: s.conn, addr: addr}
		switch opcode {
		case opRead, opWrite:
			s.processRequest(conn, opcode, reqFields)
		}
	}
}

func (s *server) processRequest(conn *requestConn, op opCode, req [][]byte) {
	if len(req) < 2 {
		conn.sendError(errNotDefined, "")
		return
	}

	if op == opWrite && s.disableWrite {
		conn.sendError(errAccessViolation, "Writes disabled")
		return
	}

	filename := string(req[0])
	mode := string(req[1])
	filename = strings.Replace(filename, "..", "", -1) // Prevent escaping from root directory
	filepath, _ := filepath.Abs(filepath.Join(s.rootDir, filename))

	log.Printf("%s request for %s with mode %s from %s", op, filename, mode, conn.addr.String())
	if mode != modeOctet {
		if flgStrict {
			conn.sendError(errAccessViolation, "Unsupported mode")
			return
		}

		log.Printf("WARNING: Client is using %s mode but the server will be using octet.", mode)
	}

	exists := fileExists(filepath)

	if op == opRead && !exists {
		conn.sendError(errFileNotFound, "File not found")
		log.Printf("File %s not found.", filepath)
		return
	}

	if op == opWrite && !exists && s.disableCreate {
		conn.sendError(errAccessViolation, "Cannot create new file")
		return
	}

	var file *os.File
	var err error

	if op == opRead {
		file, err = os.Open(filepath)
	} else {
		if exists && !s.allowOverwrite {
			log.Println("Attempted overwrite of existing file")
			conn.sendError(errFileExists, "Attempted overwrite of existing file")
			return
		}
		file, err = os.Create(filepath)
	}

	if err != nil {
		log.Println(err)
		conn.sendError(errAccessViolation, "Failed to open file")
		return
	}

	options, ackedOptions := parseOptions(req[2:])

	// tsize is -1 if the option wasn't given
	if options.tsize > -1 {
		if op == opWrite {
			ackedOptions[optionTransferSize] = strconv.FormatInt(options.tsize, 10)
		} else {
			stat, err := file.Stat()
			if err != nil {
				conn.sendError(errAccessViolation, "Failed to open file")
				file.Close()
				return
			}
			ackedOptions[optionTransferSize] = strconv.FormatInt(stat.Size(), 10)
		}
	}

	newConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Println(err)
		file.Close()
		return
	}

	directConn := &requestConn{conn: newConn, addr: conn.addr}

	// We need to send option ack
	if !flgRFC1350 && len(ackedOptions) > 0 {
		debug("ACKing requested options: %#v", ackedOptions)
		directConn.sendOAck(ackedOptions)
		options.oackSent = true // Tells the connection.recvFile() not to send an ack

		if op == opRead { // Get client's ACK for our OACK
			retransmits := 0
			for {
				resp := directConn.readNextMessage(opRead, defaultOptions)
				if resp == nil || resp.op == opError {
					newConn.Close()
					file.Close()
					return
				}

				if resp.op == opRetransmit {
					if retransmits >= maxRetransmits {
						newConn.Close()
						file.Close()
						return
					}

					debug("Retransmitting OACK")
					directConn.sendOAck(ackedOptions)
					retransmits++
					continue
				} else if resp.op == opAck {
					debug("Received ACK")
					break
				} else {
					debug("Received ILLEGAL")
					directConn.sendError(errIllegalOperation, "Invalid operation for read request")
					newConn.Close()
					file.Close()
					break
				}
			}
		}
	} else if flgRFC1350 {
		debug("TFTP options are disabled, not acknowledging")
	}

	client2 := &client{
		op:      op,
		conn:    directConn,
		data:    file,
		options: options,
	}

	go client2.run()
}
