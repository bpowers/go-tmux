// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path"
)

//go:generate stringer -type MsgType

const (
	ProtocolVersion = 8

	MsgVersion MsgType = 12
)

const (
	MsgIdentifyFlags MsgType = 100 + iota
	MsgIdentifyTerm
	MsgIdentifyTTYName
	MsgIdentifyCWD
	MsgIdentifyStdin
	MsgIdentifyEnviron
	MsgIdentifyDone
	MsgIdentifyClientPid
)

const (
	MsgCommand MsgType = 200 + iota
	MsgDetach
	MsgDetachKill
	MsgExit
	MsgExited
	MsgExiting
	MsgLock
	MsgReady
	MsgResize
	MsgShell
	MsgShutdown
	MsgStderr
	MsgStdin
	MsgStdout
	MsgSuspend
	MsgUnlock
	MsgWakeup
)

type MsgType uint32

type MsgCommandData struct {
	Args []string
	// serialied as int32 followed by packed null terminlated
	// strings.
}

func (m *MsgCommandData) WireLen() int {
	size := 0
	for _, arg := range m.Args {
		size += len(arg) + 1
	}
	return size + 4
}

func (m *MsgCommandData) WireBytes(buf []byte) error {
	var err error
	var bb bytes.Buffer

	if err = binary.Write(&bb, binary.LittleEndian, int32(len(m.Args))); err != nil {
		return fmt.Errorf("i(%d) write: %s", int32(len(m.Args)), err)
	}
	for _, s := range m.Args {
		bb.WriteString(s)
		bb.WriteByte(0)
	}
	bbuf := bb.Bytes()
	if len(buf) < len(bbuf) {
		return fmt.Errorf("Int32 1 bad len %d/%d/4", len(buf), len(bbuf))
	}
	copy(buf, bbuf)

	return nil
}

func (m *MsgCommandData) InitFromWireBytes(buf []byte) error {
	// TODO: implement

	return nil
}

type MsgStdinData struct {
	size uint64
	data []byte
}

type MsgStdoutData struct {
	size uint64
	data []byte
}

type MsgStderrData struct {
	size uint64
	data []byte
}

func SocketPath(prefix string) string {
	if prefix == "" {
		prefix = "/tmp"
	}
	return path.Join(prefix, fmt.Sprintf("tmux-%d", os.Getuid()), "default")
}

type Client struct {
	Path string
	Ibuf *ImsgBuffer
}

func NewClient(path string) (*Client, error) {
	var err error
	c := &Client{Path: path}
	c.Ibuf, err = NewImsgBuffer(path)
	if err != nil {
		return nil, fmt.Errorf("newImsgBuffer: %s", err)
	}

	if err := c.sendIdentify(0); err != nil {
		return nil, fmt.Errorf("sendIdentify: %s", err)
	}

	return c, nil
}

func (c *Client) sendIdentify(flags int) error {
	fmt.Printf("sendIdnet\n")
	//c.WriteServer(MsgIdentifyFlags, &Int32{int32(flags)})
	//c.WriteServer(MsgIdentifyClientPid, &Int32{int32(os.Getpid())})
	c.WriteServer(MsgIdentifyDone, &Nil{})

	return nil
}

func (c *Client) WriteServer(kind MsgType, data WireSerializer) error {
	return c.Ibuf.Compose(uint32(kind), ProtocolVersion, 0xffffffff, data)
}

func (c *Client) Flush() {
	c.Ibuf.Flush()
}
