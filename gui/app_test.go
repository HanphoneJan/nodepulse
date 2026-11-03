package main

import (
	"os"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/results"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/settings"
)

func TestResultsSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := settings.Save("", settings.Defaults()); err != nil {
		t.Fatal(err)
	}
	if err := results.Save(results.DefaultPath(), results.File{
		GeneratedAt: "2026-11-03T00:00:00Z",
		Rounds:      5,
		Results:     []score.Result{{Name: "A", Success: 5, Total: 5}},
	}); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	got := app.Results()
	if len(got) != 1 || got[0].Name != "A" {
		t.Fatalf("restored results = %+v", got)
	}

	app.ClearResults()
	if _, err := os.Stat(results.DefaultPath()); !os.IsNotExist(err) {
		t.Fatalf("results file should be removed, got %v", err)
	}
}

func TestResultsIgnoredForDifferentEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	s := settings.Defaults()
	s.ConfigPath = "/subscriptions/current.yaml"
	if err := settings.Save("", s); err != nil {
		t.Fatal(err)
	}
	if err := results.Save(results.DefaultPath(), results.File{
		ConfigPath: "/subscriptions/old.yaml",
		Results:    []score.Result{{Name: "A", Success: 5, Total: 5}},
	}); err != nil {
		t.Fatal(err)
	}
	if got := NewApp().Results(); len(got) != 0 {
		t.Fatalf("stale results should be ignored, got %+v", got)
	}
}
