package playwright

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path"
	"strings"

	"github.com/acarl005/stripansi"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// parseJsonOutput parses the json output from playwright --report json and returns a slice of RunSummary
// this will work if only one test is run and the output but will also work for if this contains an entire suite
func parseJsonOutput(report io.Reader) (executor.SuiteRunSummary, error) {
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

	suiteRunSummary := executor.SuiteRunSummary{
		Status:            suiteStatus,
		StartTime:         output.Stats.StartTime,
		ScenariosDuration: float32(output.Stats.Duration),
		TestsExecuted:     totalTestAmount,
		TestsFailed:       int32(output.Stats.Unexpected),
		TestsPassed:       int32(output.Stats.Expected),
		TestsError:        0,
		TotalDuration:     float32(output.Stats.Duration),
		TestRuns:          testRuns,
	}

	return suiteRunSummary, nil

}


func parseSuites( suites []Suite, testDirs map[string]string, testRuns []executor.TestRun) []executor.TestRun {
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

func parseTestRun(spec Specs, folder string) executor.TestRun {
	exitMessage := "success"
	testStatus := executor.TestPassed

	if spec.Tests[0].Status == "skipped" {
		return executor.TestRun{
			TestFolder: folder,
			TestFile:   path.Base(spec.File),

			Status:      executor.TestSkipped,
			ExitMessage: "skipped",
			Iterations:  "0",

			Attributes: map[string]string{
				"title":  spec.Title,
				"line":   fmt.Sprint(spec.Line),
				"column": fmt.Sprint(spec.Column),
			},
		}
	}

	if !spec.Ok {
		msg := stripansi.Strip(spec.Tests[0].Results[0].Error.Message)
		// only take the first line of the error message incase it is a call stack message
		msg = strings.Split(msg, "\n")[0]
		exitMessage = fmt.Sprintf("%s:%d:%d => %s", spec.File, spec.Line, spec.Column, msg)
		testStatus = executor.TestFailed
	}

	// tests can be executed more than once due to retries so we need to average the duration
	scenarioTotal := 0
	executions := 0
	for _, test := range spec.Tests {
		for _, result := range test.Results {
			scenarioTotal += result.Duration
			executions += 1
		}
	}

	averageScenarioDuration := float32(scenarioTotal)
	if executions > 0 {
		averageScenarioDuration	= float32(math.Round(float64(scenarioTotal) / float64(executions)))
	}

	run := executor.TestRun{
		TestFolder: folder,
		TestFile:   path.Base(spec.File),

		StartTime: spec.Tests[0].Results[0].StartTime,

		Status:      testStatus,
		ExitMessage: exitMessage,
		Iterations:  fmt.Sprintf("%d", len(spec.Tests[0].Results)),

		Durations: executor.TestDurations{
			ScenarioDuration: float32(averageScenarioDuration),
			TotalDuration:    float32(scenarioTotal),
		},

		Attributes: map[string]string{
			"title":  spec.Title,
			"line":   fmt.Sprint(spec.Line),
			"column": fmt.Sprint(spec.Column),
		},
	}

	return run
}
