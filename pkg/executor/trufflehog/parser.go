package trufflehog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/metrics"
)

var (
	ErrInvalidFormat = errors.New("invalid trufflehog JSON format")
)

// Finding represents a single TruffleHog finding (native JSON format).
// See: https://docs.trufflesecurity.com/docs/trufflehog/api
//
// IMPORTANT: Do not expose Raw, RawV2, Redacted, or ExtraData in report output,
// metrics, or logs—they may contain secrets. Only use DetectorName, DetectorDescription
// (generic text), source file/line, and Verified.
type Finding struct {
	SourceMetadata       SourceMetadata `json:"SourceMetadata"`
	SourceID             int64          `json:"SourceID"`
	SourceType           int            `json:"SourceType"`
	SourceName           string         `json:"SourceName"`
	DetectorType         int            `json:"DetectorType"`
	DetectorName         string         `json:"DetectorName"`
	DetectorDescription  string         `json:"DetectorDescription"`
	DecoderName          string         `json:"DecoderName"`
	Verified             bool           `json:"Verified"`
	VerificationFromCache bool          `json:"VerificationFromCache"`
	Raw                  string         `json:"Raw"`                  // Do not send to report/metrics/logs
	RawV2                string         `json:"RawV2"`                // Do not send to report/metrics/logs
	Redacted             string         `json:"Redacted"`              // Do not send to report/metrics/logs
	ExtraData            map[string]any `json:"ExtraData"`            // May contain secrets; do not send
	StructuredData       any            `json:"StructuredData"`
}

type SourceMetadata struct {
	Data SourceData `json:"Data"`
}

// SourceData can be Filesystem, Git, GitHub, etc. We only need to extract file/line when present.
type SourceData struct {
	Filesystem *FilesystemSource `json:"Filesystem,omitempty"`
	Git        *GitSource       `json:"Git,omitempty"`
	GitHub     *GitHubSource     `json:"GitHub,omitempty"`
}

type FilesystemSource struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type GitSource struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type GitHubSource struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// ParseFindings parses TruffleHog native JSON output (array of findings)
// and returns a SuiteRunSummary suitable for grafana-bench report.
func ParseFindings(report io.Reader) (executor.SuiteRunSummary, error) {
	var findings []Finding
	decoder := json.NewDecoder(report)
	if err := decoder.Decode(&findings); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("decoding trufflehog JSON: %w", err)
	}

	// TruffleHog outputs an array; empty array is valid (no findings)
	summary := executor.SuiteRunSummary{
		StartTime: time.Now(),
		Status:    executor.SuitePassed,
		SuiteName: "trufflehog",
	}

	if summary.Attributes == nil {
		summary.Attributes = make(map[string]string)
	}

	verifiedCount := 0
	unverifiedCount := 0
	detectorCounts := make(map[string]int)
	// track per-detector counts split by verified status for fine-grained metrics
	type detectorKey struct {
		name     string
		verified bool
	}
	detectorVerifiedCounts := make(map[detectorKey]int)

	for _, f := range findings {
		if f.Verified {
			verifiedCount++
		} else {
			unverifiedCount++
		}
		detectorCounts[f.DetectorName]++
		detectorVerifiedCounts[detectorKey{name: f.DetectorName, verified: f.Verified}]++

		testFile := "unknown"
		lineInfo := ""
		if f.SourceMetadata.Data.Filesystem != nil {
			testFile = f.SourceMetadata.Data.Filesystem.File
			if f.SourceMetadata.Data.Filesystem.Line > 0 {
				lineInfo = fmt.Sprintf(" (line %d)", f.SourceMetadata.Data.Filesystem.Line)
			}
		} else if f.SourceMetadata.Data.Git != nil {
			testFile = f.SourceMetadata.Data.Git.File
			if f.SourceMetadata.Data.Git.Line > 0 {
				lineInfo = fmt.Sprintf(" (line %d)", f.SourceMetadata.Data.Git.Line)
			}
		} else if f.SourceMetadata.Data.GitHub != nil {
			testFile = f.SourceMetadata.Data.GitHub.File
			if f.SourceMetadata.Data.GitHub.Line > 0 {
				lineInfo = fmt.Sprintf(" (line %d)", f.SourceMetadata.Data.GitHub.Line)
			}
		}

		exitMessage := f.DetectorName
		if f.DetectorDescription != "" {
			exitMessage = f.DetectorDescription
		}
		exitMessage += lineInfo

		// Only safe, non-secret fields: detector name, verified, file, line. Never Raw/RawV2/Redacted.
		// Selectively extract non-secret ExtraData fields: username, name, url.
		attrs := map[string]string{
			"detector": f.DetectorName,
			"verified": strconv.FormatBool(f.Verified),
		}
		if v, ok := f.ExtraData["username"].(string); ok && v != "" {
			attrs["username"] = v
		}
		if v, ok := f.ExtraData["name"].(string); ok && v != "" {
			attrs["name"] = v
		}
		if v, ok := f.ExtraData["url"].(string); ok && v != "" {
			attrs["url"] = v
		}

		// Verified findings are failures (critical); unverified are informational (like zizmor's note level).
		var status executor.TestStatus
		if f.Verified {
			status = executor.TestFailed
			summary.Status = executor.SuiteFailed
		} else {
			status = executor.TestPassed
		}

		testRun := executor.TestRunSummary{
			TestFolder:  "",
			TestFile:    testFile,
			StartTime:   time.Now(),
			Status:      status,
			ExitMessage: exitMessage,
			Attributes:  attrs,
		}

		summary.TestsExecuted++
		switch status {
		case executor.TestPassed:
			summary.TestsPassed++
		case executor.TestFailed:
			summary.TestsFailed++
		case executor.TestError:
			summary.TestsError++
		}
		summary.TestRuns = append(summary.TestRuns, testRun)
	}

	summary.Attributes["total_findings"] = strconv.Itoa(len(findings))
	summary.Attributes["verified"] = strconv.Itoa(verifiedCount)
	summary.Attributes["unverified"] = strconv.Itoa(unverifiedCount)
	summary.Attributes["unique_detectors"] = strconv.Itoa(len(detectorCounts))

	timestamp := summary.StartTime.UnixMilli()
	summary.Metrics = []metrics.Metric{
		{
			Name:      "trufflehog_total_findings",
			Value:     float64(len(findings)),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
		{
			Name:      "trufflehog_verified",
			Value:     float64(verifiedCount),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
		{
			Name:      "trufflehog_unverified",
			Value:     float64(unverifiedCount),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
		{
			Name:      "trufflehog_unique_detectors",
			Value:     float64(len(detectorCounts)),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
	}

	for key, count := range detectorVerifiedCounts {
		summary.Metrics = append(summary.Metrics, metrics.Metric{
			Name:  "trufflehog_detector_findings",
			Value: float64(count),
			Labels: map[string]string{
				"detector": key.name,
				"verified": strconv.FormatBool(key.verified),
			},
			Timestamp: timestamp,
		})
	}

	summary.Metrics = append(summary.Metrics, metrics.Metric{
		Name:      "trufflehog_scan_completed",
		Value:     1.0,
		Labels:    map[string]string{},
		Timestamp: timestamp,
	})

	return summary, nil
}
