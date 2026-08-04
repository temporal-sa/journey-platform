package testkit

import "testing"
import (
	"os"
	"path/filepath"
	"strings"
)

func TestToolVersionPinning(t *testing.T) {
	// Find repo root relative to current package directory
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to resolve root path: %v", err)
	}

	toolVersionsPath := filepath.Join(root, ".tool-versions")
	content, err := os.ReadFile(toolVersionsPath)
	if err != nil {
		t.Fatalf("failed to read .tool-versions: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	foundGo := false
	foundNode := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			t.Errorf("unpinned or invalid line in .tool-versions: %s", line)
			continue
		}
		tool, version := parts[0], parts[1]
		if version == "latest" || version == "*" {
			t.Errorf("unpinned version detected for tool %s: %s", tool, version)
		}
		if tool == "golang" || tool == "go" {
			foundGo = true
		}
		if tool == "nodejs" || tool == "node" {
			foundNode = true
		}
	}

	if !foundGo {
		t.Error("golang version not pinned in .tool-versions")
	}
	if !foundNode {
		t.Error("nodejs version not pinned in .tool-versions")
	}

	nvmrcPath := filepath.Join(root, ".nvmrc")
	nvmrcContent, err := os.ReadFile(nvmrcPath)
	if err != nil {
		t.Fatalf("failed to read .nvmrc: %v", err)
	}
	nvmrcVersion := strings.TrimSpace(string(nvmrcContent))
	if nvmrcVersion == "" || nvmrcVersion == "latest" || nvmrcVersion == "*" {
		t.Errorf("invalid or unpinned version in .nvmrc: %s", nvmrcVersion)
	}
}
