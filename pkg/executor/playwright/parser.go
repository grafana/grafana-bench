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

	testRuns := make([]executor.TestRun, 0, output.Stats.Expected+output.Stats.Unexpected)

	var suiteStatus executor.SuiteStatus = executor.SuitePassed
	for _, suite := range output.Suites {
		for _, spec := range suite.Specs {
			folder := "unknown"
			if len(spec.Tests) > 0 {
				for _, project := range output.Config.Projects {
					if spec.Tests[0].ProjectID == project.ID {
						folder = project.TestDir
						break
					}
				}
			}

			setupDuration := float32(output.Config.GlobalSetup)
			tearDownDuration := float32(output.Config.GlobalTeardown)

			run := formatTestRuns(spec, folder, setupDuration, tearDownDuration)
			if run.Status == executor.TestFailed {
				suiteStatus = executor.SuiteFailed
			}
			testRuns = append(testRuns, run)
		}
	}

	totalTestAmount := int32(output.Stats.Unexpected) + int32(output.Stats.Expected)

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

func formatTestRuns(spec Specs, folder string, globalSetupDuration, globalTeardownDuration float32) executor.TestRun {
	exitMessage := "success"
	testStatus := executor.TestPassed

	if spec.Tests[0].Status == "skipped" {
		return executor.TestRun{
			TestFolder: folder,
			TestFile:   path.Base(spec.File),

			Status:      executor.TestSkipped,
			ExitMessage: "skipped",
			Iterations:  "0",

			Durations: executor.TestDurations{
				SetupDuration:    globalSetupDuration,
				TeardownDuration: globalTeardownDuration,
				ScenarioDuration: float32(0),
				TotalDuration:    float32(0),
			},

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

	scenarioTotal := 0
	amount := 0
	for _, test := range spec.Tests {
		for _, result := range test.Results {
			scenarioTotal += result.Duration
		}
		amount += len(test.Results)
	}

	averageScenarioDuration := float32(math.Round(float64(scenarioTotal / amount)))

	summary := executor.TestRun{
		TestFolder: folder,
		TestFile:   path.Base(spec.File),

		StartTime: spec.Tests[0].Results[0].StartTime,

		Status:      testStatus,
		ExitMessage: exitMessage,
		Iterations:  fmt.Sprintf("%d", len(spec.Tests[0].Results)),

		Durations: executor.TestDurations{
			SetupDuration:    globalSetupDuration,
			TeardownDuration: globalTeardownDuration,
			ScenarioDuration: averageScenarioDuration,
			TotalDuration:    float32(scenarioTotal),
		},

		Attributes: map[string]string{
			"title":  spec.Title,
			"line":   fmt.Sprint(spec.Line),
			"column": fmt.Sprint(spec.Column),
		},
	}

	return summary
}
