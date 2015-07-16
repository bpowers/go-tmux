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

var NotFound = fmt.Errorf("object not found")

type Session struct {
	Name     string
	NWindows int
	Created  string
	Width    int
	Height   int
}

type Window struct {
	Index       int
	Name        string
	SessionName string
	NPanes      int
	Width       int
	Height      int
	Active      bool
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

func GetSession(name string) (Session, error) {
	sessions, err := ListSessions()
	if err != nil {
		return Session{}, err
	}
	for _, s := range sessions {
		if s.Name == name {
			return s, nil
		}
	}
	return Session{}, NotFound
}

func ListWindows() ([]Window, error) {
	out, err := Command("list-windows", "-a", "-F",
		`{"Index":#{window_index},"Name":"#{window_name}",`+
			`"SessionName":"#{session_name}",`+
			`"Active":#{?window_active,true,false},`+
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

func GetWindow(name string) (Window, error) {
	windows, err := ListWindows()
	if err != nil {
		return Window{}, err
	}
	for _, s := range windows {
		if s.Name == name {
			return s, nil
		}
	}
	return Window{}, NotFound
}

func NewSession(sessionName, windowName string, args ...string) (Session, error) {
	cmdArgs := make([]string, 0, 12)
	cmdArgs = append(cmdArgs, "new-session", "-d", "-s", sessionName)
	if windowName != "" {
		cmdArgs = append(cmdArgs, "-n", windowName)
	}
	cmdArgs = append(cmdArgs, args...)
	out, err := Command(cmdArgs...)
	if err != nil {
		return Session{}, fmt.Errorf("Command(new-session): %s", err)
	}
	fmt.Printf("out: `%s`\n", out)
	return GetSession(sessionName)
}

func NewWindow(target, windowName string, args ...string) (Window, error) {
	return Window{}, nil
}

func ResizePane(target string) error {
	return nil
}
