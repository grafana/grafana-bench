package zizmor

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
	ErrInvalidFormat = errors.New("invalid SARIF format")
)

// SARIF represents the SARIF 2.1.0 format
// Reference: https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html
type SARIF struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool        Tool         `json:"tool"`
	Invocations []Invocation `json:"invocations"`
	Results     []Result     `json:"results"`
}

type Tool struct {
	Driver Driver `json:"driver"`
}

type Driver struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	SemanticVersion string `json:"semanticVersion"`
	InformationURI  string `json:"informationUri"`
	DownloadURI     string `json:"downloadUri"`
	Rules           []Rule `json:"rules"`
}

type Rule struct {
	ID               string           `json:"id"`
	ShortDescription MessageString    `json:"shortDescription"`
	FullDescription  MessageString    `json:"fullDescription"`
	Help             MessageString    `json:"help"`
	Properties       RuleProperties   `json:"properties"`
	DefaultConfig    DefaultConfig    `json:"defaultConfiguration"`
}

type MessageString struct {
	Text string `json:"text"`
}

type RuleProperties struct {
	Tags []string `json:"tags"`
}

type DefaultConfig struct {
	Level string `json:"level"`
}

type Invocation struct {
	ExecutionSuccessful bool `json:"executionSuccessful"`
}

type Result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   Message    `json:"message"`
	Locations []Location `json:"locations"`
}

type Message struct {
	Text string `json:"text"`
}

type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           Region           `json:"region"`
}

type ArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type Region struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

// ParseSARIF parses SARIF format output from zizmor
func ParseSARIF(report io.Reader) (executor.SuiteRunSummary, error) {
	var sarif SARIF
	decoder := json.NewDecoder(report)
	if err := decoder.Decode(&sarif); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("decoding SARIF: %w", err)
	}

	if len(sarif.Runs) == 0 {
		return executor.SuiteRunSummary{}, fmt.Errorf("%w: no runs found in SARIF output", ErrInvalidFormat)
	}

	summary := executor.SuiteRunSummary{
		StartTime: time.Now(),
		Status:    executor.SuitePassed,
	}

	run := sarif.Runs[0]
	summary.SuiteName = run.Tool.Driver.Name

	// Track severity counts
	severityCounts := map[string]int{
		"error":   0,
		"warning": 0,
		"note":    0,
		"none":    0,
	}

	// Track violation counts per rule
	rulesViolated := make(map[string]int)

	// Process each result as a test
	for _, result := range run.Results {
		level := result.Level
		if level == "" {
			level = "warning" // SARIF default
		}

		// Track severity
		severityCounts[level]++

		// Track rule
		rulesViolated[result.RuleID]++

		// Determine test status based on severity
		var status executor.TestStatus
		switch level {
		case "error":
			status = executor.TestFailed
			summary.Status = executor.SuiteFailed
		case "warning":
			status = executor.TestFailed
			summary.Status = executor.SuiteFailed
		case "note":
			status = executor.TestPassed // Notes are informational
		default:
			status = executor.TestPassed
		}

		// Get file location
		testFile := "unknown"
		testFolder := ""
		exitMessage := result.Message.Text
		if len(result.Locations) > 0 {
			loc := result.Locations[0].PhysicalLocation
			testFile = loc.ArtifactLocation.URI
			if loc.Region.StartLine > 0 {
				exitMessage = fmt.Sprintf("%s (line %d)", result.Message.Text, loc.Region.StartLine)
			}
		}

		testRun := executor.TestRunSummary{
			TestFolder:  testFolder,
			TestFile:    testFile,
			StartTime:   time.Now(),
			Status:      status,
			ExitMessage: exitMessage,
			Attributes: map[string]string{
				"ruleId":   result.RuleID,
				"severity": level,
			},
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

	// Add summary attributes for logging/display
	if summary.Attributes == nil {
		summary.Attributes = make(map[string]string)
	}
	summary.Attributes["tool_version"] = run.Tool.Driver.Version
	summary.Attributes["high_severity"] = strconv.Itoa(severityCounts["error"])
	summary.Attributes["medium_severity"] = strconv.Itoa(severityCounts["warning"])
	summary.Attributes["low_severity"] = strconv.Itoa(severityCounts["note"])
	summary.Attributes["total_vulnerabilities"] = strconv.Itoa(len(run.Results))
	summary.Attributes["unique_rules_violated"] = strconv.Itoa(len(rulesViolated))

	// Create Prometheus metrics for security findings
	timestamp := summary.StartTime.UnixMilli()
	summary.Metrics = []metrics.Metric{
		{
			Name:      "zizmor_total_vulnerabilities",
			Value:     float64(len(run.Results)),
			Labels:    map[string]string{"tool_version": run.Tool.Driver.Version},
			Timestamp: timestamp,
		},
		{
			Name:      "zizmor_high_severity",
			Value:     float64(severityCounts["error"]),
			Labels:    map[string]string{"severity": "high"},
			Timestamp: timestamp,
		},
		{
			Name:      "zizmor_medium_severity",
			Value:     float64(severityCounts["warning"]),
			Labels:    map[string]string{"severity": "medium"},
			Timestamp: timestamp,
		},
		{
			Name:      "zizmor_low_severity",
			Value:     float64(severityCounts["note"]),
			Labels:    map[string]string{"severity": "low"},
			Timestamp: timestamp,
		},
		{
			Name:      "zizmor_unique_rules_violated",
			Value:     float64(len(rulesViolated)),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
	}

	// Add per-rule violation count metrics and a scan_completed marker
	for ruleID, count := range rulesViolated {
		summary.Metrics = append(summary.Metrics, metrics.Metric{
			Name:      "zizmor_rule_violations",
			Value:     float64(count),
			Labels:    map[string]string{"rule_id": ruleID},
			Timestamp: timestamp,
		})
	}

	// Always emit scan_completed so repos that run zizmor are visible even with zero findings
	summary.Metrics = append(summary.Metrics, metrics.Metric{
		Name:      "zizmor_scan_completed",
		Value:     1.0,
		Labels:    map[string]string{},
		Timestamp: timestamp,
	})

	// Check if execution was successful
	if len(run.Invocations) > 0 && !run.Invocations[0].ExecutionSuccessful {
		summary.Status = executor.SuiteFailed
	}

	return summary, nil
}
