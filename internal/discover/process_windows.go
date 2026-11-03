//go:build windows

package discover

import (
	"os/exec"
	"strings"
)

func runningProcesses() []string {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(string(out), "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) == 0 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(fields[0]), `"`)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
