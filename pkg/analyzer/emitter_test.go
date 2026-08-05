package analyzer

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func emitOne(t *testing.T, d ConfirmedDefect) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	e, err := NewEmitter(&buf, "json", []any{"tool", "bench", "service", d.Service})
	if err != nil {
		t.Fatalf("building emitter: %v", err)
	}
	e.EmitDefectConfirmed(d)
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatalf("emitted line not valid JSON: %v\nline: %s", err, buf.String())
	}
	return m
}

func mkDefect(axis, version string) ConfirmedDefect {
	return ConfirmedDefect{
		Service:             "grafana-clickhouse-datasource",
		RunStage:            "ci",
		TestFile:            "queries.spec.ts",
		TestFolder:          "tests/e2e",
		Version:             version,
		VersionAxis:         axis,
		PriorPassingVersion: "1.27.0",
		SignatureHash:       "a9c0f1b2d345",
		Confidence:          ConfidenceConfirmed,
		ConfidenceRuns:      3,
		PriorPassingRuns:    2,
		FirstFailureTime:    time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC),
		LastFailureTime:     time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		SourceRunIDs:        []string{"run-3", "run-4", "run-5"},
		AnalyzeWindow:       24 * time.Hour,
		AnalyzedAt:          time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC),
		RuleVersion:         RuleVersion,
	}
}

func TestEmitDefectConfirmedServiceAxis(t *testing.T) {
	m := emitOne(t, mkDefect(VersionAxisService, "1.27.1"))

	if got, _ := m["versionAxis"].(string); got != VersionAxisService {
		t.Errorf("expected versionAxis=%q, got %q", VersionAxisService, got)
	}
	if got, _ := m["serviceVersion"].(string); got != "1.27.1" {
		t.Errorf("expected serviceVersion=%q, got %q", "1.27.1", got)
	}
	if got, _ := m["priorPassingVersion"].(string); got != "1.27.0" {
		t.Errorf("expected priorPassingVersion=%q, got %q", "1.27.0", got)
	}
	if v, ok := m["grafanaVersion"]; ok {
		t.Errorf("expected no grafanaVersion field on the service axis, got %v", v)
	}
}

func TestEmitDefectConfirmedGrafanaAxis(t *testing.T) {
	d := mkDefect(VersionAxisGrafana, "13.0.0-23542128402")
	d.GrafanaVersion = d.Version
	m := emitOne(t, d)

	if got, _ := m["versionAxis"].(string); got != VersionAxisGrafana {
		t.Errorf("expected versionAxis=%q, got %q", VersionAxisGrafana, got)
	}
	if got, _ := m["grafanaVersion"].(string); got != "13.0.0-23542128402" {
		t.Errorf("expected grafanaVersion=%q, got %q", "13.0.0-23542128402", got)
	}
	if v, ok := m["serviceVersion"]; ok {
		t.Errorf("expected no serviceVersion field on the grafana axis, got %v", v)
	}
}

func TestEmitDefectConfirmedLegacyDefectDefaultsToGrafanaAxis(t *testing.T) {
	// A defect built without the axis fields (any caller predating them) must
	// emit exactly like the grafana axis so existing dashboards keep working.
	d := mkDefect("", "")
	d.Version = ""
	d.GrafanaVersion = "13.0.0-23542128402"
	m := emitOne(t, d)

	if got, _ := m["versionAxis"].(string); got != VersionAxisGrafana {
		t.Errorf("expected versionAxis to default to %q, got %q", VersionAxisGrafana, got)
	}
	if got, _ := m["grafanaVersion"].(string); got != "13.0.0-23542128402" {
		t.Errorf("expected grafanaVersion=%q, got %q", "13.0.0-23542128402", got)
	}
	if v, ok := m["serviceVersion"]; ok {
		t.Errorf("expected no serviceVersion field for a legacy defect, got %v", v)
	}
}
