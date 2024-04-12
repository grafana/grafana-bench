package cypress

import (
	"encoding/json"
	"fmt"

	e "github.com/grafana/grafana-bench/pkg/executor"
)

// parseJsonOutput parses the json output from playwright --report json and returns a slice of RunSummary
// this will work if only one test is run and the output but will also work for if this contains an entire suite
func parseJsonOutput(buf []byte) (e.SuiteRunSummary, error) {
	output := CypressJsonOutput{}

	err := json.Unmarshal(buf, &output)
	if err != nil {
		return e.SuiteRunSummary{}, fmt.Errorf("parsing Playwright json summary output: %w", err)
	}

	testRuns := make([]e.TestRun, 0, output.Stats.Expected+output.Stats.Unexpected)

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
			testRuns = append(testRuns, run)
		}
	}

	totalTestAmount := int32(output.Stats.Unexpected) + int32(output.Stats.Expected)

	suiteRunSummary := e.SuiteRunSummary{
		StartTime:         output.Stats.StartTime,
		ScenariosDuration: float32(output.Stats.Duration),
		TestsExecuted:     totalTestAmount,
		TestsFailed:       int32(output.Stats.Unexpected),
		TestsPassed:       int32(output.Stats.Expected),
		TestsError:        int32(output.Stats.Unexpected),
		TotalDuration:     float32(output.Stats.Duration),
		TestRuns:          testRuns,
	}

	return suiteRunSummary, nil

}

func formatTestRuns(spec Specs, folder string, globalSetupDuration, globalTeardownDuration float32) e.TestRun {
	// exitMessage := "success"
	// if !spec.Ok {
	// 	exitMessage = fmt.Sprintf("%s:%d:%d => %s", spec.File, spec.Line, spec.Column, spec.Tests[0].Results[0].Error.Message)
	// }

	// scenarioTotal := 0
	// amount := 0
	// for _, test := range spec.Tests {
	// 	for _, result := range test.Results {
	// 		scenarioTotal += result.Duration
	// 	}
	// 	amount += len(test.Results)
	// }

	// averageScenarioDuration := float32(math.Round(float64(scenarioTotal / amount)))

	// testStatus := e.TestPassed
	// status := spec.Tests[0].Results[0].Status
	// if status == "failed" {
	// 	testStatus = e.TestFailed
	// }

	// summary := e.TestRun{
	// 	TestFolder: folder,
	// 	TestFile:   path.Base(spec.File),

	// 	StartTime: spec.Tests[0].Results[0].StartTime,
	// 	Order:     0,

	// 	Status:      testStatus,
	// 	ExitMessage: exitMessage,
	// 	ExitCode:    0, // what are the other values here and what do they mean?
	// 	Iterations:  fmt.Sprintf("%d", len(spec.Tests[0].Results)),

	// 	Durations: e.TestDurations{
	// 		SetupDuration:    globalSetupDuration,
	// 		TeardownDuration: globalTeardownDuration,
	// 		ScenarioDuration: averageScenarioDuration,
	// 		TotalDuration:    float32(scenarioTotal),
	// 	},
	// }

	return summary
}
