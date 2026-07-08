package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
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
	// Attempts / MaxAttempts are emitted by bench's LogReporter as of the k6
	// retry-support change (#1005). Attempts is the number of times the test
	// ran (initial run + retries); MaxAttempts is the configured upper bound
	// (1 + configured retries). Records that predate that change — or JSON
	// emitted by an older bench — decode to 0, so any logic that reads them
	// must treat 0 as "no retry data" and never downgrade on its absence.
	Attempts    int `json:"attempts"`
	MaxAttempts int `json:"maxAttempts"`
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

// RetriesEnabled reports whether a real retry budget existed for this run.
// MaxAttempts is 1 when retries are off and 0 on legacy records that predate
// the attempts field, so both collapse to "no retry signal".
func (r TestRunRecord) RetriesEnabled() bool {
	return r.MaxAttempts > 1
}

// RetryExhausted reports whether this failing run burned its full retry budget
// and still failed — the strong, deterministic-defect signal described in
// #961's review. It is only ever true when retries were enabled, so it can add
// confidence but never remove it from records that lack retry data.
func (r TestRunRecord) RetryExhausted() bool {
	return r.Failed() && r.RetriesEnabled() && r.Attempts >= r.MaxAttempts
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
	case "attempts":
		n, err := parseCount(key, val)
		if err != nil {
			return err
		}
		r.Attempts = n
	case "maxAttempts":
		n, err := parseCount(key, val)
		if err != nil {
			return err
		}
		r.MaxAttempts = n
	}
	return nil
}

// parseCount decodes a small non-negative integer logfmt value, tolerating an
// empty value as 0 so a blank field doesn't fail the whole record.
func parseCount(key, val string) (int, error) {
	if val == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("parsing logfmt %s %q: %w", key, val, err)
	}
	return n, nil
}
