package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Event is one daemon decision or notable occurrence.
type Event struct {
	Time      time.Time `json:"time"`
	Kind      string    `json:"kind"` // cycle | switch | rollback | error
	Group     string    `json:"group,omitempty"`
	From      string    `json:"from,omitempty"`
	To        string    `json:"to,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Best      string    `json:"best,omitempty"`
	BestScore float64   `json:"best_score,omitempty"`
	CurScore  float64   `json:"current_score,omitempty"`
	OK        bool      `json:"ok,omitempty"`
}

// Store appends events to a JSONL file. It is deliberately simple; a SQLite
// backend can implement the same interface later.
type Store struct {
	path string
}

// New opens (or creates) a JSONL history file.
func New(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("history path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}
	return &Store{path: path}, nil
}

// DefaultPath returns ~/.local/share/nodepulse/history.jsonl.
func DefaultPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-history.jsonl"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "nodepulse", "history.jsonl")
}

// Append writes one event.
func (s *Store) Append(event Event) error {
	if event.Time.IsZero() {
		event.Time = time.Now()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	return nil
}

// Recent returns up to n events, newest last.
func (s *Store) Recent(n int) ([]Event, error) {
	if n <= 0 {
		n = 20
	}
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(events) > n {
		events = events[len(events)-n:]
	}
	return events, nil
}
