// Package lock provides a single-writer lock for the daemon across GUI, Web
// and CLI processes.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// DefaultPath returns ~/.local/share/nodepulse/daemon.lock.
func DefaultPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-daemon.lock"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "nodepulse", "daemon.lock")
}

// Lock is a held daemon lock.
type Lock struct {
	path string
	file *os.File
	once sync.Once
}

// Acquire takes the daemon lock or explains which process holds it.
func Acquire(path string) (*Lock, error) {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err == nil {
			pid := os.Getpid()
			if _, writeErr := fmt.Fprintf(file, "%d\n", pid); writeErr != nil {
				file.Close()
				_ = os.Remove(path)
				return nil, fmt.Errorf("write daemon lock: %w", writeErr)
			}
			_ = file.Sync()
			return &Lock{path: path, file: file}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("create daemon lock: %w", err)
		}
		pid, readErr := readPID(path)
		if readErr == nil && pid > 0 && processAlive(pid) {
			return nil, fmt.Errorf("另一个 NodePulse 实例正在运行自动选择（PID %d）", pid)
		}
		// Stale or unreadable lock: remove and retry. Another process may win
		// the next O_EXCL race, in which case the loop reports its PID.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove stale daemon lock: %w", err)
		}
	}
	return nil, fmt.Errorf("无法获取自动选择锁，请稍后重试")
}

// Release removes the lock. It is safe to call more than once.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	var err error
	l.once.Do(func() {
		if l.file != nil {
			_ = l.file.Close()
		}
		if removeErr := os.Remove(l.path); removeErr != nil && !os.IsNotExist(removeErr) {
			err = removeErr
		}
	})
	return err
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid in lock file")
	}
	return pid, nil
}
