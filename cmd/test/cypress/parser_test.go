package cypress

import (
	"os"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func assert(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected \n'%v'\n but got \n'%v'", expected, actual)
	}
}

func TestParseCypressJSONReport(t *testing.T) {
	testCases := []struct {
		title             string
		file              string
		expectedTotal     int32
		expectedPassed    int32
		expectedFailed    int32
		expectedError     int32
		expectedDuration  float32
		expectedFile      string
		expectedStatus    string
		expectedErrorMsg  string
		expectedTestTitle string
	}{
		{
			title:             "parse successful test correctly",
			file:              "./mocks/success.json",
			expectedTotal:     3,
			expectedPassed:    3,
			expectedFailed:    0,
			expectedError:     0,
			expectedDuration:  float32(29680),
			expectedFile:      "grafana-jira-datasource.cy.ts",
			expectedStatus:    "passed",
			expectedErrorMsg:  "",
			expectedTestTitle: "grafana-jira-datasource plugin: load the plugin correctly",
		},
		{
			title:             "parse failure test correctly",
			file:              "./mocks/failures.json",
			expectedTotal:     2,
			expectedPassed:    0,
			expectedFailed:    2,
			expectedError:     0,
			expectedDuration:  float32(53476),
			expectedFile:      "dlopes7-appdynamics-datasource.cy.ts",
			expectedStatus:    "failed",
			expectedErrorMsg:  "Timed out retrying after 30000ms: Expected to find content: '/Type: AppDynamics|Build a dashboard/g' but never did.",
			expectedTestTitle: "dlopes7-appdynamics-datasource appdynamics test",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			file, err := os.ReadFile(testCase.file)
			if err != nil {
				t.Fatalf("failed reading file: %s", err)
			}

			output, err := parseJsonOutput(file)
			if err != nil {
				t.Fatalf("failed parsing json file: %s", err)
			}

			assert(t, testCase.expectedTotal, output.TestsExecuted)
			assert(t, testCase.expectedPassed, output.TestsPassed)
			assert(t, testCase.expectedError, output.TestsError)
			assert(t, testCase.expectedFailed, output.TestsFailed)
			assert(t, testCase.expectedDuration, output.TotalDuration)

			assert(t, testCase.expectedFile, output.TestRuns[0].TestFile)
			assert(t, executor.TestStatus(testCase.expectedStatus), output.TestRuns[0].Status)
			assert(t, testCase.expectedErrorMsg, output.TestRuns[0].ExitMessage)
			assert(t, testCase.expectedTestTitle, output.TestRuns[0].Attributes["title"])
		})
	}
}
