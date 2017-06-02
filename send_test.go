package main

import (
	"bytes"
	"net"
	"testing"
	"time"
)

type testPacketConn struct {
	nextWrite []byte
	nextRead  []byte
	t         *testing.T
}

func (c *testPacketConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	copy(b, c.nextRead)
	if len(b) > len(c.nextRead) {
		return len(c.nextRead), nil, nil
	}
	return len(b), nil, nil
}

func (c *testPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	if !bytes.Equal(b, c.nextWrite) {
		c.t.Errorf("expected %#v, got %#v", c.nextWrite, b)
	}
	return len(b), nil
}

func (c *testPacketConn) Close() error                       { return nil }
func (c *testPacketConn) LocalAddr() net.Addr                { return nil }
func (c *testPacketConn) SetDeadline(t time.Time) error      { return nil }
func (c *testPacketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *testPacketConn) SetWriteDeadline(t time.Time) error { return nil }
func (c *testPacketConn) setNextWrite(req []byte)            { c.nextWrite = req }
func (c *testPacketConn) setNextRead(resp []byte)            { c.nextRead = resp }

func TestAck(t *testing.T) {
	c := &testPacketConn{t: t}
	c.setNextWrite([]byte{0, 4, 0, 10})
	sendAck(&requestConn{conn: c}, 10)

	c.setNextWrite([]byte{0, 4, 1, 0})
	sendAck(&requestConn{conn: c}, 256)
}

func TestOAck(t *testing.T) {
	c := &testPacketConn{t: t}
	next := []byte{0, 6}
	next = append(next, []byte("blksize")...)
	next = append(next, 0)
	next = append(next, []byte("1024")...)
	next = append(next, 0)
	c.setNextWrite(next)

	sendOAck(&requestConn{conn: c}, map[string]string{
		"blksize": "1024",
	})
}
