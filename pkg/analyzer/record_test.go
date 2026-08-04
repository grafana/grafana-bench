package analyzer

import (
	"testing"
	"time"
)

func TestDecodeRecordJSON(t *testing.T) {
	line := []byte(`{"time":"2026-04-17T10:00:00Z","level":"INFO","msg":"testRun","tool":"bench","service":"grafana-pro","runStage":"ci","testFile":"permissions.ts","grafanaVersion":"13.0.0-1","status":"passed","runId":"r-1"}`)
	rec, err := decodeRecord(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertRec(t, rec, "grafana-pro", "ci", "permissions.ts", "13.0.0-1", "passed", "r-1")
	if !rec.Time.Equal(time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected time: %v", rec.Time)
	}
}

// TestDecodeRecordLogfmt exercises the logfmt path — the format bench actually
// emits in prod when --report-output=log. The input mimics slog's text handler
// output, including a quoted exitMessage containing spaces.
func TestDecodeRecordLogfmt(t *testing.T) {
	line := []byte(`time=2026-04-17T10:00:00.123456789Z level=INFO msg=testRun tool=bench service=grafana-pro runStage=ci testFile=permissions.ts grafanaVersion=13.0.0-1 status=failed exitMessage="check rate expected 1 got 0.87 pid=1001" runId=r-1`)
	rec, err := decodeRecord(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertRec(t, rec, "grafana-pro", "ci", "permissions.ts", "13.0.0-1", "failed", "r-1")
	if rec.ExitMessage != "check rate expected 1 got 0.87 pid=1001" {
		t.Errorf("exitMessage mismatch: %q", rec.ExitMessage)
	}
	expected := time.Date(2026, 4, 17, 10, 0, 0, 123456789, time.UTC)
	if !rec.Time.Equal(expected) {
		t.Errorf("unexpected time: got %v want %v", rec.Time, expected)
	}
}

// TestDecodeRecordLogfmtTolerant confirms unknown/new fields don't fail the
// decoder — bench can grow the schema without breaking the analyzer.
func TestDecodeRecordLogfmtTolerant(t *testing.T) {
	line := []byte(`time=2026-04-17T10:00:00Z level=INFO msg=testRun service=grafana-pro futureField=xyz grafanaVersion=13.0.0-1 testFile=p.ts status=passed runId=r-1 runStage=ci`)
	rec, err := decodeRecord(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertRec(t, rec, "grafana-pro", "ci", "p.ts", "13.0.0-1", "passed", "r-1")
}

// TestDecodeRecordLogfmtLeadingWhitespace — Loki occasionally returns lines
// with a leading space (depends on the pipeline stage). The sniff must
// trim before checking the JSON/logfmt first byte.
func TestDecodeRecordLogfmtLeadingWhitespace(t *testing.T) {
	line := []byte("   time=2026-04-17T10:00:00Z level=INFO msg=testRun service=grafana-pro grafanaVersion=v1 testFile=p.ts status=passed runStage=ci runId=r-1")
	rec, err := decodeRecord(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Service != "grafana-pro" || rec.Status != "passed" {
		t.Errorf("unexpected record: %+v", rec)
	}
}

// TestDecodeRecordRetryFields covers the attempts/maxAttempts fields added by
// #1005, in both logfmt and JSON, plus the RetryExhausted / RetriesEnabled
// helpers that feed the v2 confidence rule.
func TestDecodeRecordRetryFields(t *testing.T) {
	logfmtLine := []byte(`time=2026-04-17T10:00:00Z level=INFO msg=testRun service=grafana-pro runStage=ci testFile=p.ts grafanaVersion=v1 status=failed attempts=3 maxAttempts=3 runId=r-1`)
	rec, err := decodeRecord(logfmtLine)
	if err != nil {
		t.Fatalf("decode logfmt: %v", err)
	}
	if rec.Attempts != 3 || rec.MaxAttempts != 3 {
		t.Errorf("logfmt attempts: got %d/%d want 3/3", rec.Attempts, rec.MaxAttempts)
	}
	if !rec.RetriesEnabled() {
		t.Errorf("expected RetriesEnabled with maxAttempts=3")
	}
	if !rec.RetryExhausted() {
		t.Errorf("expected RetryExhausted with failed attempts=3 maxAttempts=3")
	}

	jsonLine := []byte(`{"time":"2026-04-17T10:00:00Z","msg":"testRun","service":"grafana-pro","runStage":"ci","testFile":"p.ts","grafanaVersion":"v1","status":"failed","attempts":1,"maxAttempts":3,"runId":"r-1"}`)
	rec, err = decodeRecord(jsonLine)
	if err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if rec.Attempts != 1 || rec.MaxAttempts != 3 {
		t.Errorf("json attempts: got %d/%d want 1/3", rec.Attempts, rec.MaxAttempts)
	}
	if rec.RetryExhausted() {
		t.Errorf("attempts=1 < maxAttempts=3 must not count as exhausted")
	}
}

// TestRetryHelpersLegacyRecords confirms the helpers treat missing retry data
// (attempts/maxAttempts == 0) as "no retry signal" so they can never downgrade
// a record that predates #1005.
func TestRetryHelpersLegacyRecords(t *testing.T) {
	rec := TestRunRecord{Status: "failed"} // no attempts/maxAttempts
	if rec.RetriesEnabled() {
		t.Errorf("legacy record must report RetriesEnabled=false")
	}
	if rec.RetryExhausted() {
		t.Errorf("legacy record must report RetryExhausted=false")
	}
	// Retries off (maxAttempts=1) behaves the same as legacy.
	rec.Attempts, rec.MaxAttempts = 1, 1
	if rec.RetriesEnabled() || rec.RetryExhausted() {
		t.Errorf("maxAttempts=1 (retries off) must report both helpers false")
	}
}

func assertRec(t *testing.T, rec TestRunRecord, service, stage, file, version, status, runID string) {
	t.Helper()
	if rec.Service != service {
		t.Errorf("service: got %q want %q", rec.Service, service)
	}
	if rec.RunStage != stage {
		t.Errorf("runStage: got %q want %q", rec.RunStage, stage)
	}
	if rec.TestFile != file {
		t.Errorf("testFile: got %q want %q", rec.TestFile, file)
	}
	if rec.GrafanaVersion != version {
		t.Errorf("grafanaVersion: got %q want %q", rec.GrafanaVersion, version)
	}
	if rec.Status != status {
		t.Errorf("status: got %q want %q", rec.Status, status)
	}
	if rec.RunID != runID {
		t.Errorf("runId: got %q want %q", rec.RunID, runID)
	}
}
