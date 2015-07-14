// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

type Session struct {
	Name     string
	NWindows int
	Created  time.Time
	Width    int
	Height   int
}

type Window struct {
	Index  int
	Name   string
	NPanes int
	Width  int
	Height int
}

func execTmux(args ...string) ([]byte, error) {
	cmd := exec.Command("tmux", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Run: %s", err)
	}
	return out.Bytes(), nil
}

func ListSessions() []Session {
	out, err := execTmux("list-sessions", "-F", "")
	fmt.Printf("OUT: %s\n", out)
	if err != nil {
		return []Session{}
	}
	return nil
}

func ListWindows() []Window {
	return nil
}

func GetSession(name string) (Session, bool) {
	sessions := ListSessions()
	for _, s := range sessions {
		if s.Name == name {
			return s, true
		}
	}
	return Session{}, false
}

func NewSession(sessionName, windowName string, args []string) error {
	return nil
}

func NewWindow(target, windowName string, args []string) error {
	return nil
}

func ResizePane(target string) error {
	return nil
}

func Connect() {

	// open socket
	// queue send identify
	// queue send command/msg_shell
	// flush imsg buffer
	// wait for response
}
