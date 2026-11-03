//go:build windows

package autostart

import (
	"fmt"
	"os/exec"
)

const runKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

// Path returns the registry value name used for the login item.
func Path() string {
	return runKey + `\NodePulse`
}

// Enabled reports whether the login item exists.
func Enabled() bool {
	return exec.Command("reg", "query", runKey, "/v", "NodePulse").Run() == nil
}

// Set creates or removes the HKCU Run entry.
func Set(enabled bool) error {
	if !enabled {
		cmd := exec.Command("reg", "delete", runKey, "/v", "NodePulse", "/f")
		_ = cmd.Run()
		return nil
	}
	exe, err := executable()
	if err != nil {
		return err
	}
	value := fmt.Sprintf(`"%s" --background`, exe)
	if out, err := exec.Command("reg", "add", runKey, "/v", "NodePulse", "/t", "REG_SZ", "/d", value, "/f").CombinedOutput(); err != nil {
		return fmt.Errorf("set login item: %v: %s", err, out)
	}
	return nil
}
