// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"io"
	"os"
)

type IBuffer struct {
}

type WireSerializer interface {
	InitFromBytes([]byte) error
	ToBytes() []byte
}

type ImsgBuffer struct {
	conn io.ReadWriteCloser
	buf  bytes.Buffer
	// Linux kernel defines pid_t as a 32-bit signed int in
	// include/uapi/asm-generic/posix_types.h
	pid  int32
}

func NewImsgBuffer(conn io.ReadWriteCloser) *ImsgBuffer {
	return &ImsgBuffer{
		conn: conn,
		pid:  int32(os.Getpid()),
	}
}

func (ibuf *ImsgBuffer) Compose(kind uint32, peerid, pid int32, data WireSerializer) error {

	// creates a buffer w/ header
	wbuf := ibuf.Create()
	// add the data to the buffer
	wbuf.Add(data)

	// TODO: support FD passing

	// enqueue the buffer on the msgbuf
	wbuf.Close()

	return nil
}

func (ibuf *ImsgBuffer) Create() *IBuffer {
	return nil
}

// Flush writes all pending buffers to the 
func (ibuf *ImsgBuffer) Flush() {
}

func (imsg *IBuffer) Add(data WireSerializer) error {
	return nil
}

func (imsg *IBuffer) Close() error {
	return nil
}
