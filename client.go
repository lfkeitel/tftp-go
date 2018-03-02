package main

import (
	"io"
	"log"
	"os"
	"time"
)

type stater interface {
	Stat() (os.FileInfo, error)
}

type lengther interface {
	Len() int
}

type client struct {
	op           opCode
	currentBlock []byte
	blockCounter uint16
	conn         *requestConn
	data         io.ReadWriter
	options      *tftpOptions
}

type response struct {
	op        opCode
	blockID   uint16
	errorCode uint16
	errorMsg  string
	data      []byte
}

func (c *client) start() {
	c.currentBlock = make([]byte, c.options.blockSize)
	c.blockCounter = 0

	start := time.Now()
	switch c.op {
	case opRead:
		c.doRead()
	case opWrite:
		c.doWrite()
	}
	log.Printf("Transfer completed in %s", time.Since(start).String())
}

func (c *client) doRead() {
	var size int64

	if file, ok := c.data.(stater); ok {
		stat, err := file.Stat()
		if err != nil {
			c.close()
			return
		}
		size = stat.Size()
	} else if buf, ok := c.data.(lengther); ok {
		size = int64(buf.Len())
	}

	log.Printf("Starting transfer of %d bytes\n", size)
	prepareNextBlock := true
	retransmits := 0

	for {
		if prepareNextBlock {
			if err := c.prepareNextBlock(); err != nil {
				log.Println(err)
				c.conn.sendError(errAccessViolation, "")
				c.close()
				break
			}
		}

		c.sendBlock()
		resp := c.conn.readNextMessage(c.op, c.options)
		if resp == nil {
			c.close()
			break
		}

		if resp.op == opAck { // Client acknowledged data block
			prepareNextBlock = resp.blockID == c.blockCounter
			retransmits = 0
		} else if resp.op == opError { // Client sent error
			log.Printf("Error %d: %s", resp.errorCode, resp.errorMsg)
			c.close()
			break
		} else if resp.op == opRetransmit { // Read timed out
			if retransmits >= maxRetransmits {
				log.Println("Max retransmits exceeded, terminating tranfer")
				c.close()
				break
			}

			log.Println("Retransmitting last block")
			prepareNextBlock = false
			retransmits++
			continue
		} else {
			c.conn.sendError(errIllegalOperation, "Invalid operation for read request")
			c.close()
			break
		}

		if len(c.currentBlock) < c.options.blockSize {
			c.close()
			break
		}
	}
}

func (c *client) doWrite() {
	log.Println("Starting write")
	if !c.options.oackSent {
		c.conn.sendAck(0)
	}

	for {
		resp := c.conn.readNextMessage(c.op, c.options)
		if resp == nil {
			c.close()
			break
		}

		if resp.op == opData {
			_, err := c.data.Write(resp.data)
			if err != nil {
				log.Println(err)
				c.conn.sendError(errAccessViolation, "Failed to write block")
				c.close()
				break
			}

			c.blockCounter = resp.blockID
			c.conn.sendAck(c.blockCounter)

			if len(resp.data) < c.options.blockSize { // Transfer complete
				c.close()
				break
			}
		} else if resp.op == opError { // Client sent error
			log.Printf("Error %d: %s", resp.errorCode, resp.errorMsg)
			c.close()
			break
		} else if resp.op == opRetransmit {
			c.conn.sendAck(c.blockCounter)
		} else {
			c.conn.sendError(errIllegalOperation, "Invalid operation for read request")
			c.close()
			break
		}
	}
}

func (c *client) close() {
	c.conn.conn.Close()
	if closer, ok := c.data.(io.Closer); c.data != nil && ok {
		closer.Close()
	}
}

func (c *client) prepareNextBlock() error {
	c.blockCounter++

	n, err := c.data.Read(c.currentBlock)
	if err != nil && err != io.EOF {
		return err
	}

	// Shrink the slice for last transmit
	c.currentBlock = c.currentBlock[:n]
	return nil
}

func (c *client) sendBlock() {
	c.conn.sendData(c.blockCounter, c.currentBlock)
}
