package playwright

import (
	"os"
	"testing"

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
		// {
		// 	title: "parse successful test correctly",
		// 	file:  "./mocks/success.json",
		// 	expected: executor.SuiteRunSummary{
		// 		TestRuns: []executor.TestRun{
		// 			{
		// 				TestFile:    "smoke.test.ts",
		// 				Status:      executor.TestPassed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 2148,
		// 					TotalDuration: 2148,
		// 				},
		// 				ExitMessage: "success",
		// 				Attributes: map[string]string{
		// 					"title": "should redirect to start page when permissions to navigate to page is missing",
		// 				},
		// 			},
		// 		},
		// 		TotalDuration: float32(2745.645),
		// 		TestsExecuted: 1,
		// 		TestsPassed:   1,
		// 		TestsFailed:   0,
		// 		TestsError:    0,
		// 	},
		// },
		// {
		// 	title: "parse failure test correctly",
		// 	file:  "./mocks/failures.json",
		// 	expected: executor.SuiteRunSummary{
		// 		TestRuns: []executor.TestRun{
		// 			{
		// 				TestFile:    "failures.test.ts",
		// 				Status:      executor.TestFailed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 5001,
		// 					TotalDuration: 5001,
		// 				},
		// 				ExitMessage: "failures.test.ts:22:6 => Test timeout of 5000ms exceeded.",
		// 				Attributes: map[string]string{
		// 					"title": "should fail due to missing element",
		// 				},
		// 			},
		// 		},
		// 		TotalDuration: float32(5942.315),
		// 		TestsExecuted: 1,
		// 		TestsPassed:   0,
		// 		TestsFailed:   1,
		// 		TestsError:    0,
		// 	},
		// },
		// {
		// 	title:             "parse fully skipped tests correctly",
		// 	file:              "./mocks/broken.json",
		// 	expected: executor.SuiteRunSummary{
		// 		TestRuns: []executor.TestRun{
		// 			{
		// 				TestFile:    "skipped.test.ts",
		// 				Status:      executor.TestSkipped,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 0,
		// 					TotalDuration: 0,
		// 				},
		// 				ExitMessage: "skipped",
		// 				Attributes: map[string]string{
		// 					"title": "data query should return values 1 and 3",
		// 				},
		// 			},
		// 		},
		// 		TotalDuration: float32(2327.512),
		// 		TestsExecuted: 1,
		// 		TestsPassed:   0,
		// 		TestsFailed:   1,
		// 		TestsError:    0,
		// 	},
		// },
		// {
		// 	title:             "parse output with passed and failed tests",
		// 	file:              "./mocks/full-report.json",
		// 	expected: executor.SuiteRunSummary{
		// 		TestRuns: []executor.TestRun{
		// 			{
		// 				TestFile:    "auth.setup.js",
		// 				Status:      executor.TestPassed,
		// 				ExitMessage: "success",
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 104,
		// 					TotalDuration: 104,
		// 				},
		// 				Attributes: map[string]string{
		// 					"title": "authenticate",
		// 				},
		// 			},
		// 			{
		// 				TestFile:    "failures.test.ts",
		// 				Status:      executor.TestFailed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 2557,
		// 					TotalDuration: 2557,
		// 				},
		// 				ExitMessage: "failures.test.ts:3:5 => Error: ENOENT: no such file or directory, open '/Users/timmulqueen/projects/grafana-plugin-tests/provisioning/datasources/jfkladsjfkldasjdfklasjlk.yml'",
		// 				Attributes: map[string]string{
		// 					"title": "should fail due to missing file",
		// 				},
		// 			},
		// 			{
		// 				TestFile:    "failures.test.ts",
		// 				Status:      executor.TestFailed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 1570,
		// 					TotalDuration: 1570,
		// 				},
		// 				ExitMessage: "failures.test.ts:13:5 => Error: expect(received).toEqual(expected) // deep equality",
		// 				Attributes: map[string]string{
		// 					"title": "should fail due to expect",
		// 				},
		// 			},
		// 			{
		// 				TestFile:    "failures.test.ts",
		// 				Status:      executor.TestFailed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 30000,
		// 					TotalDuration: 30000,
		// 				},
		// 				ExitMessage: "failures.test.ts:21:5 => Test timeout of 30000ms exceeded.",
		// 				Attributes: map[string]string{
		// 					"title": "should fail due to missing element",
		// 				},
		// 			},
		// 			{
		// 				TestFile:    "failures.test.ts",
		// 				Status:      executor.TestFailed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 1437,
		// 					TotalDuration: 1437,
		// 				},
		// 				ExitMessage: "failures.test.ts:26:5 => Error: This is a random javascript type error failure",
		// 				Attributes: map[string]string{
		// 					"title": "should fail due to type error in test",
		// 				},
		// 			},
		// 			{
		// 				TestFile:    "smoke.test.ts",
		// 				Status:      executor.TestPassed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 2774,
		// 					TotalDuration: 2774,
		// 				},
		// 				ExitMessage: "success",
		// 				Attributes: map[string]string{
		// 					"title": "data query should return values 1 and 3",
		// 				},
		// 			},
		// 			{
		// 				TestFile:    "smoke.test.ts",
		// 				Status:      executor.TestPassed,
		// 				Durations:    executor.TestDurations{
		// 					ScenarioDuration: 1518,
		// 					TotalDuration: 1518,
		// 				},
		// 				ExitMessage: "success",
		// 				Attributes: map[string]string{
		// 					"title": "should redirect to start page when permissions to navigate to page is missing",
		// 				},
		// 			},

		// 		},
		// 		TotalDuration: float32(37814.297),
		// 		TestsExecuted: 7,
		// 		TestsPassed:   3,
		// 		TestsFailed:   4,
		// 		TestsError:    0,
		// 	},
		// },
		{
			title: "parse nested suites",
			file:  "./mocks/nested-suites.json",
			expected: executor.SuiteRunSummary{
				TestRuns: []executor.TestRun{
					{
						TestFile:    "auth.setup.js",
						Status:      executor.TestPassed,
						Durations:    executor.TestDurations{
							ScenarioDuration: 63,
							TotalDuration: 63,
						},
						ExitMessage: "success",
						Attributes: map[string]string{
							"title": "authenticate",
						},
					},
					{
						TestFile:    "configEditor.spec.ts",
						Status:      executor.TestFailed,
						Durations:    executor.TestDurations{
							ScenarioDuration: 10159,
							TotalDuration: 10159,
						},
						ExitMessage: "configEditor.spec.ts:7:7 => Test timeout of 10000ms exceeded.",
						Attributes: map[string]string{
							"title": "invalid credentials should return an error",
						},
					},
					{
						TestFile:    "configEditor.spec.ts",
						Status:      executor.TestFailed,
						Durations:    executor.TestDurations{
							ScenarioDuration: 10155,
							TotalDuration: 10155,
						},
						ExitMessage: "configEditor.spec.ts:13:7 => Test timeout of 10000ms exceeded.",
						Attributes: map[string]string{
							"title": "valid credentials should display a success alert on the page",
						},
					},
					{
						TestFile:    "configEditor.spec.ts",
						Status:      executor.TestFailed,
						Durations:    executor.TestDurations{
							ScenarioDuration: 10155,
							TotalDuration: 10155,
						},
						ExitMessage: "configEditor.spec.ts:26:7 => Test timeout of 10000ms exceeded.",
						Attributes: map[string]string{
							"title": "mandatory fields should show error if left empty",
						},
					},
				},
				TotalDuration: float32(11508.904),
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

			summary, err := parseJsonOutput(file)
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
