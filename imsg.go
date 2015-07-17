// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
)

const (
	ImsgfHasFD = 1

	IbufReadLen = 65535
	MaxImsgLen  = 16384
	cmsgBufLen  = 4096
)

var (
	imsgHeaderLen = (&ImsgHeader{}).WireLen()

	ErrNoSocket         = fmt.Errorf("no socket")
	ErrImsgBufferClosed = fmt.Errorf("Buffer channel closed")
)

type WireSerializer interface {
	InitFromWireBytes([]byte) error
	WireBytes([]byte) error
	WireLen() int
}

type rawBuf struct {
	data []byte
	fds  []*os.File
}

type Imsg struct {
	Header ImsgHeader
	Data   []byte
	FD     *os.File
}

// ImsgHeader describes the current message.
type ImsgHeader struct {
	Type   uint32
	Len    uint16
	Flags  uint16
	PeerID uint32
	Pid    uint32
}

// header length is used often, calc it once
func (ihdr *ImsgHeader) WireLen() int {
	return 16
}

func (ihdr *ImsgHeader) WireBytes(buf []byte) error {
	var err error
	var bb bytes.Buffer

	if err = binary.Write(&bb, binary.LittleEndian, ihdr.Type); err != nil {
		return fmt.Errorf("Write(Type): %s", err)
	}
	if err = binary.Write(&bb, binary.LittleEndian, ihdr.Len); err != nil {
		return fmt.Errorf("Write(Len): %s", err)
	}
	if err = binary.Write(&bb, binary.LittleEndian, ihdr.Flags); err != nil {
		return fmt.Errorf("Write(Flags): %s", err)
	}
	if err = binary.Write(&bb, binary.LittleEndian, ihdr.PeerID); err != nil {
		return fmt.Errorf("Write(PeerID): %s", err)
	}
	if err = binary.Write(&bb, binary.LittleEndian, ihdr.Pid); err != nil {
		return fmt.Errorf("Write(Pid): %s", err)
	}
	bbuf := bb.Bytes()
	if len(buf) < len(bbuf) {
		return fmt.Errorf("ImsgHeader 1 bad len %d/%d/16", len(buf), len(bbuf))
	}
	copy(buf, bbuf)
	//log.Printf("hdr(%#v) bytes : %#v", ihdr, bbuf)
	return nil
}

func (ihdr *ImsgHeader) InitFromWireBytes(buf []byte) error {
	if len(buf) != 16 {
		return fmt.Errorf("ImsgHeader 2 bad len %d/16", len(buf))
	}
	ihdr.Type = binary.LittleEndian.Uint32(buf[0:4])
	ihdr.Len = binary.LittleEndian.Uint16(buf[4:6])
	ihdr.Flags = binary.LittleEndian.Uint16(buf[6:8])
	ihdr.PeerID = binary.LittleEndian.Uint32(buf[8:12])
	ihdr.Pid = binary.LittleEndian.Uint32(buf[12:16])
	return nil
}

type ImsgBuffer struct {
	conn   *net.UnixConn
	mu     sync.Mutex
	wQueue []Imsg
	msgs   chan *Imsg
	// Linux kernel defines pid_t as a 32-bit signed int in
	// include/uapi/asm-generic/posix_types.h
	pid int32
}

func NewImsgBuffer(path string) (*ImsgBuffer, error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, fmt.Errorf("ResolveAddrUnix(%s): %s", path, err)
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, ErrNoSocket
	}
	ibuf := &ImsgBuffer{
		conn: conn,
		pid:  int32(os.Getpid()),
		msgs: make(chan *Imsg),
	}

	between := make(chan rawBuf)
	go ibuf.readSocket(between)
	go ibuf.reader(between)

	return ibuf, nil
}

func (ibuf *ImsgBuffer) Compose(kind, peerID, pid uint32, data WireSerializer, fd *os.File) error {
	var err error
	size := imsgHeaderLen + data.WireLen()
	header := ImsgHeader{
		Type:   kind,
		Len:    uint16(size),
		PeerID: peerID,
		Pid:    pid,
	}
	if header.Pid == 0 {
		header.Pid = uint32(ibuf.pid)
	}
	if fd != nil {
		header.Flags |= ImsgfHasFD
	}
	buf := make([]byte, size)

	if err = header.WireBytes(buf[0:imsgHeaderLen]); err != nil {
		return fmt.Errorf("header.WireBytes: %s", err)
	}
	if err = data.WireBytes(buf[imsgHeaderLen:]); err != nil {
		return fmt.Errorf("header.WireBytes: %s", err)
	}

	ibuf.mu.Lock()
	ibuf.wQueue = append(ibuf.wQueue, Imsg{Data: buf, FD: fd})
	ibuf.mu.Unlock()

	return nil
}

// Flush writes all pending buffers to the socket
func (ibuf *ImsgBuffer) Flush() {
	ibuf.mu.Lock()
	defer ibuf.mu.Unlock()

	for _, buf := range ibuf.wQueue {
		var cmsgBuf []byte
		if buf.FD != nil {
			cmsgBuf = syscall.UnixRights(int(buf.FD.Fd()))
			defer buf.FD.Close()
		}
		n, _, err := ibuf.conn.WriteMsgUnix(buf.Data, cmsgBuf, nil)
		if err != nil {
			log.Printf("ibuf.conn.Write: %s", err)
		} else if n != len(buf.Data) {
			log.Printf("ibuf.conn.Write short: %d/%d", n, len(buf.Data))
		}
	}
	ibuf.wQueue = nil
}

func (ibuf *ImsgBuffer) Get() (*Imsg, error) {
	result := <-ibuf.msgs
	if result == nil {
		return nil, ErrImsgBufferClosed
	}

	return result, nil
}

func (ibuf *ImsgBuffer) Close() {
	ibuf.conn.Close()
}

func zero(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

// fd -> chan []byte
// []byte chan -> imsg chan

func (ibuf *ImsgBuffer) readSocket(out chan<- rawBuf) {
	cmsgBuf := make([]byte, cmsgBufLen)
	imsgBuf := make([]byte, IbufReadLen)

	for {
		// TODO: not necessary
		zero(cmsgBuf)
		zero(imsgBuf)

		n, cn, _, _, err := ibuf.conn.ReadMsgUnix(imsgBuf, cmsgBuf)
		if err != nil {
			if err == io.EOF {
				close(ibuf.msgs)
			}
			// if we have a read error, its probably
			// because our connection closed, just quietly
			// exit.
			return
		}

		// copy into a new buffer, so we can safely pass it to
		// another goroutine
		buf := make([]byte, n)
		copy(buf, imsgBuf)

		var files []*os.File

		// TODO: not sure how this works.  a Read might return
		// multiple messages coalessed into one.  How does
		// that correspond to socket control messages?
		if cn > 0 {
			cmsgs, err := syscall.ParseSocketControlMessage(cmsgBuf)
			if err != nil {
				log.Printf("ParseSocketControlMessage: %s", err)
				return
			}
			log.Printf("n cmsgs: %d", len(cmsgs))
			for _, cmsg := range cmsgs {
				fds, err := syscall.ParseUnixRights(&cmsg)
				if err != nil {
					log.Printf("ParseUnixRights: %s", err)
				}
				for _, fd := range fds {
					files = append(files, os.NewFile(uintptr(fd), "<from tmux>"))
				}
			}
		}

		// TODO: remove debugging
		if len(files) > 0 {
			log.Printf("FDs: %#v", files)
		}

		out <- rawBuf{buf, files}
	}
}

func (ibuf *ImsgBuffer) reader(in <-chan rawBuf) {
	var inbetween bytes.Buffer
	hBytes := make([]byte, imsgHeaderLen)

	for {
		rawBuf := <-in
		inbetween.Write(rawBuf.data)

		for inbetween.Len() > 0 {
			zero(hBytes)

			n, err := inbetween.Read(hBytes)
			if err != nil {
				log.Printf("inbetween.Read: %s", err)
				return
			} else if n != imsgHeaderLen {
				// short read - should never happen
				log.Printf("inbetween.Read short: %d/%d",
					n, imsgHeaderLen)
				return
			}

			var header ImsgHeader
			if err = header.InitFromWireBytes(hBytes); err != nil {
				log.Printf("InitFromWireBytes: %s", err)
				return
			}

			var payload []byte
			payloadLen := int(header.Len) - imsgHeaderLen
			if payloadLen > 0 {
				payload = make([]byte, payloadLen)
				n, err := inbetween.Read(payload)
				if err != nil {
					log.Printf("inbetween.Read 2: %s", err)
					return
				} else if n != payloadLen {
					log.Printf("inbetween.Read 2 short: %d/%d", n, payloadLen)
					return
				}
			}

			var fd *os.File
			if len(rawBuf.fds) > 0 {
				fd = rawBuf.fds[0]
			}

			// TODO: this FD handling isn't quite right
			imsg := &Imsg{header, payload, fd}
			ibuf.msgs <- imsg
		}
	}
}

type String struct {
	S string
}

func (s *String) WireLen() int {
	return len(s.S)
}

func (s *String) WireBytes(buf []byte) error {
	sBytes := []byte(s.S)
	if len(buf) < len(sBytes)+1 {
		return fmt.Errorf("String bad len %d/%d", len(buf), len(sBytes)+1)
	}

	copy(buf, sBytes)
	// make sure its null terminated
	buf[len(sBytes)] = 0

	return nil
}

func (s *String) InitFromWireBytes(buf []byte) error {
	if len(buf) > 0 && buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}
	s.S = string(buf)

	return nil
}

type Int32 struct {
	int32
}

func (i *Int32) WireLen() int {
	return 4
}

func (i *Int32) WireBytes(buf []byte) error {
	var err error
	if len(buf) < 4 {
		return fmt.Errorf("Int32 1 bad len %d/4", len(buf))
	}
	var bb bytes.Buffer

	if err = binary.Write(&bb, binary.LittleEndian, int32(i.int32)); err != nil {
		return fmt.Errorf("i(%d) write: %s", int32(i.int32), err)
	}

	bbuf := bb.Bytes()
	copy(buf, bbuf)

	return nil
}

func (i *Int32) InitFromWireBytes(buf []byte) error {
	if len(buf) != 4 {
		return fmt.Errorf("Int32 bad len %d/16", len(buf))
	}
	i.int32 = int32(binary.LittleEndian.Uint32(buf))

	return nil
}

type Nil struct{}

func (n Nil) WireLen() int {
	return 0
}

func (n Nil) WireBytes(buf []byte) error {
	return nil
}

func (n Nil) InitFromWireBytes(buf []byte) error {
	return nil
}
