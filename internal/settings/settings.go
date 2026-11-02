package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Profile is one switchable environment: a client endpoint, a subscription,
// a managed group and the test parameters that go with them.
type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	API         string `json:"api"`
	Secret      string `json:"secret"`
	Group       string `json:"group"`
	ConfigPath  string `json:"config_path"`
	MihomoPath  string `json:"mihomo_path"`
	URL         string `json:"url"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	TimeoutMs   int    `json:"timeout_ms"`
}

// Settings is the GUI / web UI configuration.
//
// The top-level fields are the currently active environment. Profiles keeps
// every saved environment; Save synchronises the active profile from the
// top-level fields, and Load applies the active profile back to the top-level
// fields so existing callers keep working unchanged.
type Settings struct {
	API         string `json:"api"`
	Secret      string `json:"secret"`
	Group       string `json:"group"`
	ConfigPath  string `json:"config_path"`
	MihomoPath  string `json:"mihomo_path"`
	URL         string `json:"url"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	TimeoutMs   int    `json:"timeout_ms"`

	Profiles      []Profile `json:"profiles,omitempty"`
	ActiveProfile string    `json:"active_profile,omitempty"`

	// Background auto-selection is global, not per environment: it follows the
	// active environment's client/subscription but keeps the same policy.
	AutoOptimize   bool    `json:"auto_optimize,omitempty"`
	DaemonInterval int     `json:"daemon_interval,omitempty"`
	AutoMargin     float64 `json:"auto_margin,omitempty"`
	AutoConfirm    int     `json:"auto_confirm,omitempty"`
	AutoMinSuccess float64 `json:"auto_min_success,omitempty"`
}

// DefaultPath is ~/.config/nodepulse/settings.json.
func DefaultPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-settings.json"
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "nodepulse", "settings.json")
}

// Defaults returns sensible starting values.
func Defaults() Settings {
	return Settings{
		URL:            "https://www.gstatic.com/generate_204",
		Rounds:         5,
		Concurrency:    8,
		TimeoutMs:      5000,
		DaemonInterval: 300,
		AutoMargin:     0.2,
		AutoConfirm:    2,
		AutoMinSuccess: 0.8,
	}
}

// NewProfileID returns a stable, non-guessable identifier for a new profile.
func NewProfileID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "env-" + hex.EncodeToString(b[:])
	}
	return "env-default"
}

func normalizeProfile(p *Profile, index int) {
	if p.ID == "" {
		if index == 0 {
			p.ID = "default"
		} else {
			p.ID = NewProfileID()
		}
	}
	if p.Name == "" {
		if index == 0 {
			p.Name = "默认环境"
		} else {
			p.Name = fmt.Sprintf("环境 %d", index+1)
		}
	}
	if p.URL == "" {
		p.URL = "https://www.gstatic.com/generate_204"
	}
	if p.Rounds <= 0 {
		p.Rounds = 5
	}
	if p.Concurrency <= 0 {
		p.Concurrency = 8
	}
	if p.TimeoutMs <= 0 {
		p.TimeoutMs = 5000
	}
}

func (s *Settings) normalizeValues() {
	if s.URL == "" {
		s.URL = "https://www.gstatic.com/generate_204"
	}
	if s.Rounds <= 0 {
		s.Rounds = 5
	}
	if s.Concurrency <= 0 {
		s.Concurrency = 8
	}
	if s.TimeoutMs <= 0 {
		s.TimeoutMs = 5000
	}
	if s.DaemonInterval <= 0 {
		s.DaemonInterval = 300
	}
	if s.AutoMargin <= 0 {
		s.AutoMargin = 0.2
	}
	if s.AutoConfirm <= 0 {
		s.AutoConfirm = 2
	}
	if s.AutoMinSuccess <= 0 {
		s.AutoMinSuccess = 0.8
	}
}

func (s Settings) snapshotProfile() Profile {
	return Profile{
		API:         s.API,
		Secret:      s.Secret,
		Group:       s.Group,
		ConfigPath:  s.ConfigPath,
		MihomoPath:  s.MihomoPath,
		URL:         s.URL,
		Rounds:      s.Rounds,
		Concurrency: s.Concurrency,
		TimeoutMs:   s.TimeoutMs,
	}
}

func (s *Settings) applyProfile(p Profile) {
	s.API = p.API
	s.Secret = p.Secret
	s.Group = p.Group
	s.ConfigPath = p.ConfigPath
	s.MihomoPath = p.MihomoPath
	s.URL = p.URL
	s.Rounds = p.Rounds
	s.Concurrency = p.Concurrency
	s.TimeoutMs = p.TimeoutMs
	s.ActiveProfile = p.ID
}

// NormalizeProfiles makes the profile list structurally valid. A legacy
// settings file with no profiles becomes a single "默认环境".
func (s *Settings) NormalizeProfiles() {
	if len(s.Profiles) == 0 {
		p := s.snapshotProfile()
		normalizeProfile(&p, 0)
		p.ID = "default"
		p.Name = "默认环境"
		s.Profiles = []Profile{p}
		s.ActiveProfile = p.ID
		return
	}

	seen := map[string]bool{}
	for i := range s.Profiles {
		normalizeProfile(&s.Profiles[i], i)
		if seen[s.Profiles[i].ID] {
			s.Profiles[i].ID = NewProfileID()
		}
		seen[s.Profiles[i].ID] = true
	}
	if s.ActiveProfile == "" || !s.hasProfile(s.ActiveProfile) {
		s.ActiveProfile = s.Profiles[0].ID
	}
}

func (s Settings) hasProfile(id string) bool {
	for _, p := range s.Profiles {
		if p.ID == id {
			return true
		}
	}
	return false
}

// SyncActiveProfile copies the current top-level fields into the active
// profile, preserving its ID and display name.
func (s *Settings) SyncActiveProfile() {
	s.NormalizeProfiles()
	for i := range s.Profiles {
		if s.Profiles[i].ID != s.ActiveProfile {
			continue
		}
		p := s.snapshotProfile()
		p.ID = s.Profiles[i].ID
		p.Name = s.Profiles[i].Name
		s.Profiles[i] = p
		return
	}
}

// ActivateProfile applies a saved profile to the top-level fields.
func (s *Settings) ActivateProfile(id string) bool {
	for _, p := range s.Profiles {
		if p.ID == id {
			s.applyProfile(p)
			return true
		}
	}
	return false
}

// Profile returns the active profile.
func (s Settings) Profile() (Profile, bool) {
	for _, p := range s.Profiles {
		if p.ID == s.ActiveProfile {
			return p, true
		}
	}
	return Profile{}, false
}

func readRaw(path string) (Settings, error) {
	var s Settings
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// Load reads settings; a missing file returns defaults. Existing legacy files
// without profiles are migrated in memory.
func Load(path string) (Settings, error) {
	if path == "" {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s := Defaults()
			s.NormalizeProfiles()
			return s, nil
		}
		return Defaults(), fmt.Errorf("read settings: %w", err)
	}

	s := Defaults()
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse settings: %w", err)
	}
	s.normalizeValues()
	s.NormalizeProfiles()
	s.ActivateProfile(s.ActiveProfile)
	return s, nil
}

// Save writes settings with 0600 permissions. It synchronises the active
// profile from the top-level fields. If an older caller omitted profiles
// entirely, existing profiles on disk are preserved instead of wiped.
func Save(path string, s Settings) error {
	if path == "" {
		path = DefaultPath()
	}
	s.normalizeValues()
	if len(s.Profiles) == 0 {
		if existing, err := readRaw(path); err == nil && len(existing.Profiles) > 0 {
			s.Profiles = existing.Profiles
			if s.ActiveProfile == "" {
				s.ActiveProfile = existing.ActiveProfile
			}
		}
	}
	s.NormalizeProfiles()
	s.SyncActiveProfile()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	_ = os.Chmod(path, 0o600)
	return nil
}
