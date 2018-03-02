package main

import (
	"flag"
	"fmt"
	"log"
)

var (
	flgRootDir        string
	flgDisableCreate  bool
	flgDisableWrite   bool
	flgAllowOverwrite bool
	flgServer         bool
)

func init() {
	flag.StringVar(&flgRootDir, "root", ".", "Server root")
	flag.BoolVar(&flgDisableCreate, "nocreate", false, "Disable creation of new files")
	flag.BoolVar(&flgDisableWrite, "nowrite", false, "Disable writing any files")
	flag.BoolVar(&flgAllowOverwrite, "ow", false, "Allow overwriting existing files")
	flag.BoolVar(&flgServer, "server", false, "Run a TFTP server")
}

func main() {
	flag.Parse()

	if flgAllowOverwrite && flgDisableWrite {
		log.Fatalln("-nowrite cannot be used with -ow")
	}

	if flgServer {
		startServer()
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
