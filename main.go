package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	rootDir        string
	allowCreate    bool
	disableWrite   bool
	allowOverwrite bool
)

func init() {
	flag.StringVar(&rootDir, "root", ".", "Server root")
	flag.BoolVar(&allowCreate, "create", false, "Allow creation of new files")
	flag.BoolVar(&disableWrite, "nowrite", false, "Disable writing any files")
	flag.BoolVar(&allowOverwrite, "ow", false, "Allow overwriting existing files")
}

func main() {
	flag.Parse()

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

	if op == opWrite && disableWrite {
		writeError(conn, errAccessViolation, "Writes disabled")
		return
	}

	filename := string(req[0])
	mode := string(req[1])
	filename = strings.Replace(filename, "..", "", -1) // Prevent escaping from root directory
	filepath, _ := filepath.Abs(filepath.Join(rootDir, filename))

	log.Printf("%s request for %s with mode %s from %s\n", optionToString(op), filename, mode, conn.addr.String())

	exists := fileExists(filepath)

	if op == opRead && !exists {
		writeError(conn, errFileNotFound, "File not found")
		return
	}

	if op == opWrite && !exists && !allowCreate {
		writeError(conn, errFileNotFound, "File not found")
		return
	}

	var file *os.File
	var err error

	if op == opRead {
		file, err = os.Open(filepath)
	} else {
		if fileExists(filepath) && !allowOverwrite {
			log.Println("Attempted overwrite of existing file")
			writeError(conn, errAccessViolation, "Attempted overwrite of existing file")
			return
		}
		file, err = os.Create(filepath)
	}

	if err != nil {
		log.Println(err)
		writeError(conn, errAccessViolation, "Failed to open file")
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
				writeError(conn, errAccessViolation, "Failed to open file")
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

	rconn := &requestConn{conn: newConn, addr: conn.addr}

	// We need to send option ack
	if len(ackedOptions) > 0 {
		sendOAck(rconn, ackedOptions)
		options.oackSent = true // Tells the connection.doWrite() not to send an ack

		if op == opRead { // Get client's ACK for our OACK
			retransmits := 0
			for {
				resp := nextMessage(rconn, opRead, defaultOptions)
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

					sendOAck(rconn, ackedOptions)
					retransmits++
					continue
				} else if resp.op == opAck {
					break
				}
			}
		}
	}

	client := &connection{
		op:      op,
		client:  rconn,
		file:    file,
		options: options,
	}

	go client.start()
}

func parseOptions(options [][]byte) (*tftpOptions, map[string]string) {
	// Default options to be modified by requested options
	base := &tftpOptions{
		blockSize:  512,
		timeout:    5 * time.Second,
		windowSize: 1,
		tsize:      -1,
	}

	optionLen := len(options)
	if optionLen < 2 {
		return base, nil
	}

	ackedOptions := make(map[string]string)

	if optionLen&1 == 1 { // Check option slice is even length
		options = options[:len(options)-1]
	}

	for i := 0; i < len(options); i += 2 {
		option := strings.ToLower(string(options[i])) // options names are case insensitive
		value := string(options[i+1])

		switch option {
		case optionBlockSize:
			val, err := strconv.Atoi(value)
			if err != nil {
				continue
			}

			if val < 8 || val > 65464 { // Request value out of range
				// Respond with default
				ackedOptions[optionBlockSize] = strconv.Itoa(base.blockSize)
				continue
			}
			base.blockSize = val
			ackedOptions[optionBlockSize] = value
		case optionTimeout:
			val, err := strconv.Atoi(value)
			if err != nil {
				continue
			}

			if val < 1 || val > 255 { // Request value out of range
				// Response with default
				ackedOptions[optionTimeout] = strconv.FormatInt(base.timeout.Nanoseconds()/int64(time.Second), 10)
				continue
			}
			base.timeout = time.Duration(val) * time.Second
			ackedOptions[optionTimeout] = value
		case optionTransferSize:
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				continue
			}
			// The calling function is responsible for fulfilling tsize
			base.tsize = val
		}
	}

	return base, ackedOptions
}
