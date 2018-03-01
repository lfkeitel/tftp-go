package main

import (
	"io"
	"log"
	"os"
	"time"
)

type connection struct {
	op           uint16
	currentBlock []byte
	blockCounter uint16
	client       *requestConn
	file         *os.File
	options      *tftpOptions
}

type response struct {
	op        uint16
	blockID   uint16
	errorCode uint16
	errorMsg  string
	data      []byte
}

func (c *connection) start() {
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

func (c *connection) doRead() {
	stat, err := c.file.Stat()
	if err != nil {
		c.close()
		return
	}

	log.Printf("Starting transfer of %d bytes\n", stat.Size())
	prepareNextBlock := true
	retransmits := 0

	for {
		if prepareNextBlock {
			if err := c.prepareNextBlock(); err != nil {
				log.Println(err)
				writeError(c.client, errAccessViolation, "")
				c.close()
				break
			}
		}

		c.sendBlock()
		resp := nextMessage(c.client, c.op, c.options)
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
			writeError(c.client, errIllegalOperation, "Invalid operation for read request")
			c.close()
			break
		}

		if len(c.currentBlock) < c.options.blockSize {
			c.close()
			break
		}
	}
}

func (c *connection) doWrite() {
	log.Println("Starting write")
	if !c.options.oackSent {
		sendAck(c.client, 0)
	}

	for {
		resp := nextMessage(c.client, c.op, c.options)
		if resp == nil {
			c.close()
			break
		}

		if resp.op == opData {
			_, err := c.file.Write(resp.data)
			if err != nil {
				log.Println(err)
				writeError(c.client, errAccessViolation, "Failed to write block")
				c.close()
				break
			}

			c.blockCounter = resp.blockID
			sendAck(c.client, c.blockCounter)

			if len(resp.data) < c.options.blockSize { // Transfer complete
				c.close()
				break
			}
		} else if resp.op == opError { // Client sent error
			log.Printf("Error %d: %s", resp.errorCode, resp.errorMsg)
			c.close()
			break
		} else if resp.op == opRetransmit {
			sendAck(c.client, c.blockCounter)
		} else {
			writeError(c.client, errIllegalOperation, "Invalid operation for read request")
			c.close()
			break
		}
	}
}

func (c *connection) close() {
	c.client.conn.Close()
	if c.file != nil {
		c.file.Close()
	}
}

func (c *connection) prepareNextBlock() error {
	c.blockCounter++

	n, err := c.file.Read(c.currentBlock)
	if err != nil && err != io.EOF {
		return err
	}

	// Shrink the slice for last transmit
	c.currentBlock = c.currentBlock[:n]
	return nil
}

func (c *connection) sendBlock() {
	sendData(c.client, c.blockCounter, c.currentBlock)
}
