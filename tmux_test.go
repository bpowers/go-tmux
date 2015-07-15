// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"fmt"
	"testing"
)

func TestListSessions(t *testing.T) {
	//sessions := ListSessions()
	//if len(sessions) != 0 {
	//	t.Errorf("non-zero session length")
	//	return
	//}
}

func TestClient(t *testing.T) {
	path := SocketPath("")

	client, err := NewClient(path)
	if err != nil {
		t.Errorf("NewClient: %s", err)
		return
	}
	client.WriteServer(MsgCommand, &MsgCommandData{[]string{"list-windows"}})
	client.Flush()

	for {
		msg, err := client.Ibuf.Get()
		if err == nil {
			t.Errorf("client: %s", err)
		}
		fmt.Printf("msg: %#v\n", msg)
	}
}
