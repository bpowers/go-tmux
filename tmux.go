// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Session struct {
	Name     string
	NWindows int
	Created  string
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

func ListSessions() ([]Session, error) {
	out, err := Command("list-sessions", "-F",
		`{"Name":"#{session_name}","NWindow":#{session_windows},`+
			`"Created":"#{session_created_string}",`+
			`"Width":#{session_width},"Height":#{session_height}}`)
	if err != nil {
		return nil, fmt.Errorf("Command: %s", err)
	}
	bufs := bytes.Split(out, []byte{'\n'})
	sessions := make([]Session, len(bufs))
	for i, buf := range bufs {
		err := json.Unmarshal(buf, &sessions[i])
		if err != nil {
			return nil, fmt.Errorf("Unmarshal(%s): %s", string(buf), err)
		}
	}
	return sessions, nil
}

func GetSession(name string) (Session, bool) {
	sessions, err := ListSessions()
	if err != nil {
		return Session{}, false
	}
	for _, s := range sessions {
		if s.Name == name {
			return s, true
		}
	}
	return Session{}, false
}

func ListWindows() ([]Window, error) {
	out, err := Command("list-windows", "-F",
		`{"Index":#{window_index},"Name":"#{window_name}",`+
			`"NPanes":#{window_panes},"Width":#{window_width},`+
			`"Height":#{window_height}}`)
	if err != nil {
		return nil, fmt.Errorf("Command: %s", err)
	}
	windowBufs := bytes.Split(out, []byte{'\n'})
	windows := make([]Window, len(windowBufs))
	for i, buf := range windowBufs {
		err := json.Unmarshal(buf, &windows[i])
		if err != nil {
			return nil, fmt.Errorf("Unmarshal: %s", err)
		}
	}
	return windows, nil
}

func GetWindow(name string) (Window, bool) {
	windows, err := ListWindows()
	if err != nil {
		return Window{}, false
	}
	for _, s := range windows {
		if s.Name == name {
			return s, true
		}
	}
	return Window{}, false
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
