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

type Client struct {
	TTY         string
	SessionName string
	Width       int
	Height      int
	TermName    string
	UTF8        bool
	ReadOnly    bool
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

func ListClients() ([]Client, error) {
	out, err := Command("list-clients", "-F",
		`{"TTY":"#{client_tty}",`+
			`"SessionName":"#{session_name}",`+
			`"Width":#{client_width},`+
			`"Height":#{client_height},`+
			`"TermName":"#{client_termname}",`+
			`"UTF8":#{?client_utf8,true,false},`+
			`"ReadOnly":#{?client_readonly,true,false}}`)
	if err != nil {
		return nil, fmt.Errorf("Command: %s", err)
	}
	bufs := bytes.Split(out, []byte{'\n'})
	clients := make([]Client, len(bufs))
	for i, buf := range bufs {
		err := json.Unmarshal(buf, &clients[i])
		if err != nil {
			return nil, fmt.Errorf("Unmarshal(%s): %s", string(buf), err)
		}
	}
	return clients, nil
}

func GetClient(tty string) (Client, error) {
	clients, err := ListClients()
	if err != nil {
		return Client{}, err
	}
	for _, c := range clients {
		if c.TTY == tty {
			return c, nil
		}
	}
	return Client{}, NotFound
}

func SwitchClient(tty, target string) error {
	_, err := Command("switch-client", "-c", tty, "-t", target)
	return err
}

func NewSession(sessionName, windowName string, args ...string) (Session, error) {
	cmdArgs := make([]string, 0, 16)
	cmdArgs = append(cmdArgs, "new-session", "-c", "/", "-d", "-s", sessionName)
	if windowName != "" {
		cmdArgs = append(cmdArgs, "-n", windowName)
	}
	cmdArgs = append(cmdArgs, args...)
	_, err := Command(cmdArgs...)
	if err != nil {
		return Session{}, fmt.Errorf("Command(new-session): %s", err)
	}
	return GetSession(sessionName)
}

func NewWindow(target, windowName string, args ...string) (Window, error) {
	cmdArgs := make([]string, 0, 16)
	cmdArgs = append(cmdArgs, "new-window", "-t", target)
	if windowName != "" {
		cmdArgs = append(cmdArgs, "-n", windowName)
	}
	cmdArgs = append(cmdArgs, args...)
	_, err := Command(cmdArgs...)
	if err != nil {
		return Window{}, fmt.Errorf("Command(new-window): %s", err)
	}
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
