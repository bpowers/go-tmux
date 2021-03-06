// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
)

//go:generate stringer -type MsgType,ClientFlag -output enum_strings.go

const (
	ProtocolVersion = 8

	MsgVersion MsgType = 12
)

const (
	MsgIdentifyFlags MsgType = 100 + iota
	MsgIdentifyTerm
	MsgIdentifyTTYName
	MsgIdentifyOldCWD
	MsgIdentifyStdin
	MsgIdentifyEnviron
	MsgIdentifyDone
	MsgIdentifyClientPid
	MsgIdentifyCWD
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

const (
	ClientTerminal ClientFlag = 1 << iota
	_
	ClientExit
	ClientRedraw
	ClientStatus
	CientRepeat
	ClientSuspended
	ClientBad
	ClientIdentify
	ClientDead
	ClientBorders
	ClientReadOnly
	ClientRedrawWindow
	ClientControl
	ClientControlControl
	ClientFocused
	ClientUTF8
	Client256Colours
	ClientIdentified
)

type ClientFlag int
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

type MsgStdioData struct {
	Size int64
	Data []byte
}

func (m *MsgStdioData) WireLen() int {
	return len(m.Data) + 8
}

func (m *MsgStdioData) WireBytes(buf []byte) error {
	var err error
	var bb bytes.Buffer

	m.Size = int64(len(m.Data))
	if err = binary.Write(&bb, binary.LittleEndian, int64(m.Size)); err != nil {
		return fmt.Errorf("i(%d) write: %s", int64(m.Size), err)
	}
	for _, b := range m.Data {
		bb.WriteByte(b)
	}
	bbuf := bb.Bytes()
	if len(buf) < len(bbuf) {
		return fmt.Errorf("StdioData 1 bad len %d/%d/4", len(buf), len(bbuf))
	}
	copy(buf, bbuf)

	return nil
}

func (m *MsgStdioData) InitFromWireBytes(buf []byte) error {
	if len(buf) < 8 {
		m.Data = nil
		m.Size = 0
		return nil
	}
	size := int64(binary.LittleEndian.Uint64(buf[:8]))
	buf = buf[8:]
	if size < int64(len(buf)) {
		buf = buf[:size]
	}

	m.Data = make([]byte, len(buf))
	copy(m.Data, buf)
	m.Size = size

	return nil
}

func (m *MsgStdioData) String() string {
	end := bytes.IndexByte(m.Data, 0)
	if end < 0 {
		end = len(m.Data)
	}
	return string(bytes.TrimSpace(m.Data[:end]))
}

func SocketPath(prefix string) string {
	if prefix == "" {
		prefix = "/tmp"
	}
	return path.Join(prefix, fmt.Sprintf("tmux-%d", os.Getuid()), "default")
}

type client struct {
	Path string
	Ibuf *ImsgBuffer
}

func newClient(path string) (*client, error) {
	var err error
	c := &client{Path: path}
	c.Ibuf, err = NewImsgBuffer(path)
	if err != nil {
		if err == ErrNoSocket {
			return nil, ErrNoSocket
		} else {
			return nil, fmt.Errorf("newImsgBuffer: %s", err)
		}
	}

	if err := c.sendIdentify(ClientUTF8 | Client256Colours); err != nil {
		return nil, fmt.Errorf("sendIdentify: %s", err)
	}

	return c, nil
}

func (c *client) sendIdentify(flags ClientFlag) error {
	c.WriteServer(MsgIdentifyFlags, &Int32{int32(flags)}, nil)
	c.WriteServer(MsgIdentifyClientPid, &Int32{int32(os.Getpid())}, nil)
	c.WriteServer(MsgIdentifyTerm, &String{os.Getenv("TERM")}, nil)
	//cwd, err := os.Open(".")
	//if err != nil {
	//	return fmt.Errorf("Open(.): %s", err)
	//}
	//c.WriteServer(MsgIdentifyOldCWD, &Nil{}, cwd)
	cwd, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("Abs(.): %s", err)
	}
	c.WriteServer(MsgIdentifyCWD, &String{cwd}, nil)
	stdin, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("Dup(stdin): %s", err)
	}
	c.WriteServer(MsgIdentifyStdin, &Nil{}, os.NewFile(uintptr(stdin), "/dev/stdin"))
	c.WriteServer(MsgIdentifyDone, &Nil{}, nil)

	return nil
}

func (c *client) WriteServer(kind MsgType, data WireSerializer, f *os.File) error {
	return c.Ibuf.Compose(uint32(kind), ProtocolVersion, 0xffffffff, data, f)
}

func (c *client) Flush() {
	c.Ibuf.Flush()
}

func (c *client) Get() (*Imsg, error) {
	msg, err := c.Ibuf.Get()
	if err != nil {
		return nil, fmt.Errorf("c.Ibuf.Get(): %s", err)
	}
	kind := MsgType(msg.Header.Type)
	if kind == MsgExit || kind == MsgShutdown || kind == MsgExited {
		c.Ibuf.Close()
	}
	return msg, err
}

func execCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("tmux", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Run: %s", err)
	}
	return out.Bytes(), nil
}

func Command(args ...string) ([]byte, error) {
	path := SocketPath("")

	client, err := newClient(path)
	if err != nil && err == ErrNoSocket {
		return execCommand(args...)
	} else if err != nil {
		return nil, fmt.Errorf("newClient: %s", err)
	}
	client.WriteServer(MsgCommand, &MsgCommandData{args}, nil)
	client.Flush()

	outs := make([][]byte, 0, 1)
	var errs [][]byte

outer:
	for {
		imsg, err := client.Get()
		if err != nil {
			if err != ErrImsgBufferClosed {
				return nil, fmt.Errorf("client.Get: %s", err)
			}
			break
		}

		switch kind := MsgType(imsg.Header.Type); kind {
		case MsgStdin:
			// ignore
		case MsgStdout:
			var payload MsgStdioData
			err := payload.InitFromWireBytes(imsg.Data)
			if err != nil {
				return nil, fmt.Errorf("payload.InitFromWireBytes: %s", err)
			}
			// for now, ignore the prefix + suffix emitted
			// by command mode.
			if bytes.HasPrefix(payload.Data, []byte("%begin ")) ||
				bytes.HasPrefix(payload.Data, []byte("%end ")) {
				continue
			}
			outs = append(outs, payload.Data)
		case MsgStderr:
			var payload MsgStdioData
			err := payload.InitFromWireBytes(imsg.Data)
			if err != nil {
				return nil, fmt.Errorf("payload.InitFromWireBytes: %s", err)
			}
			errs = append(errs, payload.Data)
		case MsgExit, MsgShutdown, MsgExited:
			break outer
		default:
			return nil, fmt.Errorf("unknown imsg(%s)\n", kind)
		}
	}

	if len(errs) > 0 {
		result := make([]byte, 0)
		for _, buf := range outs {
			result = append(result, buf...)
		}
		return nil, fmt.Errorf("stderr: %s", string(bytes.TrimSpace(result)))
	}

	size := 0
	for _, buf := range outs {
		size += len(buf)
	}
	result := make([]byte, 0, size)
	for _, buf := range outs {
		result = append(result, buf...)
	}
	return bytes.TrimSpace(result), nil
}
