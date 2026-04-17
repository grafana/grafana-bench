package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logfmt/logfmt"
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

// decodeRecord parses a single log line from Loki into a TestRunRecord.
// It sniffs the first non-whitespace byte to decide between JSON (`{`) and
// logfmt (everything else). bench's LogReporter emits logfmt when
// --report-output=log and JSON when --report-output=json; both shapes land
// in Loki and both need to be decodable here.
func decodeRecord(line []byte) (TestRunRecord, error) {
	trimmed := bytes.TrimLeft(line, " \t")
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var rec TestRunRecord
		if err := json.Unmarshal(trimmed, &rec); err != nil {
			return TestRunRecord{}, fmt.Errorf("decoding testRun record as JSON: %w", err)
		}
		return rec, nil
	}
	return decodeLogfmt(trimmed)
}

// decodeLogfmt parses a single logfmt line (as produced by slog's text
// handler) into a TestRunRecord. Unknown keys are silently ignored so new
// fields added to bench's LogReporter don't break the analyzer.
func decodeLogfmt(line []byte) (TestRunRecord, error) {
	var rec TestRunRecord
	dec := logfmt.NewDecoder(bytes.NewReader(line))
	if !dec.ScanRecord() {
		if err := dec.Err(); err != nil {
			return TestRunRecord{}, fmt.Errorf("decoding testRun record as logfmt: %w", err)
		}
		return rec, nil
	}
	for dec.ScanKeyval() {
		key := string(dec.Key())
		val := string(dec.Value())
		if err := rec.setLogfmtField(key, val); err != nil {
			return TestRunRecord{}, err
		}
	}
	if err := dec.Err(); err != nil {
		return TestRunRecord{}, fmt.Errorf("decoding testRun record as logfmt: %w", err)
	}
	return rec, nil
}

// setLogfmtField assigns a single key/value pair onto the record. Unknown
// keys are ignored; the only errors returned are for malformed typed fields.
func (r *TestRunRecord) setLogfmtField(key, val string) error {
	switch key {
	case "time":
		if val == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339Nano, val)
		if err != nil {
			// slog's text handler emits an unquoted RFC3339 with no subsecond
			// precision; try that too before giving up.
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				return fmt.Errorf("parsing logfmt time %q: %w", val, err)
			}
		}
		r.Time = t
	case "level":
		r.Level = val
	case "msg":
		r.Msg = val
	case "service":
		r.Service = val
	case "runId":
		r.RunID = val
	case "runStage":
		r.RunStage = val
	case "suiteName":
		r.SuiteName = val
	case "suiteRevision":
		r.SuiteRevision = val
	case "suiteRun":
		r.SuiteRun = val
	case "testExecutor":
		r.TestExecutor = val
	case "testTrigger":
		r.TestTrigger = val
	case "benchRevision":
		r.BenchRevision = val
	case "serviceUrl":
		r.ServiceURL = val
	case "serviceVersion":
		r.ServiceVersion = val
	case "grafanaUrl":
		r.GrafanaURL = val
	case "grafanaSlug":
		r.GrafanaSlug = val
	case "grafanaVersion":
		r.GrafanaVersion = val
	case "folder":
		r.Folder = val
	case "testFile":
		r.TestFile = val
	case "status":
		r.Status = val
	case "exitMessage":
		r.ExitMessage = val
	case "order":
		r.Order = val
	case "testRun":
		r.TestRun = val
	}
	return nil
}
