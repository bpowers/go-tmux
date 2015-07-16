// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"fmt"
	"testing"
)

func TestListSessions(t *testing.T) {
	sessions, err := ListSessions()
	if err != nil {
		t.Errorf("ListSessions: %s", err)
	}
	for _, session := range sessions {
		fmt.Printf("session: %#v\n", session)
	}
}

func TestListWindows(t *testing.T) {
	windows, err := ListWindows()
	if err != nil {
		t.Errorf("ListWindows: %s", err)
	}
	for _, window := range windows {
		fmt.Printf("window: %#v\n", window)
	}
	//sessions := ListSessions()
	//if len(sessions) != 0 {
	//	t.Errorf("non-zero session length")
	//	return
	//}
}

func TestClient(t *testing.T) {
	msg, err := Command("list-windows")
	if err != nil {
		t.Errorf("Command: %s", err)
		return
	}
	fmt.Printf("RESULT: `%s`\n", string(msg))
}
