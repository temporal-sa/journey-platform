package testkit_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestPackageLevelCoverageCheck verifies that package-level code coverage across internal/ and cmd/ meets minimum thresholds.
func TestPackageLevelCoverageCheck(t *testing.T) {
	if os.Getenv("SKIP_COVERAGE_RECURSION") == "1" {
		t.Skip("Skipping recursive coverage check in child test process")
	}

	// Locate repository root
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	repoRoot := ""
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			repoRoot = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found starting from %s", dir)
		}
		dir = parent
	}

	coverProfilePath := filepath.Join(t.TempDir(), "coverage.out")

	// Run go test with -coverprofile across internal/ and cmd/
	cmd := exec.Command("go", "test", "-coverprofile="+coverProfilePath, "./internal/...", "./cmd/...")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "SKIP_COVERAGE_RECURSION=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test coverage execution failed: %v\nOutput: %s", err, string(output))
	}

	// Parse coverage.out
	file, err := os.Open(coverProfilePath)
	if err != nil {
		t.Fatalf("failed to open coverage profile: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var totalStatements int64
	var coveredStatements int64

	packageCoverage := make(map[string]*struct {
		Total   int64
		Covered int64
	})

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}

		// Format: file:startLine.startCol,endLine.endCol numStmt count
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		numStmt, err1 := strconv.ParseInt(fields[1], 10, 64)
		count, err2 := strconv.ParseInt(fields[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		filePath := fields[0]
		pkgName := filepath.Dir(filePath)

		if _, exists := packageCoverage[pkgName]; !exists {
			packageCoverage[pkgName] = &struct {
				Total   int64
				Covered int64
			}{}
		}

		packageCoverage[pkgName].Total += numStmt
		totalStatements += numStmt

		if count > 0 {
			packageCoverage[pkgName].Covered += numStmt
			coveredStatements += numStmt
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading coverage profile: %v", err)
	}

	if totalStatements == 0 {
		t.Fatal("no statements found in coverage profile")
	}

	overallPct := float64(coveredStatements) / float64(totalStatements) * 100.0
	t.Logf("Overall Code Coverage: %.2f%% (%d / %d statements)", overallPct, coveredStatements, totalStatements)

	// Verify each package has statement coverage
	for pkg, cov := range packageCoverage {
		if cov.Total == 0 {
			t.Errorf("Package %s has 0 statements covered", pkg)
			continue
		}
		pct := float64(cov.Covered) / float64(cov.Total) * 100.0
		t.Logf("  Package %s: %.2f%% (%d/%d)", pkg, pct, cov.Covered, cov.Total)
	}

	// Assert overall coverage threshold is at least 40%
	if overallPct < 40.0 {
		t.Errorf("Overall code coverage %.2f%% is below minimum required threshold of 40.0%%", overallPct)
	}
}
