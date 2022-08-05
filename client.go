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
	op               opCode
	currentBlock     []byte
	blockCounter     uint16
	conn             *requestConn
	data             io.ReadWriter
	options          *tftpOptions
	requestedOptions *tftpOptions
	remotePath       string
}

type response struct {
	op        opCode
	blockID   uint16
	errorCode uint16
	errorMsg  string
	data      []byte
	options   *tftpOptions
}

func (c *client) run() {
	c.currentBlock = make([]byte, c.options.blockSize)
	c.blockCounter = 0

	start := time.Now()
	switch c.op {
	case opRead:
		c.sendFile()
	case opWrite:
		c.recvFile()
	}
	log.Printf("Transfer completed in %s", time.Since(start).String())
}

func (c *client) sendFile() {
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
			debug("Received ACK")
			prepareNextBlock = resp.blockID == c.blockCounter
			retransmits = 0

			if len(c.currentBlock) < c.options.blockSize {
				c.close()
				break
			}
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

			debug("Retransmitting last block")
			prepareNextBlock = false
			retransmits++
			continue
		} else {
			debug("Received ILLEGAL")
			c.conn.sendError(errIllegalOperation, "Invalid operation for read request")
			c.close()
			break
		}
	}
}

func (c *client) recvFile() {
	c.blockCounter = 0

	log.Println("Starting file receive")
	if !c.options.oackSent && c.requestedOptions == nil {
		c.conn.sendAck(c.blockCounter)
	}
	retransmits := 0

	for {
		resp := c.conn.readNextMessage(c.op, c.options)
		if resp == nil {
			c.close()
			break
		}

		if resp.op == opData {
			debug("Received DATA")
			retransmits = 0

			if resp.blockID != c.blockCounter+1 {
				log.Printf("Warning: Block # expected %d, block # received %d", c.blockCounter+1, resp.blockID)
				c.conn.sendAck(c.blockCounter)
				continue
			}

			c.requestedOptions = nil
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
			if retransmits >= maxRetransmits {
				log.Println("Max retransmits exceeded, terminating tranfer")
				c.close()
				break
			}

			if c.requestedOptions != nil {
				debug("Retransmitting read request")
				c.conn.sendReadRequest(c.remotePath, modeOctet, c.requestedOptions.toMap())
			} else {
				debug("Retransmitting ACK")
				c.conn.sendAck(c.blockCounter)
			}
			retransmits++
		} else if resp.op == opOAck {
			debug("Received OACK")
			if c.requestedOptions != nil {
				c.options = resp.options
			}
			debug("ACKing OACK")
			c.conn.sendAck(0)
		} else {
			debug("Received ILLEGAL")
			c.conn.sendError(errIllegalOperation, "Invalid operation for write request")
			c.close()
			break
		}
	}
}

func (c *client) close() {
	c.conn.Close()
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
	debug("Sending DATA block # %d", c.blockCounter)
	c.conn.sendData(c.blockCounter, c.currentBlock)
}
