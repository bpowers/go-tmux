// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// StartServer requires you have sessions specified in your tmux
// config, otherwise it is effectively a noop.
func StartServer() error {
	_, err := execCommand("start-server")
	return err
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
	cmdArgs := make([]string, 0, 16)
	cmdArgs = append(cmdArgs, "new-session", "-d", "-s", sessionName)
	if windowName != "" {
		cmdArgs = append(cmdArgs, "-n", windowName)
	}
	cmdArgs = append(cmdArgs, args...)
	out, err := Command(cmdArgs...)
	if err != nil {
		return Session{}, fmt.Errorf("Command(new-session): %s", err)
	}
	return GetSession(sessionName)
}

func NewWindow(target, windowName string, args ...string) (Window, error) {
	cmdArgs := make([]string, 0, 16)
	cmdArgs = append(cmdArgs, "new-window", "-a", "-t", target)
	if windowName != "" {
		cmdArgs = append(cmdArgs, "-n", windowName)
	}
	cmdArgs = append(cmdArgs, args...)
	out, err := Command(cmdArgs...)
	if err != nil {
		return Window{}, fmt.Errorf("Command(new-window): %s", err)
	}
	fmt.Printf("out2: `%s`\n", out)
	return GetWindow(windowName)
}

func SplitWindow(target string, args ...string) error {
	cmdArgs := make([]string, 0, 3+len(args))
	cmdArgs = append(cmdArgs, "split-window", "-t", target)
	cmdArgs = append(cmdArgs, args...)
	_, err := Command(cmdArgs...)
	return err
}

func ResizePane(target string, args ...string) error {
	cmdArgs := make([]string, 0, 3+len(args))
	cmdArgs = append(cmdArgs, "resize-pane", "-t", target)
	cmdArgs = append(cmdArgs, args...)
	_, err := Command(cmdArgs...)
	return err
}

func SelectPane(target string) error {
	_, err := Command("select-pane", "-t", target)
	return err
}
