// Copyright 2015 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package tmux

import (
	"testing"
)

func TestListSessions(t *testing.T) {
	sessions := ListSessions()
	if len(sessions) != 0 {
		t.Errorf("non-zero session length")
	}
}
