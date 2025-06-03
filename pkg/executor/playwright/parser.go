package playwright

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/acarl005/stripansi"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// ParseJsonOutput parses the json output from playwright --report json and returns a slice of RunSummary
// this will work if only one test is run and the output but will also work for if this contains an entire suite
func ParseJsonOutput(report io.Reader) (executor.SuiteRunSummary, error) {
	output := PlaywrightJsonOutput{}

	buf, err := io.ReadAll(report)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("error failed to read report.json: %s", err.Error())
	}

	err = json.Unmarshal(buf, &output)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("parsing Playwright json summary output: %w", err)
	}

	// collect the test dirs per project. Needed to map tests to folders
	testDirs := map[string]string{}
	for _, project := range output.Config.Projects {
		testDirs[project.ID] = project.TestDir
	}

	testRuns := parseSuites(output.Suites, testDirs, nil)

	totalTestAmount := int32(output.Stats.Unexpected) + int32(output.Stats.Expected)
	var suiteStatus executor.SuiteStatus = executor.SuitePassed
	if output.Stats.Unexpected > 0 {
		suiteStatus = executor.SuiteFailed
	}

	scenarioDuration := time.Duration(0)
	for _, r := range testRuns {
		scenarioDuration += r.Durations.ScenarioDuration
	}
	suiteRunSummary := executor.SuiteRunSummary{
		Status:            suiteStatus,
		StartTime:         output.Stats.StartTime,
		ScenariosDuration: scenarioDuration,
		TestsExecuted:     totalTestAmount,
		TestsFailed:       int32(output.Stats.Unexpected),
		TestsPassed:       int32(output.Stats.Expected),
		TestsError:        0,
		TotalDuration:     time.Duration(output.Stats.Duration * float64(time.Millisecond)),
		TestRuns:          testRuns,
	}

	return suiteRunSummary, nil

}

func parseSuites(suites []Suite, testDirs map[string]string, testRuns []executor.TestRunSummary) []executor.TestRunSummary {
	for _, suite := range suites {
		for _, spec := range suite.Specs {
			folder := "unknown"
			if len(spec.Tests) > 0 {
				folder = testDirs[spec.Tests[0].ProjectID]
			}

			run := parseTestRun(spec, folder)
			if run.Status != executor.TestSkipped {
				testRuns = append(testRuns, run)
			}
		}

		testRuns = parseSuites(suite.Suites, testDirs, testRuns)
	}

	return testRuns
}

func parseTestRun(spec Specs, folder string) executor.TestRunSummary {
	run := executor.TestRunSummary{
		TestFolder: folder,
		TestFile:   path.Base(spec.File),
		Iterations: "0",
		Attributes: map[string]string{
			"title":  spec.Title,
			"line":   fmt.Sprint(spec.Line),
			"column": fmt.Sprint(spec.Column),
		},
	}

	switch spec.Tests[0].Status {
	case "skipped":
		run.Status = executor.TestSkipped
		run.ExitMessage = "skipped"
	case "expected":
		run.Status = executor.TestPassed
		run.ExitMessage = "success"
	case "unexpected":
		run.Status = executor.TestFailed
		msg := stripansi.Strip(spec.Tests[0].Results[0].Error.Message)
		// only take the first line of the error message incase it is a call stack message
		msg = strings.Split(msg, "\n")[0]
		run.ExitMessage = fmt.Sprintf("%s:%d:%d => %s", spec.File, spec.Line, spec.Column, msg)
	case "flaky":
		run.Status = executor.TestFlaky
	}

	// FIXME: tests can be executed more than once due to retries so we average the duration.
	// maybe we should take the duration of the last execution (either failed or passed)
	scenarioTotal := float64(0)
	executions := 0
	for _, test := range spec.Tests {
		for _, result := range test.Results {
			scenarioTotal += result.Duration
			executions += 1
		}
	}

	averageScenarioDuration := float64(scenarioTotal)
	if executions > 0 {
		averageScenarioDuration = scenarioTotal / float64(executions)
	}

	run.Iterations = fmt.Sprintf("%d", len(spec.Tests[0].Results))
	// the test was not skipped and has results
	if len(spec.Tests[0].Results) > 0 {
		run.StartTime = spec.Tests[0].Results[0].StartTime
	}
	run.Durations = executor.TestDurations{
		ScenarioDuration: time.Duration(averageScenarioDuration * float64(time.Millisecond)),
		TotalDuration:    time.Duration(scenarioTotal * float64(time.Millisecond)),
	}

	return run
}
