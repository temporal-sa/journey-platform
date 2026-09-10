package test

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEIntegrity(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		content, err = os.ReadFile("README.md")
		if err != nil {
			t.Fatalf("failed to read README.md: %v", err)
		}
	}

	text := string(content)

	requiredSections := []string{
		"# Event-Driven Journey Platform",
		"## Repository Layout",
		"## Local Development Prerequisites",
		"## Developer Command Surface",
		"## Local Infrastructure Stack",
		"## Step-by-Step Local Golden Path",
		"## Verification & Quality Assurance",
	}

	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			t.Errorf("README.md missing required section: %s", section)
		}
	}

	requiredCommands := []string{
		"make bootstrap",
		"make dev",
		"make seed",
		"make verify",
		"make acceptance",
		"make reset",
		"make down",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(text, cmd) {
			t.Errorf("README.md missing command documentation: %s", cmd)
		}
	}

	requiredURLs := []string{
		"http://localhost:3002",
		"http://localhost:8087/health",
		"http://localhost:8233",
		"http://localhost:8025",
		"http://localhost:9001",
		"http://localhost:8089/health",
	}

	for _, u := range requiredURLs {
		if !strings.Contains(text, u) {
			t.Errorf("README.md missing local URL: %s", u)
		}
	}
}
