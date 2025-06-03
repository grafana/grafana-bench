package playwright

import (
	"os"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func assert(t *testing.T, message string, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s expected '%v', got '%v'", message, expected, actual)
	}
}

func TestParsePlaywrightJSONReport(t *testing.T) {
	testCases := []struct {
		title    string
		file     string
		expected executor.SuiteRunSummary
	}{
		{
			title: "parse successful test correctly",
			file:  "./testdata/success.json",
			expected: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{
						TestFile: "smoke.test.ts",
						Status:   executor.TestPassed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 2148,
							TotalDuration:    time.Millisecond * 2148,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "should redirect to start page when permissions to navigate to page is missing",
						},
					},
				},
				TotalDuration: time.Duration(float64(2745.645) * float64(time.Millisecond)),
				TestsExecuted: 1,
				TestsPassed:   1,
				TestsFailed:   0,
				TestsError:    0,
			},
		},
		{
			title: "parse failure test correctly",
			file:  "./testdata/failures.json",
			expected: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 5001,
							TotalDuration:    time.Millisecond * 5001,
						},
						ExitMessage: "failures.test.ts:22:6 => Test timeout of 5000ms exceeded.",
						Attributes: map[string]string{
							"title": "should fail due to missing element",
						},
					},
				},
				TotalDuration: time.Duration(float64(5942.315) * float64(time.Millisecond)),
				TestsExecuted: 1,
				TestsPassed:   0,
				TestsFailed:   1,
				TestsError:    0,
			},
		},
		{
			title: "parse fully skipped tests correctly",
			file:  "./testdata/skipped.json",
			expected: executor.SuiteRunSummary{
				TestRuns:      []executor.TestRunSummary{},
				TotalDuration: time.Duration(float64(2327.512) * float64(time.Millisecond)),
				TestsExecuted: 0,
				TestsPassed:   0,
				TestsFailed:   0,
				TestsError:    0,
			},
		},
		{
			title: "parse output with passed and failed tests",
			file:  "./testdata/full-report.json",
			expected: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{
						TestFile:    "auth.setup.js",
						Status:      executor.TestPassed,
						ExitMessage: "success",
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 104,
							TotalDuration:    time.Millisecond * 104,
						},
						Attributes: map[string]string{
							"title": "authenticate",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 2557,
							TotalDuration:    time.Millisecond * 2557,
						},
						ExitMessage: "failures.test.ts:3:5 => Error: ENOENT: no such file or directory, open '/Users/timmulqueen/projects/grafana-plugin-tests/provisioning/datasources/jfkladsjfkldasjdfklasjlk.yml'",
						Attributes: map[string]string{
							"title": "should fail due to missing file",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 1570,
							TotalDuration:    time.Millisecond * 1570,
						},
						ExitMessage: "failures.test.ts:13:5 => Error: expect(received).toEqual(expected) // deep equality",
						Attributes: map[string]string{
							"title": "should fail due to expect",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 30000,
							TotalDuration:    time.Millisecond * 30000,
						},
						ExitMessage: "failures.test.ts:21:5 => Test timeout of 30000ms exceeded.",
						Attributes: map[string]string{
							"title": "should fail due to missing element",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 1437,
							TotalDuration:    time.Millisecond * 1437,
						},
						ExitMessage: "failures.test.ts:26:5 => Error: This is a random javascript type error failure",
						Attributes: map[string]string{
							"title": "should fail due to type error in test",
						},
					},
					{
						TestFile: "smoke.test.ts",
						Status:   executor.TestPassed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 2774,
							TotalDuration:    time.Millisecond * 2774,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "data query should return values 1 and 3",
						},
					},
					{
						TestFile: "smoke.test.ts",
						Status:   executor.TestPassed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 1518,
							TotalDuration:    time.Millisecond * 1518,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "should redirect to start page when permissions to navigate to page is missing",
						},
					},
				},
				TotalDuration: time.Duration(float64(37814.297) * float64(time.Millisecond)),
				TestsExecuted: 7,
				TestsPassed:   3,
				TestsFailed:   4,
				TestsError:    0,
			},
		},
		{
			title: "parse output with retries",
			file:  "./testdata/report-retries.json",
			expected: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{
						TestFile:    "auth.setup.js",
						Status:      executor.TestPassed,
						ExitMessage: "success",
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 101,
							TotalDuration:    time.Millisecond * 101,
						},
						Attributes: map[string]string{
							"title": "authenticate",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 6757/3,
							TotalDuration:    time.Millisecond * 6757,
						},
						ExitMessage: "failures.test.ts:3:5 => Error: ENOENT: no such file or directory, open '/Users/timmulqueen/projects/grafana-plugin-tests/provisioning/datasources/jfkladsjfkldasjdfklasjlk.yml'",
						Attributes: map[string]string{
							"title": "should fail due to missing file",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 4628/3,
							TotalDuration:    time.Millisecond * 4628,
						},
						ExitMessage: "failures.test.ts:13:5 => Error: expect(received).toEqual(expected) // deep equality",
						Attributes: map[string]string{
							"title": "should fail due to expect",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 90000/3,
							TotalDuration:    time.Millisecond * 90000,
						},
						ExitMessage: "failures.test.ts:21:5 => Test timeout of 30000ms exceeded.",
						Attributes: map[string]string{
							"title": "should fail due to missing element",
						},
					},
					{
						TestFile: "failures.test.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 1527,
							TotalDuration:    time.Millisecond * 4581,
						},
						ExitMessage: "failures.test.ts:26:5 => Error: This is a random javascript type error failure",
						Attributes: map[string]string{
							"title": "should fail due to type error in test",
						},
					},
					{
						TestFile: "smoke.test.ts",
						Status:   executor.TestPassed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 2854,
							TotalDuration:    time.Millisecond * 2854,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "data query should return values 1 and 3",
						},
					},
					{
						TestFile: "smoke.test.ts",
						Status:   executor.TestPassed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 1565,
							TotalDuration:    time.Millisecond * 1565,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "should redirect to start page when permissions to navigate to page is missing",
						},
					},
				},
				// The value in the json file is 11114061499999999 but we lose some precision in the calculations
				TotalDuration: time.Duration(111140614999),
				TestsExecuted: 7,
				TestsPassed:   3,
				TestsFailed:   4,
				TestsError:    0,
			},
		},
		{
			title: "parse nested suites",
			file:  "./testdata/nested-suites.json",
			expected: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{
						TestFile: "auth.setup.js",
						Status:   executor.TestPassed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 63,
							TotalDuration:    time.Millisecond * 63,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "authenticate",
						},
					},
					{
						TestFile: "configEditor.spec.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 10159,
							TotalDuration:    time.Millisecond * 10159,
						},
						ExitMessage: "configEditor.spec.ts:7:7 => Test timeout of 10000ms exceeded.",
						Attributes: map[string]string{
							"title": "invalid credentials should return an error",
						},
					},
					{
						TestFile: "configEditor.spec.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 10155,
							TotalDuration:    time.Millisecond * 10155,
						},
						ExitMessage: "configEditor.spec.ts:13:7 => Test timeout of 10000ms exceeded.",
						Attributes: map[string]string{
							"title": "valid credentials should display a success alert on the page",
						},
					},
					{
						TestFile: "configEditor.spec.ts",
						Status:   executor.TestFailed,
						Durations: executor.TestDurations{
							ScenarioDuration: time.Millisecond * 10155,
							TotalDuration:    time.Millisecond * 10155,
						},
						ExitMessage: "configEditor.spec.ts:26:7 => Test timeout of 10000ms exceeded.",
						Attributes: map[string]string{
							"title": "mandatory fields should show error if left empty",
						},
					},
				},
				TotalDuration: time.Duration(float64(11508.904) * float64(time.Millisecond)),
				TestsExecuted: 4,
				TestsPassed:   1,
				TestsFailed:   3,
				TestsError:    0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			file, err := os.Open(tc.file)
			if err != nil {
				t.Fatalf("failed reading file: %s", err)
			}

			summary, err := ParseJsonOutput(file)
			if err != nil {
				t.Fatalf("failed parsing json file: %s", err)
			}

			assert(t, "tests executed", tc.expected.TestsExecuted, summary.TestsExecuted)
			assert(t, "tests passed", tc.expected.TestsPassed, summary.TestsPassed)
			assert(t, "tests error", tc.expected.TestsError, summary.TestsError)
			assert(t, "tests failed", tc.expected.TestsFailed, summary.TestsFailed)
			assert(t, "total duration", tc.expected.TotalDuration, summary.TotalDuration)

			assert(t, "test runs len", len(tc.expected.TestRuns), len(summary.TestRuns))
			for i, tr := range tc.expected.TestRuns {
				assert(t, "test file", tr.TestFile, summary.TestRuns[i].TestFile)
				assert(t, "test status", tr.Status, summary.TestRuns[i].Status)
				assert(t, "test durations", tr.Durations, summary.TestRuns[i].Durations)
				assert(t, "exit message", tr.ExitMessage, summary.TestRuns[i].ExitMessage)
				assert(t, "test title", tr.Attributes["title"], summary.TestRuns[i].Attributes["title"])
			}

		})
	}
}
