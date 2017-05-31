package main

import (
	"log"
	"net"
	"os"
	"time"
)

type connection struct {
	op           int
	currentBlock []byte
	blockCounter uint16
	client       *requestConn
	file         *os.File
	options      *options
}

type response struct {
	op        int
	blockID   uint16
	errorCode uint16
	errorMsg  string
}

func (c *connection) start() {
	c.currentBlock = make([]byte, c.options.blockSize)
	c.blockCounter = 0

	start := time.Now()
	switch c.op {
	case opRead:
		c.doRead()
		log.Printf("Transfer completed in %s", time.Since(start).String())
	}
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
		resp := c.getResp()
		if resp == nil {
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
		}

		if len(c.currentBlock) < c.options.blockSize {
			c.close()
			break
		}
	}
}

func (c *connection) getResp() *response {
	buffer := make([]byte, 512)
	c.client.conn.SetReadDeadline(time.Now().Add(c.options.timeout))

	n, addr, err := c.client.conn.ReadFrom(buffer)
	if err != nil {
		netErr := err.(net.Error)
		if netErr.Timeout() {
			return &response{op: opRetransmit}
		}
		log.Println(err)
		return nil
	}

	c.client.addr = addr

	if n < 4 {
		writeError(c.client, errNotDefined, "Malformatted message")
		c.close()
		return nil
	}

	recv := buffer[:n]

	opcode := decodeUInt16(recv[:2])

	switch opcode {
	case opAck:
		return &response{
			op:      opAck,
			blockID: decodeUInt16(recv[2:4]),
		}
	case opError:
		errorMsg := ""
		if len(recv) > 4 {
			errorMsg = string(recv[4 : len(recv)-1])
		}

		return &response{
			op:        opError,
			errorCode: decodeUInt16(recv[2:4]),
			errorMsg:  errorMsg,
		}
	default:
		writeError(c.client, errIllegalOperation, "")
		c.close()
		return nil
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
	if err != nil {
		return err
	}

	// Shrink the slice for last transmit
	c.currentBlock = c.currentBlock[:n]
	return nil
}

func (c *connection) sendBlock() {
	sendData(c.client, c.blockCounter, c.currentBlock)
}
