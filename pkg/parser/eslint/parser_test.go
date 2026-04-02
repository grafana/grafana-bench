package eslint

import (
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
)

const testPkg = "@grafana/grafana"

// --- testdata-backed tests ---

func TestParseESLintReport_FullReport(t *testing.T) {
	f, err := os.Open("testdata/full-report.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	summary, err := ParseESLintReport(f, testPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Status != executor.SuitePassed {
		t.Errorf("expected SuitePassed, got %v", summary.Status)
	}

	if len(summary.Metrics) != 0 {
		t.Errorf("expected no metrics, got %d", len(summary.Metrics))
	}
}

func TestParseESLintReport_SingleEntry(t *testing.T) {
	f, err := os.Open("testdata/single-report.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	summary, err := ParseESLintReport(f, testPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary.Metrics) != 0 {
		t.Errorf("expected no metrics, got %d", len(summary.Metrics))
	}
}

func TestParseESLintReport_NoEntries(t *testing.T) {
	f, err := os.Open("testdata/no-entries.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	summary, err := ParseESLintReport(f, testPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary.Metrics) != 0 {
		t.Errorf("expected no metrics, got %d", len(summary.Metrics))
	}
}

// --- error / edge-case tests ---

func TestParseESLintReport_MissingSummary(t *testing.T) {
	f, err := os.Open("testdata/missing-suppressions.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	_, err = ParseESLintReport(f, testPkg)
	if err == nil {
		t.Fatal("expected an error for empty/invalid JSON, got nil")
	}
}

func TestParseESLintReport_MissingCodeowner(t *testing.T) {
	input := strings.NewReader(`{"summary":[{"suppressions":10}]}`)

	_, err := ParseESLintReport(input, testPkg)
	if err == nil {
		t.Fatal("expected an error when codeowner is missing, got nil")
	}
}

func TestParseESLintReport_MissingSuppressions(t *testing.T) {
	// suppressions field absent — entry should be silently skipped, no error.
	input := strings.NewReader(`{"summary":[{"codeowner":"@grafana/x"}]}`)

	summary, err := ParseESLintReport(input, testPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary.Metrics) != 0 {
		t.Errorf("expected no metrics, got %d", len(summary.Metrics))
	}
}

func TestParseESLintReport_PackageLabel(t *testing.T) {
	f, err := os.Open("testdata/single-report.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	summary, err := ParseESLintReport(f, "@grafana/scenes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Attributes["package"] != "@grafana/scenes" {
		t.Errorf("expected package attribute @grafana/scenes, got %q", summary.Attributes["package"])
	}
}

func TestParseESLintReport_EmptyPackage(t *testing.T) {
	f, err := os.Open("testdata/single-report.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	summary, err := ParseESLintReport(f, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := summary.Attributes["package"]; ok {
		t.Error("expected no package attribute when pkg is empty")
	}
}

func TestParseESLintReport_InvalidJSON(t *testing.T) {
	_, err := ParseESLintReport(strings.NewReader("{bad json}"), testPkg)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
