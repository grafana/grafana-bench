package playwright

import (
	"os"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func assert(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected '%v', got '%v'", expected, actual)
	}
}

func TestParsePlaywrightJSONReport(t *testing.T) {
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
			expectedTotal:     1,
			expectedPassed:    1,
			expectedFailed:    0,
			expectedError:     0,
			expectedDuration:  float32(2745.645),
			expectedFile:      "smoke.test.ts",
			expectedStatus:    "passed",
			expectedErrorMsg:  "success",
			expectedTestTitle: "should redirect to start page when permissions to navigate to page is missing",
		},
		{
			title:             "parse failure test correctly",
			file:              "./mocks/failures.json",
			expectedTotal:     1,
			expectedPassed:    0,
			expectedFailed:    1,
			expectedError:     0,
			expectedDuration:  float32(5942.315),
			expectedFile:      "failures.test.ts",
			expectedStatus:    "failed",
			expectedErrorMsg:  "failures.test.ts:22:6 => Test timeout of 5000ms exceeded.",
			expectedTestTitle: "should fail due to missing element",
		},
		{
			title:             "parse fully skipped tests correctly",
			file:              "./mocks/broken.json",
			expectedTotal:     1,
			expectedPassed:    0,
			expectedFailed:    1,
			expectedError:     0,
			expectedDuration:  float32(2327.512),
			expectedFile:      "skipped.test.ts",
			expectedStatus:    "skipped",
			expectedErrorMsg:  "skipped",
			expectedTestTitle: "data query should return values 1 and 3",
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
