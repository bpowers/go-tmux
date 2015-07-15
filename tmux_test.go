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
		imsg, err := client.Ibuf.Get()
		if err != nil {
			if err != ImsgBufferClosed {
				t.Errorf("client: %s", err)
			}
			break
		}
		kind := MsgType(imsg.Header.Type)
		if kind == MsgStdin || kind == MsgStdout {
			var payload MsgStdioData
			err := payload.InitFromWireBytes(imsg.Data)
			if err != nil {
				t.Errorf("payload.InitFromWireBytes: %s", err)
				return
			}
			fmt.Printf("imsg(%s): %s\n", kind, payload.String())
		} else {
			fmt.Printf("imsg(%s)\n", kind)
		}
	}
}
