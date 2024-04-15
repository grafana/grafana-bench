package playwright

import (
	"log/slog"
	"os"
	"testing"
)

func TestJsonTestsSummary(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	testCases := []struct {
		title    string
		input    string
		expected string
		wantErr  bool
	}{
		{
			title: "parse 1 from small output sample",
			input: "../mocks/report.json",
			// expected: ",
			wantErr: false,
		},
		// {
		// 	title:    "parse 100 from small output sample",
		// 	input:    []byte(iterationOutput100),
		// 	expected: "100",
		// 	wantErr:  false,
		// },
		// {
		// 	title:    "parse 100 from large output",
		// 	input:    []byte(iterationOutputFull),
		// 	expected: "100",
		// 	wantErr:  false,
		// },
		// {
		// 	title:    "parse 1 from small sample",
		// 	input:    []byte(iterationOuputMissing),
		// 	expected: "",
		// 	wantErr:  true,
		// },
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			pwd, _ := os.Getwd()
			jsonFile, err := os.ReadFile(pwd + "/mocks/report-retries.json")
			if err != nil {
				t.Errorf("Error reading file: %v, %s", err, pwd)
			}

			actual, err := parseJsonOutput(jsonFile)

			println(actual)

			if (err != nil) != testCase.wantErr {
				t.Errorf("Expected error: %v, but got error: %v", testCase.wantErr, err)
			}

			// if actual != testCase.expected {
			// 	t.Errorf("Expected: %s, but got: %s", testCase.expected, actual)
			// }
		})
	}
}
