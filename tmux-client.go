// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

const (
	ProtocolVersion = 8

	MsgVersion MsgType = 12

	MsgIdentifyFlags MsgType = 100
	MsgIdentifyTerm
	MsgIdentifyTTYName
	MsgIdentifyCWD
	MsgIdentifyStdin
	MsgIdentifyEnviron
	MsgIdentifyDone
	MsgIdentifyClientPid

	MsgCommand MsgType = 200
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

