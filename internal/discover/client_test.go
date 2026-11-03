package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestControllerFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := []byte(`
external-controller: 127.0.0.1:9097
secret: test-secret
proxies: []
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	endpoint, secret := controllerFromConfig(path)
	if endpoint != "http://127.0.0.1:9097" {
		t.Fatalf("endpoint = %q", endpoint)
	}
	if secret != "test-secret" {
		t.Fatalf("secret = %q", secret)
	}
}

func TestControllerFromConfigDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("external-controller: \"\"\nproxies: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	endpoint, secret := controllerFromConfig(path)
	if endpoint != "" || secret != "" {
		t.Fatalf("endpoint = %q secret = %q", endpoint, secret)
	}
}

func TestNormalizeControllerURL(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:9090":         "http://127.0.0.1:9090",
		":9097":                  "http://127.0.0.1:9097",
		"0.0.0.0:9090":           "http://127.0.0.1:9090",
		"http://localhost:9090/": "http://localhost:9090",
	}
	for input, want := range cases {
		got, err := normalizeControllerURL(input)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if got != want {
			t.Errorf("%q = %q, want %q", input, got, want)
		}
	}
}
