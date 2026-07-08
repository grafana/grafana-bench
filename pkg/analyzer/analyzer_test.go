package analyzer

import (
	"testing"
	"time"
)

const (
	svcAthena   = "grafana-athena-datasource"
	stageCI     = "ci"
	testFile    = "permissions.ts"
	versionGood = "13.0.0-23563050832"
	versionBad  = "13.0.0-23542128402"
)

var baseTime = time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)

func mkRun(offset time.Duration, version, status, exitMsg, runID string) TestRunRecord {
	return TestRunRecord{
		Time:           baseTime.Add(offset),
		Msg:            "testRun",
		Service:        svcAthena,
		RunStage:       stageCI,
		TestFile:       testFile,
		Folder:         "tests/folders",
		GrafanaVersion: version,
		GrafanaSlug:    "k6testinstant1",
		GrafanaURL:     "k6testinstant1.grafana-dev.net",
		Status:         status,
		ExitMessage:    exitMsg,
		RunID:          runID,
	}
}

func TestAnalyzeCleanRegression(t *testing.T) {
	records := []TestRunRecord{
		mkRun(0, versionGood, "passed", "", "run-1"),
		mkRun(1*time.Hour, versionGood, "passed", "", "run-2"),
		mkRun(2*time.Hour, versionGood, "passed", "", "run-3"),
		mkRun(3*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=4711", "run-4"),
		mkRun(4*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=9001", "run-5"),
		mkRun(5*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=1234", "run-6"),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected 1 confirmed defect, got %d: %+v", len(defects), defects)
	}
	d := defects[0]
	if d.Confidence != ConfidenceConfirmed {
		t.Errorf("expected confirmed, got %q", d.Confidence)
	}
	if d.GrafanaVersion != versionBad {
		t.Errorf("expected bad version %q, got %q", versionBad, d.GrafanaVersion)
	}
	if d.PriorPassingVersion != versionGood {
		t.Errorf("expected prior version %q, got %q", versionGood, d.PriorPassingVersion)
	}
	if d.ConfidenceRuns != 3 {
		t.Errorf("expected 3 confidence runs, got %d", d.ConfidenceRuns)
	}
	if d.PriorPassingRuns != 3 {
		t.Errorf("expected 3 prior passing runs, got %d", d.PriorPassingRuns)
	}
	if len(d.SourceRunIDs) != 3 {
		t.Errorf("expected 3 source run ids, got %d", len(d.SourceRunIDs))
	}
}

func TestAnalyzeFlakyDoesNotTrigger(t *testing.T) {
	// Interleaved pass/fail on the same version fails rule (1): the tail of
	// the block is not all-failures.
	records := []TestRunRecord{
		mkRun(0, versionGood, "passed", "", "run-1"),
		mkRun(1*time.Hour, versionGood, "passed", "", "run-2"),
		mkRun(2*time.Hour, versionBad, "failed", "flake", "run-3"),
		mkRun(3*time.Hour, versionBad, "passed", "", "run-4"),
		mkRun(4*time.Hour, versionBad, "failed", "flake", "run-5"),
		mkRun(5*time.Hour, versionBad, "passed", "", "run-6"),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 0 {
		t.Fatalf("expected 0 defects for flaky pattern, got %d: %+v", len(defects), defects)
	}
}

func TestAnalyzeSuspectedSignatureDrift(t *testing.T) {
	// Three consecutive failures on the bad version, but each emits a
	// different exit message that doesn't collapse under canonicalization.
	records := []TestRunRecord{
		mkRun(0, versionGood, "passed", "", "run-1"),
		mkRun(1*time.Hour, versionGood, "passed", "", "run-2"),
		mkRun(2*time.Hour, versionBad, "failed", "folder.get failed", "run-3"),
		mkRun(3*time.Hour, versionBad, "failed", "dashboard.get failed", "run-4"),
		mkRun(4*time.Hour, versionBad, "failed", "alert.get failed", "run-5"),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected 1 suspected defect, got %d: %+v", len(defects), defects)
	}
	if defects[0].Confidence != ConfidenceSuspected {
		t.Errorf("expected suspected, got %q", defects[0].Confidence)
	}
}

func TestAnalyzeRollbackNoPriorGreen(t *testing.T) {
	// Prior version also had failures — no clean green to regress from.
	records := []TestRunRecord{
		mkRun(0, versionGood, "failed", "already broken", "run-1"),
		mkRun(1*time.Hour, versionGood, "failed", "already broken", "run-2"),
		mkRun(2*time.Hour, versionBad, "failed", "still broken", "run-3"),
		mkRun(3*time.Hour, versionBad, "failed", "still broken", "run-4"),
		mkRun(4*time.Hour, versionBad, "failed", "still broken", "run-5"),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 0 {
		t.Fatalf("expected 0 defects when there's no green prior version, got %d: %+v", len(defects), defects)
	}
}

func TestAnalyzeRunStageScoping(t *testing.T) {
	// CI stream shows a clean regression. Nightly stream is all-failing
	// across all versions — must not leak into the CI analysis.
	mk := func(offset time.Duration, stage, version, status, msg, id string) TestRunRecord {
		r := mkRun(offset, version, status, msg, id)
		r.RunStage = stage
		return r
	}
	records := []TestRunRecord{
		mk(0, "ci", versionGood, "passed", "", "run-ci-1"),
		mk(1*time.Hour, "ci", versionGood, "passed", "", "run-ci-2"),
		mk(2*time.Hour, "ci", versionBad, "failed", "sig", "run-ci-3"),
		mk(3*time.Hour, "ci", versionBad, "failed", "sig", "run-ci-4"),
		mk(4*time.Hour, "ci", versionBad, "failed", "sig", "run-ci-5"),

		mk(0, "nightly", versionGood, "failed", "infra", "run-n-1"),
		mk(1*time.Hour, "nightly", versionGood, "failed", "infra", "run-n-2"),
		mk(2*time.Hour, "nightly", versionBad, "failed", "infra", "run-n-3"),
		mk(3*time.Hour, "nightly", versionBad, "failed", "infra", "run-n-4"),
		mk(4*time.Hour, "nightly", versionBad, "failed", "infra", "run-n-5"),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected exactly 1 defect (from CI stream only), got %d: %+v", len(defects), defects)
	}
	if defects[0].RunStage != "ci" {
		t.Errorf("expected defect from ci stage, got %q", defects[0].RunStage)
	}
}

// mkRetryRun is mkRun plus the retry-budget fields added in #1005.
func mkRetryRun(offset time.Duration, version, status, exitMsg, runID string, attempts, maxAttempts int) TestRunRecord {
	r := mkRun(offset, version, status, exitMsg, runID)
	r.Attempts = attempts
	r.MaxAttempts = maxAttempts
	return r
}

func TestAnalyzeRetryExhaustedStaysConfirmed(t *testing.T) {
	// Retries enabled (maxAttempts=3) and every failure burned the full budget:
	// stable signature + exhausted budget => confirmed, retryExhausted=true.
	records := []TestRunRecord{
		mkRetryRun(0, versionGood, "passed", "", "run-1", 1, 3),
		mkRetryRun(1*time.Hour, versionGood, "passed", "", "run-2", 1, 3),
		mkRetryRun(2*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=4711", "run-3", 3, 3),
		mkRetryRun(3*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=9001", "run-4", 3, 3),
		mkRetryRun(4*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=1234", "run-5", 3, 3),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected 1 defect, got %d: %+v", len(defects), defects)
	}
	if defects[0].Confidence != ConfidenceConfirmed {
		t.Errorf("expected confirmed, got %q", defects[0].Confidence)
	}
	if !defects[0].RetryExhausted {
		t.Errorf("expected retryExhausted=true when the whole tail burned its budget")
	}
}

func TestAnalyzeRetryNotExhaustedDowngradesToSuspected(t *testing.T) {
	// Signature is stable, but one tail failure gave up before spending the
	// retry budget (attempts=1 < maxAttempts=3). The failure isn't reliably
	// reproducing through the full budget, so confidence drops to suspected.
	records := []TestRunRecord{
		mkRetryRun(0, versionGood, "passed", "", "run-1", 1, 3),
		mkRetryRun(1*time.Hour, versionGood, "passed", "", "run-2", 1, 3),
		mkRetryRun(2*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=4711", "run-3", 3, 3),
		mkRetryRun(3*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=9001", "run-4", 1, 3),
		mkRetryRun(4*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=1234", "run-5", 3, 3),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected 1 defect, got %d: %+v", len(defects), defects)
	}
	if defects[0].Confidence != ConfidenceSuspected {
		t.Errorf("expected suspected when a tail failure didn't exhaust its retry budget, got %q", defects[0].Confidence)
	}
	if defects[0].RetryExhausted {
		t.Errorf("expected retryExhausted=false when a tail failure left budget unspent")
	}
}

func TestAnalyzeRetriesOffPreservesV1Confidence(t *testing.T) {
	// Retries off (maxAttempts=1) is indistinguishable from legacy records with
	// no retry data: the retry gate must not fire, so a stable-signature
	// regression stays confirmed exactly as it did under v1.
	records := []TestRunRecord{
		mkRetryRun(0, versionGood, "passed", "", "run-1", 1, 1),
		mkRetryRun(1*time.Hour, versionGood, "passed", "", "run-2", 1, 1),
		mkRetryRun(2*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=4711", "run-3", 1, 1),
		mkRetryRun(3*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=9001", "run-4", 1, 1),
		mkRetryRun(4*time.Hour, versionBad, "failed", "check rate expected 1 got 0.87 pid=1234", "run-5", 1, 1),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected 1 defect, got %d: %+v", len(defects), defects)
	}
	if defects[0].Confidence != ConfidenceConfirmed {
		t.Errorf("expected confirmed with retries off, got %q", defects[0].Confidence)
	}
	if defects[0].RetryExhausted {
		t.Errorf("expected retryExhausted=false when retries are off (no real budget)")
	}
}

func TestAnalyzeSignatureDriftNotRescuedByRetries(t *testing.T) {
	// Exhausting the retry budget can't rescue a drifting signature: rule (3)
	// still downgrades to suspected even though retryExhausted is true.
	records := []TestRunRecord{
		mkRetryRun(0, versionGood, "passed", "", "run-1", 1, 3),
		mkRetryRun(1*time.Hour, versionGood, "passed", "", "run-2", 1, 3),
		mkRetryRun(2*time.Hour, versionBad, "failed", "folder.get failed", "run-3", 3, 3),
		mkRetryRun(3*time.Hour, versionBad, "failed", "dashboard.get failed", "run-4", 3, 3),
		mkRetryRun(4*time.Hour, versionBad, "failed", "alert.get failed", "run-5", 3, 3),
	}

	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 1 {
		t.Fatalf("expected 1 defect, got %d: %+v", len(defects), defects)
	}
	if defects[0].Confidence != ConfidenceSuspected {
		t.Errorf("expected suspected on signature drift even with exhausted retries, got %q", defects[0].Confidence)
	}
	if !defects[0].RetryExhausted {
		t.Errorf("expected retryExhausted=true — the tail did burn its budget even though the signature drifted")
	}
}

func TestAnalyzeInsufficientPersistence(t *testing.T) {
	// Only 2 consecutive failures — under the default MinFailures=3.
	records := []TestRunRecord{
		mkRun(0, versionGood, "passed", "", "run-1"),
		mkRun(1*time.Hour, versionGood, "passed", "", "run-2"),
		mkRun(2*time.Hour, versionBad, "failed", "sig", "run-3"),
		mkRun(3*time.Hour, versionBad, "failed", "sig", "run-4"),
	}
	defects := Analyze(records, RuleConfig{}, 24*time.Hour, baseTime)
	if len(defects) != 0 {
		t.Fatalf("expected 0 defects (only 2 failures), got %d", len(defects))
	}
}
