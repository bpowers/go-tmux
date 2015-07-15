// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
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
	ArgC int32 // followed by packed argv
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

	return nil
}

func (c *Client) WriteOne(msg MsgType, data []byte) error {
	return nil
}

func (c *Client) WriteServer(msg MsgType, data WireSerializer) error {
	return nil
}
