package zizmor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
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

	// Track unique rules violated
	rulesViolated := make(map[string]bool)

	// Process each result as a test
	for _, result := range run.Results {
		level := result.Level
		if level == "" {
			level = "warning" // SARIF default
		}

		// Track severity
		severityCounts[level]++

		// Track rule
		rulesViolated[result.RuleID] = true

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

	// Add summary attributes for metrics
	if summary.Attributes == nil {
		summary.Attributes = make(map[string]string)
	}
	summary.Attributes["tool_version"] = run.Tool.Driver.Version
	summary.Attributes["high_severity"] = fmt.Sprintf("%d", severityCounts["error"])
	summary.Attributes["medium_severity"] = fmt.Sprintf("%d", severityCounts["warning"])
	summary.Attributes["low_severity"] = fmt.Sprintf("%d", severityCounts["note"])
	summary.Attributes["total_vulnerabilities"] = fmt.Sprintf("%d", len(run.Results))
	summary.Attributes["unique_rules_violated"] = fmt.Sprintf("%d", len(rulesViolated))

	// Check if execution was successful
	if len(run.Invocations) > 0 && !run.Invocations[0].ExecutionSuccessful {
		summary.Status = executor.SuiteFailed
	}

	return summary, nil
}
