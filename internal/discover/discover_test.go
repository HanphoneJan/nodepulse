package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfilesDiscoversFlClashSubscriptions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := filepath.Join(home, ".local", "share", "com.follow.clash")
	dir := filepath.Join(base, "profiles")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(path, group string) {
		if err := os.WriteFile(path, []byte("proxy-groups:\n  - {name: "+group+", type: select, proxies: [A]}\nproxies:\n  - {name: A, type: ss, server: 127.0.0.1, port: 1}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(base, "config.yaml"), "一分机场")
	write(filepath.Join(dir, "1.yaml"), "赔钱机场")
	write(filepath.Join(dir, "profiles.yaml"), "metadata")
	write(filepath.Join(dir, "2.yaml"), "备用机场")

	profiles := Profiles()
	if len(profiles) != 3 {
		t.Fatalf("profiles = %+v", profiles)
	}
	labels := map[string]bool{}
	for _, profile := range profiles {
		labels[profile.Label] = true
		if profile.Client != "FlClash" {
			t.Fatalf("client = %q", profile.Client)
		}
	}
	for _, want := range []string{"一分机场", "赔钱机场", "备用机场"} {
		if !labels[want] {
			t.Fatalf("missing label %q in %+v", want, labels)
		}
	}
	if len(SubscriptionFiles()) != 3 {
		t.Fatalf("SubscriptionFiles = %+v", SubscriptionFiles())
	}
}
