package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// bufConn is a simple in-memory net.Conn implementation that collects writes.
type bufConn struct{ bytes.Buffer }

func (c *bufConn) Read(b []byte) (int, error)         { return 0, io.EOF }
func (c *bufConn) Write(b []byte) (int, error)        { return c.Buffer.Write(b) }
func (c *bufConn) Close() error                       { return nil }
func (c *bufConn) LocalAddr() net.Addr                { return dummyAddr{} }
func (c *bufConn) RemoteAddr() net.Addr               { return dummyAddr{} }
func (c *bufConn) SetDeadline(t time.Time) error      { return nil }
func (c *bufConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *bufConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr struct{}

func (dummyAddr) Network() string { return "dummy" }
func (dummyAddr) String() string  { return "dummy" }

// extractCommand reads the command number from a packet written to bufConn.
func extractCommand(t *testing.T, buf *bufConn) uint32 {
	data := buf.Bytes()
	if len(data) < 22 { // size (2) + header (20)
		t.Fatalf("packet too small: %d bytes", len(data))
	}
	size := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+size {
		t.Fatalf("incomplete packet: got %d want %d", len(data)-2, size)
	}
	pkt := data[2 : 2+size]
	return binary.BigEndian.Uint32(pkt[16:20])
}

func TestSendPlayerInputCommandNumIncrements(t *testing.T) {
	// Preserve globals used by sendPlayerInput.
	oldCommandNum := commandNum
	oldPending := pendingCommand
	defer func() {
		commandNum = oldCommandNum
		pendingCommand = oldPending
	}()

	commandNum = 1
	pendingCommand = ""

	conn := &bufConn{}
	if err := sendPlayerInput(conn, 0, 0, false, false); err != nil {
		t.Fatalf("sendPlayerInput: %v", err)
	}
	if got, want := commandNum, uint32(2); got != want {
		t.Fatalf("commandNum=%d, want %d", got, want)
	}
	if cmd := extractCommand(t, conn); cmd != 1 {
		t.Fatalf("packet command=%d, want 1", cmd)
	}

	conn2 := &bufConn{}
	if err := sendPlayerInput(conn2, 0, 0, false, false); err != nil {
		t.Fatalf("sendPlayerInput: %v", err)
	}
	if got, want := commandNum, uint32(3); got != want {
		t.Fatalf("commandNum=%d, want %d", got, want)
	}
	if cmd := extractCommand(t, conn2); cmd != 2 {
		t.Fatalf("packet command=%d, want 2", cmd)
	}
}

func TestSendPlayerInputCommandNumIncrementsWithCommand(t *testing.T) {
	oldCommandNum := commandNum
	oldPending := pendingCommand
	defer func() {
		commandNum = oldCommandNum
		pendingCommand = oldPending
	}()

	commandNum = 10
	pendingCommand = "/test"

	conn := &bufConn{}
	if err := sendPlayerInput(conn, 0, 0, false, false); err != nil {
		t.Fatalf("sendPlayerInput: %v", err)
	}
	if got, want := commandNum, uint32(11); got != want {
		t.Fatalf("commandNum=%d, want %d", got, want)
	}
	if cmd := extractCommand(t, conn); cmd != 10 {
		t.Fatalf("packet command=%d, want 10", cmd)
	}
}
