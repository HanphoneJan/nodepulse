package mcp

import (
	"fmt"
	"os"

	"github.com/HanphoneJan/nodepulse/internal/store"
)

func resourceList() []map[string]any {
	return []map[string]any{
		{
			"uri":         "nodepulse://results",
			"name":        "Latest results",
			"description": "The most recent results.json produced by nodes.test or the CLI.",
			"mimeType":    "application/json",
		},
		{
			"uri":         "nodepulse://history",
			"name":        "Daemon history",
			"description": "Recent daemon events (switch / rollback / error).",
			"mimeType":    "text/plain",
		},
	}
}

func readResource(uri string) (text, mime string, err error) {
	switch uri {
	case "nodepulse://results":
		path := os.Getenv("NODEPULSE_RESULTS")
		if path == "" {
			path = "results.json"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", "", fmt.Errorf("read results: %w", err)
		}
		return string(data), "application/json", nil
	case "nodepulse://history":
		path := os.Getenv("NODEPULSE_HISTORY")
		if path == "" {
			path = store.DefaultPath()
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return "no history yet", "text/plain", nil
			}
			return "", "", fmt.Errorf("read history: %w", err)
		}
		return string(data), "text/plain", nil
	default:
		return "", "", fmt.Errorf("unknown resource: %s", uri)
	}
}
