package analyze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// TestAnalyzeCommandReplaysPermissionsIncident is the end-to-end acceptance
// test from the plan: it feeds the analyzer a fixture stream that replays
// the 2026-03-26 permissions.ts RRC regression and asserts the exact
// msg=defectConfirmed event that should fire.
//
// The fixture has:
//   - 3 green runs on the last-known-good build 23563050832
//   - 4 failing runs on the first-bad build 23542128402, all with the same
//     canonicalised exit message
//
// Expected output: exactly one confirmed defect with the bad version as
// grafanaVersion and the good version as priorPassingVersion.
func TestAnalyzeCommandReplaysPermissionsIncident(t *testing.T) {
	const (
		svc            = "grafana-pro"
		runStage       = "ci"
		testFile       = "permissions.ts"
		goodVersion    = "13.0.0-23563050832"
		badVersion     = "13.0.0-23542128402"
		expectSig      = "" // set after we see the real one — assertion is inequality against empty
		goodSlug       = "k6testinstant1"
		goodURL        = "k6testinstant1.grafana-dev.net"
	)
	_ = expectSig // unused — kept as a reminder of what to add if signature stability is ever asserted

	// Anchor fixture to recent time so the analyzer's default 24h window covers it.
	start := time.Now().Add(-6 * time.Hour)
	fixture := []map[string]any{}

	// Green runs on the last-known-good build.
	for i := 0; i < 3; i++ {
		fixture = append(fixture, map[string]any{
			"time":           start.Add(time.Duration(i) * 10 * time.Minute).Format(time.RFC3339Nano),
			"level":          "INFO",
			"msg":            "testRun",
			"service":        svc,
			"runStage":       runStage,
			"testFile":       testFile,
			"folder":         "rrc-grafana-api-tests",
			"grafanaVersion": goodVersion,
			"grafanaSlug":    goodSlug,
			"grafanaUrl":     goodURL,
			"status":         "passed",
			"exitMessage":    "",
			"runId":          fmt.Sprintf("run-good-%d", i),
		})
	}

	// Failing runs on the first-bad build. Vary the pid so canonicalization
	// has to do its job for the signature to stabilize.
	for i := 0; i < 4; i++ {
		fixture = append(fixture, map[string]any{
			"time":           start.Add(time.Duration(i+3) * 10 * time.Minute).Format(time.RFC3339Nano),
			"level":          "INFO",
			"msg":            "testRun",
			"service":        svc,
			"runStage":       runStage,
			"testFile":       testFile,
			"folder":         "rrc-grafana-api-tests",
			"grafanaVersion": badVersion,
			"grafanaSlug":    goodSlug,
			"grafanaUrl":     goodURL,
			"status":         "failed",
			"exitMessage":    fmt.Sprintf("check rate expected 1 got 0.87 pid=%d at %s", 1000+i, start.Format(time.RFC3339)),
			"runId":          fmt.Sprintf("run-bad-%d", i),
		})
	}

	srv := startFakeLoki(t, fixture)
	defer srv.Close()

	var stdout bytes.Buffer
	cmd := NewCmd(slog.New(slog.NewTextHandler(io.Discard, nil)))
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"--analyze-loki-url", srv.URL,
		"--analyze-service", svc,
		"--analyze-run-stage", runStage,
		"--analyze-window", "24h",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("analyze command failed: %v", err)
	}

	events := parseJSONL(t, stdout.Bytes())
	if len(events) != 1 {
		t.Fatalf("expected 1 defectConfirmed event, got %d:\n%s", len(events), stdout.String())
	}
	d := events[0]

	assertString(t, d, "msg", "defectConfirmed")
	assertString(t, d, "tool", "bench")
	assertString(t, d, "service", svc)
	assertString(t, d, "runStage", runStage)
	assertString(t, d, "testFile", testFile)
	assertString(t, d, "grafanaVersion", badVersion)
	assertString(t, d, "priorPassingVersion", goodVersion)
	assertString(t, d, "confidence", "confirmed")
	assertString(t, d, "ruleVersion", "v1")
	assertNumber(t, d, "confidenceRuns", 3)
	assertNumber(t, d, "priorPassingRuns", 3)
	if sig, _ := d["signatureHash"].(string); len(sig) != 12 {
		t.Errorf("expected 12-char signatureHash, got %q", sig)
	}
}

// TestAnalyzeCommandDryRunSuppressesEmission verifies that --analyze-emit=false
// leaves stdout empty even when a defect would otherwise fire.
func TestAnalyzeCommandDryRunSuppressesEmission(t *testing.T) {
	start := time.Now().Add(-6 * time.Hour)
	fixture := []map[string]any{
		mkTestRun(start, "grafana-pro", "ci", "permissions.ts", "13.0.0-good", "passed", "", "g-1"),
		mkTestRun(start.Add(10*time.Minute), "grafana-pro", "ci", "permissions.ts", "13.0.0-good", "passed", "", "g-2"),
		mkTestRun(start.Add(20*time.Minute), "grafana-pro", "ci", "permissions.ts", "13.0.0-bad", "failed", "boom", "b-1"),
		mkTestRun(start.Add(30*time.Minute), "grafana-pro", "ci", "permissions.ts", "13.0.0-bad", "failed", "boom", "b-2"),
		mkTestRun(start.Add(40*time.Minute), "grafana-pro", "ci", "permissions.ts", "13.0.0-bad", "failed", "boom", "b-3"),
	}

	srv := startFakeLoki(t, fixture)
	defer srv.Close()

	var stdout bytes.Buffer
	cmd := NewCmd(slog.New(slog.NewTextHandler(io.Discard, nil)))
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"--analyze-loki-url", srv.URL,
		"--analyze-service", "grafana-pro",
		"--analyze-emit=false",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("analyze command failed: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout in dry-run mode, got:\n%s", stdout.String())
	}
}

// TestAnalyzeCommandRejectsMissingRequiredFlags verifies validation.
func TestAnalyzeCommandRejectsMissingRequiredFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing loki url",
			args: []string{"--analyze-service", "grafana-pro"},
		},
		{
			name: "missing service",
			args: []string{"--analyze-loki-url", "http://localhost:3100"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCmd(slog.New(slog.NewTextHandler(io.Discard, nil)))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.ExecuteContext(context.Background())
			if err == nil {
				t.Fatalf("expected validation error for %s, got nil", tc.name)
			}
		})
	}
}

// startFakeLoki spins up an httptest server that answers /loki/api/v1/query_range
// with records from the supplied fixture whose timestamps fall inside the
// requested window. This mimics how a real Loki would slice records across
// the analyzer's chunked queries.
func startFakeLoki(t *testing.T, records []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		startNs, _ := strconv.ParseInt(q.Get("start"), 10, 64)
		endNs, _ := strconv.ParseInt(q.Get("end"), 10, 64)

		var values [][]string
		for _, rec := range records {
			ts, err := time.Parse(time.RFC3339Nano, rec["time"].(string))
			if err != nil {
				t.Errorf("fixture time parse: %v", err)
				continue
			}
			ns := ts.UnixNano()
			if ns < startNs || ns > endNs {
				continue
			}
			line, _ := json.Marshal(rec)
			values = append(values, []string{strconv.FormatInt(ns, 10), string(line)})
		}
		body := map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "streams",
				"result": []map[string]any{{
					"stream": map[string]string{"service_name": "grafana-pro"},
					"values": values,
				}},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func mkTestRun(ts time.Time, svc, stage, file, version, status, exitMsg, runID string) map[string]any {
	return map[string]any{
		"time":           ts.Format(time.RFC3339Nano),
		"level":          "INFO",
		"msg":            "testRun",
		"service":        svc,
		"runStage":       stage,
		"testFile":       file,
		"folder":         "tests",
		"grafanaVersion": version,
		"grafanaSlug":    "k6testinstant1",
		"grafanaUrl":     "k6testinstant1.grafana-dev.net",
		"status":         status,
		"exitMessage":    exitMsg,
		"runId":          runID,
	}
}

func parseJSONL(t *testing.T, b []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	for i, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("line %d not valid JSON: %v\nline: %s", i, err, line)
		}
		out = append(out, m)
	}
	return out
}

func assertString(t *testing.T, m map[string]any, key, want string) {
	t.Helper()
	got, ok := m[key].(string)
	if !ok {
		t.Errorf("expected field %q to be a string, got %T", key, m[key])
		return
	}
	if got != want {
		t.Errorf("expected %s=%q, got %q", key, want, got)
	}
}

func assertNumber(t *testing.T, m map[string]any, key string, want int) {
	t.Helper()
	got, ok := m[key].(float64)
	if !ok {
		t.Errorf("expected field %q to be a number, got %T", key, m[key])
		return
	}
	if int(got) != want {
		t.Errorf("expected %s=%d, got %v", key, want, got)
	}
}
