// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

const (
	ImsgfHasFD = 1

	IbufReadLen = 65535
	MaxImsgLen  = 16384
)

var (
	imsgHeaderLen = (&ImsgHeader{}).WireLen()
)

type WireSerializer interface {
	InitFromWireBytes([]byte) error
	WireBytes([]byte) error
	WireLen() int
}

type Imsg struct {
	Header ImsgHeader
	Data   []byte
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
	if len(buf) < 16 {
		return fmt.Errorf("ImsgHeader 1 bad len %d/16", len(buf))
	}
	bb := bytes.NewBuffer(buf)

	if err = binary.Write(bb, binary.LittleEndian, ihdr.Type); err != nil {
		return fmt.Errorf("Write(Type): %s", err)
	}
	if err = binary.Write(bb, binary.LittleEndian, ihdr.Len); err != nil {
		return fmt.Errorf("Write(Len): %s", err)
	}
	if err = binary.Write(bb, binary.LittleEndian, ihdr.Flags); err != nil {
		return fmt.Errorf("Write(Flags): %s", err)
	}
	if err = binary.Write(bb, binary.LittleEndian, ihdr.PeerID); err != nil {
		return fmt.Errorf("Write(PeerID): %s", err)
	}
	if err = binary.Write(bb, binary.LittleEndian, ihdr.Pid); err != nil {
		return fmt.Errorf("Write(Pid): %s", err)
	}
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
	conn       *net.UnixConn
	mu         sync.Mutex
	wQueue     [][]byte
	rScratch   []byte
	rInbetween bytes.Buffer
	rBuf       *bufio.Reader
	msgs       chan *Imsg
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
		return nil, fmt.Errorf("DialUnix(%s): %s", path, err)
	}
	ibuf := &ImsgBuffer{
		conn:     conn,
		pid:      int32(os.Getpid()),
		rScratch: make([]byte, IbufReadLen),
	}
	ibuf.rBuf = bufio.NewReader(&ibuf.rInbetween)
	go ibuf.reader()

	return ibuf, nil
}

func (ibuf *ImsgBuffer) Compose(kind, peerID, pid uint32, data WireSerializer) error {
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
	buf := make([]byte, size)

	// TODO: rights/send FD

	if err = header.WireBytes(buf[0:imsgHeaderLen]); err != nil {
		return fmt.Errorf("header.WireBytes: %s", err)
	}
	if err = data.WireBytes(buf[imsgHeaderLen:]); err != nil {
		return fmt.Errorf("header.WireBytes: %s", err)
	}

	ibuf.mu.Lock()
	ibuf.wQueue = append(ibuf.wQueue, buf)
	ibuf.mu.Unlock()

	return nil
}

// Flush writes all pending buffers to the socket
func (ibuf *ImsgBuffer) Flush() {
	ibuf.mu.Lock()
	defer ibuf.mu.Unlock()

	for _, buf := range ibuf.wQueue {
		n, _, err := ibuf.conn.WriteMsgUnix(buf, nil, nil)
		if err != nil {
			log.Printf("ibuf.conn.Write: %s", err)
		} else if n != len(buf) {
			log.Printf("ibuf.conn.Write short: %d/%d", n, len(buf))
		}
	}
	ibuf.wQueue = nil
}

func (ibuf *ImsgBuffer) Get() (*Imsg, error) {
	result := <-ibuf.msgs
	if result == nil {
		return nil, fmt.Errorf("channel closed")
	}

	return result, nil
}

func (ibuf *ImsgBuffer) reader() {
	hBytes := make([]byte, imsgHeaderLen)

	for {
		n, _, _, _, err := ibuf.conn.ReadMsgUnix(ibuf.rScratch, nil)
		if err != nil {
			log.Printf("ReadMsgUnix: %s", err)
			return
		}

		buf := ibuf.rScratch[:n]
		ibuf.rInbetween.Write(buf)

		for {
			n, err = ibuf.rBuf.Read(hBytes)
			if err != nil {
				log.Printf("rBuf.Read: %s", err)
				return
			} else if n != imsgHeaderLen {
				log.Printf("rBuf.Read short: %d/%d", n, imsgHeaderLen)
				return
			}

			var header ImsgHeader
			if err = header.InitFromWireBytes(hBytes); err != nil {
				log.Printf("InitFromWireBytes: %s", err)
				return
			}

			// TODO: rights/receive FD

			var payload []byte
			payloadLen := int(header.Len) - imsgHeaderLen
			if payloadLen > 0 {
				payload := make([]byte, payloadLen)
				n, err := ibuf.rBuf.Read(payload)
				if err != nil {
					log.Printf("rBuf.Read 2: %s", err)
					return
				} else if n != payloadLen {
					log.Printf("rBuf.Read 2 short: %d/%d", n, payloadLen)
					return
				}
			}
			imsg := &Imsg{header, payload}
			ibuf.msgs <- imsg
		}
	}
}
