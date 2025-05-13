package gotest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

var (
	ErrInvalidFormat = errors.New("invalid format")
)

type line struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float32 `json:"Elapsed"`
}

type testkey struct {
	pkg  string
	test string
}

type testRuns map[testkey]*executor.TestRunSummary

// ParseJsonOutput parses output from https://pkg.go.dev/cmd/test2json
func ParseJsonOutput(report io.Reader) (executor.SuiteRunSummary, error) {
	summary := executor.SuiteRunSummary{
		StartTime: time.Now(),
	}

	testRuns, err := parseTestRuns(report)
	if err != nil {
		return executor.SuiteRunSummary{}, err
	}

	endTime := time.Time{}
	for _, test := range testRuns {
		// set the summary time to the first test that started don't relay on ordering of tests
		if test.StartTime.Before(summary.StartTime) {
			summary.StartTime = test.StartTime
		}

		// calculate the latest end time among all tests
		testDuration := time.Duration(float32(time.Second) * test.Durations.TotalDuration)
		if test.StartTime.Add(testDuration).After(endTime) {
			endTime = test.StartTime.Add(testDuration)
		}

		if test.Status == executor.TestSkipped {
			continue
		}

		switch test.Status {
		case executor.TestError:
			summary.TestsError += 1
		case executor.TestPassed:
			summary.TestsPassed += 1
		case executor.TestFailed:
			summary.TestsFailed += 1
		case executor.TestFlaky:
			summary.TestsFlaky += 1
		}

		summary.TestRuns = append(summary.TestRuns, *test)
	}

	summary.TotalDuration += float32(endTime.Sub(summary.StartTime).Seconds())
	summary.TestsExecuted = int32(len(summary.TestRuns))

	return summary, nil
}

func parseTestRuns(report io.Reader) (testRuns, error) {
	testRuns := testRuns{}

	decoder := json.NewDecoder(report)
	for {
		var line line
		err := decoder.Decode(&line)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		// we are only processing tests for now
		if line.Test == "" {
			continue
		}

		testkey := testkey{pkg: line.Package, test: line.Test}

		if line.Action == "start" || line.Action == "run" { // found a new stream, initialize
			time, err := time.Parse(time.RFC3339, line.Time)
			if err != nil {
				return nil, err
			}
			testRuns[testkey] = &executor.TestRunSummary{
				StartTime:  time,
				TestFolder: line.Package,
				TestFile:   line.Test,
			}

			continue
		}

		testRun, ok := testRuns[testkey]
		if !ok {
			return nil, fmt.Errorf("%w missing start/run for test %q %v", ErrInvalidFormat, line.Package, line.Test)
		}

		switch line.Action {
		case "output":
			testRun.ExitMessage = testRun.ExitMessage + line.Output
			continue
		case "pass":
			testRun.Status = executor.TestPassed
			testRun.ExitMessage = "" // delete message for passed tests to reduce noise
		case "fail":
			testRun.Status = executor.TestFailed
		case "skip":
			testRun.Status = executor.TestSkipped
		default:
			continue
		}

		testRun.Durations.TotalDuration = line.Elapsed
	}

	return testRuns, nil
}
