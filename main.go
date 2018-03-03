package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var (
	flgRootDir        string
	flgDisableCreate  bool
	flgDisableWrite   bool
	flgAllowOverwrite bool
	flgServer         bool
	flgDebug          bool
	flgRFC1350        bool
	flgStrict         bool
)

func init() {
	flag.StringVar(&flgRootDir, "root", ".", "Server root")
	flag.BoolVar(&flgDisableCreate, "nocreate", false, "Disable creation of new files")
	flag.BoolVar(&flgDisableWrite, "nowrite", false, "Disable writing any files")
	flag.BoolVar(&flgAllowOverwrite, "ow", false, "Allow overwriting existing files")
	flag.BoolVar(&flgServer, "server", false, "Run a TFTP server")
	flag.BoolVar(&flgDebug, "debug", false, "Enable debug output")
	flag.BoolVar(&flgRFC1350, "rfc1350", false, "Disable TFTP options")
	flag.BoolVar(&flgStrict, "strict", false, "Reject clients wanting to use netascii or mail modes")
}

func main() {
	flag.Parse()

	if flgAllowOverwrite && flgDisableWrite {
		log.Fatalln("-nowrite cannot be used with -ow")
	}

	if flgServer && flag.NArg() > 0 {
		log.Fatalln("-server cannot be used with a command")
	}

	if flgServer {
		startServer()
	} else {
		runCommand(flag.Args())
	}
}

func startServer() {
	serverOptions := []serverOption{withRootDir(flgRootDir)}
	if flgDisableCreate {
		serverOptions = append(serverOptions, withDisableCreate)
	}
	if flgDisableWrite {
		serverOptions = append(serverOptions, withDisableWrite)
	}
	if flgAllowOverwrite {
		serverOptions = append(serverOptions, withAllowOverwrite)
	}

	s := newServer(serverOptions...)
	s.listenAndServe(fmt.Sprintf(":%d", tftpPort))
}

func runCommand(args []string) {
	if len(args) != 3 {
		printClientUsage()
	}

	remote := strings.SplitN(args[1], ":", 2)
	if len(remote) != 2 {
		printClientUsage()
	}

	newConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Fatalln(err)
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remote[0], tftpPort))
	if err != nil {
		log.Fatalln(err)
	}
	directConn := &requestConn{conn: newConn, addr: addr}

	switch args[0] {
	case "put":
		putFile(directConn, args[2], remote[1])
	case "get":
		getFile(directConn, remote[1], args[2])
	default:
		newConn.Close()
		printClientUsage()
	}
}

func putFile(conn *requestConn, source, dest string) {
	file, err := os.Open(source)
	if err != nil {
		log.Fatalln(err)
	}

	opts := defaultOptions.copy()

	debug("Sending write request")
	if flgRFC1350 {
		conn.sendWriteRequest(dest, modeOctet, nil)
	} else {
		tsize := filesize(file)
		opts.blockSize = 1428
		if tsize > -1 {
			opts.tsize = tsize
		}

		conn.sendWriteRequest(dest, modeOctet, opts.toMap())
	}

	// Wait for server to ACK write request and/or options
	retransmits := 0
	for {
		resp := conn.readNextMessage(opRead, defaultOptions)
		if resp == nil || resp.op == opError {
			conn.Close()
			file.Close()
			return
		}

		if resp.op == opRetransmit {
			if retransmits >= maxRetransmits {
				conn.Close()
				file.Close()
				return
			}

			debug("Retransmitting WRITE request")
			conn.sendWriteRequest(dest, modeOctet, opts.toMap())
			retransmits++
			continue
		} else if resp.op == opOAck {
			debug("Received OACK")
			opts = resp.options
			break
		} else if resp.op == opAck {
			debug("Received ACK")
			break
		}
	}

	remote := &client{
		op:         opRead, // From the client we're reading a file to the server
		conn:       conn,
		data:       file,
		options:    opts,
		remotePath: dest,
	}

	remote.run()
}

func getFile(conn *requestConn, source, dest string) {
	file, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}

	opts := defaultOptions.copy()

	debug("Sending read request")
	if flgRFC1350 {
		conn.sendReadRequest(source, modeOctet, nil)
	} else {
		opts.blockSize = 1428
		conn.sendReadRequest(source, modeOctet, opts.toMap())
		// The client will respond to OACKS and retransmit if needed.
	}

	remote := &client{
		op:               opWrite, // From the client we're writing to a file
		conn:             conn,
		data:             file,
		options:          defaultOptions.copy(),
		requestedOptions: opts,
		remotePath:       source,
	}

	remote.run()
}

func printClientUsage() {
	log.Fatalln("Usage: tftp [put|get] REMOTE:PATH LOCAL")
}
