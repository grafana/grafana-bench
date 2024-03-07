package playwright

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func getTextSummaryOutputFile(filename, path string) string {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	// return os.ReadFile(path + name)
	return path + name
}

type TestDurations struct {
	SetupDuration    float32
	ScenarioDuration float32
	TeardownDuration float32
	TotalDuration    float32
}

type RunTags struct {
	testRun      string
	scenarioName string
	folder       string
	testFile     string
	order        string
}

type RunSummary struct {
	tags      RunTags
	Durations TestDurations

	AnyFailures bool
	ExitCode    int
	ExitMessage string
	Iterations  string
}

func newTestIdentifier(filename string, startTime time.Time, projectId string, scenario string, grafanaVersion string) string {
	return fmt.Sprintf("%s-%s-%s-%s-graf-%s",
		filepath.Base(filename),
		startTime.Format("15:04:05"),
		projectId,
		scenario,
		grafanaVersion,
	)
}

func parseTextSummaryOutput(log *slog.Logger, buf []byte) (RunSummary, error) {
	var (
		summary RunSummary
		err     error
	)

	if err != nil {
		return summary, fmt.Errorf("parsing k6 summary: %w", err)
	}

	return summary, nil
}

// parseJsonOutput parses the json output from playwright --report json and returns a slice of RunSummary
// this will work if only one test is run and the output but will also work for if this contains an entire suite
func parseJsonOutput(log *slog.Logger, buf []byte) ([]RunSummary, error) {

	output := PlaywrightJsonOutput{}

	fmt.Println(string(buf))

	err := json.Unmarshal(buf, &output)
	if err != nil {
		return nil, fmt.Errorf("parsing Playwright json summary output: %w", err)
	}

	summaries := make([]RunSummary, 0, output.Stats.Expected+output.Stats.Unexpected)

	for _, suite := range output.Suites {
		for _, spec := range suite.Specs {

			exitMessage := "success"
			if !spec.Ok {
				exitMessage = fmt.Sprintf("%s:%d:%d => %s", spec.File, spec.Line, spec.Column, spec.Tests[0].Results[0].Error.Message)
			}

			folder := "unknown"
			if len(spec.Tests) > 0 {
				for _, project := range output.Config.Projects {
					if spec.Tests[0].ProjectID == project.ID {
						folder = project.TestDir
						break
					}
				}
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

			summary := RunSummary{
				tags: RunTags{
					testRun:      newTestIdentifier(spec.File, output.Stats.StartTime, spec.Tests[0].ProjectID, spec.Title, "0.0.0"),
					scenarioName: spec.Title,
					testFile:     path.Base(spec.File),
					folder:       folder,
					order:        "0", // relevence of order?
				},

				AnyFailures: !spec.Ok,
				ExitCode:    0, // what are the other values here and what do they mean?
				Iterations:  fmt.Sprintf("%d", len(spec.Tests[0].Results)),
				ExitMessage: exitMessage,

				Durations: TestDurations{
					SetupDuration:    float32(output.Config.GlobalSetup),
					TeardownDuration: float32(output.Config.GlobalTeardown),
					ScenarioDuration: averageScenarioDuration,
					TotalDuration:    float32(scenarioTotal),
				},
			}

			summaries = append(summaries, summary)
		}

	}

	return summaries, nil

}

func logTestOutput(log *slog.Logger, testSummaries []RunSummary) error {

	// for _, summary := range testSummaries {
	// 	// test complete log
	// 	// testTags := []any{
	// 	// 	"testRun", summary.tags.testRun,
	// 	// 	"scenarioName", summary.tags.scenarioName,
	// 	// 	"folder", summary.tags.folder,
	// 	// 	"testFile", summary.tags.testFile,
	// 	// 	"order", "0",
	// 	// }
	// 	// log.With(t.suiteRunLogAttrs()...).
	// 	// 	With(testTags...).
	// 	// 	Info("testrun")

	// }
	return nil
}
