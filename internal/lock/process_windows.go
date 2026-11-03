//go:build windows

package lock

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// processAlive uses tasklist instead of pulling a Windows-only syscall
// dependency into the CLI module.
func processAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		return false
	}
	want := strconv.Itoa(pid)
	for _, field := range strings.Fields(string(out)) {
		if field == want {
			return true
		}
	}
	return false
}
