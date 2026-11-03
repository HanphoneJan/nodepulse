//go:build !windows

package discover

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func runningProcesses() []string {
	out, err := exec.Command("ps", "-A", "-o", "comm=").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(string(out), "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		names = append(names, filepath.Base(name))
	}
	return names
}
