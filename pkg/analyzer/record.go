package analyzer

import (
	"encoding/json"
	"fmt"
	"time"
)

// TestRunRecord is a lenient decode target for msg=testRun events pulled from
// Loki. It intentionally omits the validator tags used by
// pkg/reporter.TestRunLine so that we tolerate missing/optional fields and new
// fields added by the reporter without breaking the analyzer.
type TestRunRecord struct {
	Time           time.Time `json:"time"`
	Level          string    `json:"level"`
	Msg            string    `json:"msg"`
	Service        string    `json:"service"`
	RunID          string    `json:"runId"`
	RunStage       string    `json:"runStage"`
	SuiteName      string    `json:"suiteName"`
	SuiteRevision  string    `json:"suiteRevision"`
	SuiteRun       string    `json:"suiteRun"`
	TestExecutor   string    `json:"testExecutor"`
	TestTrigger    string    `json:"testTrigger"`
	BenchRevision  string    `json:"benchRevision"`
	ServiceURL     string    `json:"serviceUrl"`
	ServiceVersion string    `json:"serviceVersion"`
	GrafanaURL     string    `json:"grafanaUrl"`
	GrafanaSlug    string    `json:"grafanaSlug"`
	GrafanaVersion string    `json:"grafanaVersion"`
	Folder         string    `json:"folder"`
	TestFile       string    `json:"testFile"`
	Status         string    `json:"status"`
	ExitMessage    string    `json:"exitMessage"`
	Order          string    `json:"order"`
	TestRun        string    `json:"testRun"`
}

// Failed reports whether this record counts as a failure for rule evaluation.
// Flaky/skipped do not count as failures; status="error" does (infra/exec
// errors still prevent a version shipping clean).
func (r TestRunRecord) Failed() bool {
	return r.Status == "failed" || r.Status == "error"
}

// Passed reports whether this record counts as a pass for the regression
// boundary rule.
func (r TestRunRecord) Passed() bool {
	return r.Status == "passed"
}

// GroupKey identifies the unit of regression analysis.
type GroupKey struct {
	Service        string
	RunStage       string
	TestFile       string
	GrafanaVersion string
}

func (k GroupKey) String() string {
	return fmt.Sprintf("%s|%s|%s|%s", k.Service, k.RunStage, k.TestFile, k.GrafanaVersion)
}

// decodeRecord parses a single JSON log line from Loki into a TestRunRecord.
func decodeRecord(line []byte) (TestRunRecord, error) {
	var rec TestRunRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return TestRunRecord{}, fmt.Errorf("decoding testRun record: %w", err)
	}
	return rec, nil
}
