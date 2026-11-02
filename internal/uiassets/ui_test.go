package uiassets

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var usedIcons = []string{
	"dashboard", "list", "settings", "history",
	"refresh", "link", "download", "check", "play", "search",
	"sun", "moon", "monitor", "chevron", "palette",
	"monitoring", "speed", "checkCircle", "wifiOff", "close", "swap",
}

func TestIconSpriteMatchesManifest(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("icons", "manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, line := range strings.Split(string(manifest), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		declared[strings.Fields(line)[0]] = true
	}

	sprite, err := os.ReadFile(filepath.Join("dist", "icons.svg"))
	if err != nil {
		t.Fatal(err)
	}
	symbols := map[string]bool{}
	for _, m := range regexp.MustCompile(`<symbol id="i-([^"]+)"`).FindAllStringSubmatch(string(sprite), -1) {
		symbols[m[1]] = true
	}
	if len(symbols) != len(declared) {
		t.Fatalf("sprite has %d symbols, manifest declares %d", len(symbols), len(declared))
	}
	for name := range declared {
		if !symbols[name] {
			t.Errorf("manifest icon %q is missing from icons.svg", name)
		}
	}
	for _, name := range usedIcons {
		if !declared[name] {
			t.Errorf("UI uses %q but it is not in icons/manifest.txt", name)
		}
	}
}

func TestNoHandWrittenIconPathsOrRemoteAssets(t *testing.T) {
	index, err := os.ReadFile(filepath.Join("dist", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(index), "<path ") {
		t.Error("index.html contains a hand-written SVG path; use icons.svg")
	}
	if strings.Contains(string(index), "fonts.googleapis") || strings.Contains(string(index), "cdn.") {
		t.Error("index.html must not reference remote fonts or a CDN")
	}
	jsFiles, err := filepath.Glob(filepath.Join("dist", "js", "*.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range jsFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "<path ") {
			t.Errorf("%s contains a hand-written SVG path; use the Icon component", path)
		}
	}

	cssFiles, err := filepath.Glob(filepath.Join("dist", "css", "*.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range cssFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "@import") {
			t.Errorf("%s uses @import; load stylesheets explicitly", path)
		}
		if strings.Contains(text, "url(http") {
			t.Errorf("%s references a remote asset", path)
		}
	}
}

func TestGeneratedLicensesExist(t *testing.T) {
	paths := []string{
		filepath.Join("dist", "icons.LICENSE.txt"),
		filepath.Join("dist", "material-color-utilities.LICENSE.txt"),
		filepath.Join("dist", "vendor", "LICENSES", "preact-LICENSE.txt"),
		filepath.Join("dist", "vendor", "LICENSES", "htm-LICENSE.txt"),
		filepath.Join("dist", "vendor", "LICENSES", "floating-ui-LICENSE.txt"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing license %s: %v", path, err)
		}
	}
}

func TestJavaScriptIconNamesAreDeclared(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("icons", "manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, line := range strings.Split(string(manifest), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		declared[strings.Fields(line)[0]] = true
	}
	jsFiles, err := filepath.Glob(filepath.Join("dist", "js", "*.js"))
	if err != nil {
		t.Fatal(err)
	}
	iconAttr := regexp.MustCompile(`(?:Icon[^\n>]{0,120}name=|icon=)"([A-Za-z][A-Za-z0-9]*)"`)
	iconData := regexp.MustCompile(`icon:\s*"([A-Za-z][A-Za-z0-9]*)"`)
	for _, path := range jsFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range []*regexp.Regexp{iconAttr, iconData} {
			for _, m := range re.FindAllStringSubmatch(string(data), -1) {
				if !declared[m[1]] {
					t.Errorf("%s uses icon %q not declared in icons/manifest.txt", path, m[1])
				}
			}
		}
	}
}
