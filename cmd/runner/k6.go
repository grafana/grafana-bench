package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// K6RunSummary summarizes the execution of a k6 execution
type K6RunSummary struct {
	ExitCode    int
	ExitMessage string
	Iterations  string
	Durations   TestDurations
	CloudId     string
	CloudURL    string
}

// Execute the test
func K6ExecTest(log *slog.Logger, testFile string, scenarioName string, runIdentifier string, env map[string]string, args ...string) (K6RunSummary, error) {
	jsonFile := getJsonOutputFilename(testFile)

	// build the command with buffer
	cmd, buf := prepareK6Command(
		runIdentifier,
		testFile,
		jsonFile,
		env,
		args...)

	// run command
	var (
		cmdErr   string
		exitCode int
	)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		cmdErr = "error running k6 command: " + err.Error()
		log.Info("error running k6 command", "error", err)
	}

	// scenario + testDuration will be in milliseconds
	duration, err := parseDurationFromJsonFile(log, scenarioName, jsonFile)
	if err != nil {
		log.Warn("error processing json file", "error", err)
	}

	cloudId, cloudURL, err := parseK6CloudIdentifiersFromCLIOutput(log, buf.Bytes())
	if err != nil {
		log.Warn("error parsing cloud run from K6 summary", "error", err)
	}

	iterations, err := parseIterationCountFromCLIOutput(buf.Bytes())
	if err != nil {
		log.Warn("error parsing iterations from k6 summary", "error", err)
	}

	return K6RunSummary{
		ExitCode:    exitCode,
		Durations:   duration,
		Iterations:  iterations,
		ExitMessage: cmdErr,
		CloudId:     cloudId,
		CloudURL:    cloudURL,
	}, nil
}

// pattern to match the cloud output url inside parenthesis
//
//	output: cloud (https://jefflevinslunch.grafana.net/a/k6-app/runs/1876021), json (/tmp/dashboard_create.json)
var k6CloudOutputURLPattern = regexp.MustCompile(`\s*cloud\s*\(([^)]+)\)`)
var K6CloudOutputIDPattern = regexp.MustCompile(`(\d+)$`)

// prepareK6Command builds the command with output set to standard output and a
// buffer and passes the cmd and buffer back to be executed and parsed
func prepareK6Command(identifier, testFile, jsonFile string, envVars map[string]string, args ...string) (*exec.Cmd, *bytes.Buffer) {
	defaultArgs := []string{
		"run",
		testFile,
		"--out", fmt.Sprintf(`json=%s`, jsonFile),
	}
	defaultArgs = append(defaultArgs, args...)

	cmd := exec.Command("k6", defaultArgs...)

	envVars["path"] = os.Getenv("PATH")
	envVars["K6_BROWSER_ENABLED"] = "true"
	envVars["K6_BROWSER_ARGS"] = "no-sandbox"
	// set env vars
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	buf := bytes.NewBuffer(nil)
	cmd.Stdout = io.MultiWriter(buf, os.Stderr)
	cmd.Stderr = os.Stderr

	return cmd, buf
}

// parseK6CloudIdentifiersFromCLIOutput parses cloud run id and url output of k6 cli
func parseK6CloudIdentifiersFromCLIOutput(log *slog.Logger, b []byte) (string, string, error) {
	// Find the first match of the pattern in the input
	match := k6CloudOutputURLPattern.FindSubmatch(b)

	if len(match) >= 2 {
		// The URL is captured in the second element of the match
		url := string(match[1])
		log.Info("Found k6 cloud output url", "url", url)

		matches := K6CloudOutputIDPattern.FindStringSubmatch(url)
		if len(matches) > 1 {
			id := matches[1]
			log.Info("Found K6 cloud output id", "id", id)
			return id, url, nil
		} else {
			return "", url, fmt.Errorf("K6 cloud output id not found! this should not happen if we have a correctly formed url")
		}
	} else {
		return "", "", fmt.Errorf("URL not found")
	}
}

// pattern to match iterations count from k6 summary
// http_reqs......................: 4       201.085864/s
// iteration_duration.............: avg=6.06ms   min=2.24ms med=3.72ms  max=12.21ms p(90)=10.51ms p(95)=11.36ms
// iterations.....................: 1       50.271466/s
var iterationPattern = regexp.MustCompile(`iterations\.*?:\s*(\d+)`)

// parseIterationCountFromCLIOutput parses iteration count from output of k6 cli
func parseIterationCountFromCLIOutput(b []byte) (string, error) {
	match := iterationPattern.FindSubmatch(b)
	if len(match) > 1 {
		return string(match[1]), nil
	}
	return "", fmt.Errorf("iterations not found in output. Was there an error running the test?")
}

type Metric struct {
	Metric string `json:"metric"`
	Type   string `json:"type"`
	Data   struct {
		Time  time.Time `json:"time"`
		Value float64   `json:"value"`
		Tags  struct {
			SuiteRun string `json:"suiteRun"`
			Group    string `json:"group,omitempty"`
			Scenario string `json:"scenario,omitempty"`
		} `json:"tags"`
	} `json:"data"`
}

type TestDurations struct {
	SetupDuration    float32
	ScenarioDuration float32
	TeardownDuration float32
	TotalDuration    float32
}

// {"metric":"iteration_duration","type":"Point","data":{"time":"2023-08-09T09:02:13.291575-08:00","value":325.78425,"tags":{"SUITE_RUN":"08bf3d97-155e-42d0-a709-bab1d8c08941","group":"::setup"}}
// {"metric":"iteration_duration","type":"Point","data":{"time":"2023-08-09T09:02:13.650349-08:00","value":358.328625,"tags":{"SUITE_RUN":"08bf3d97-155e-42d0-a709-bab1d8c08941","group":"","scenario":"createDashboard"}}}
// {"metric":"iteration_duration","type":"Point","data":{"time":"2023-08-09T09:02:16.191412-08:00","value":2149.020291,"tags":{"SUITE_RUN":"08bf3d97-155e-42d0-a709-bab1d8c08941","group":"::teardown"}}}
func parseDurationFromJsonFile(log *slog.Logger, scenarioName, jsonFile string) (TestDurations, error) {
	var td TestDurations

	file, err := os.Open(jsonFile)
	if err != nil {
		return td, err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Iterate over each line
	for scanner.Scan() {
		line := scanner.Bytes()
		var logEntry Metric
		err := json.Unmarshal(line, &logEntry)
		if err != nil {
			log.Warn("Error unmarshalling JSON", "err", err)
			continue
		}

		if logEntry.Metric != "iteration_duration" {
			continue
		}

		// Can we have non-empty group AND a non-empty scenario?
		//if logEntry.Data.Tags.Group != "" && logEntry.Data.Tags.Scenario != "" {
		//  fmt.Printf("%#v", logEntry)
		//}

		switch logEntry.Data.Tags.Group {
		case "::setup":
			td.SetupDuration += float32(logEntry.Data.Value)
		case "::teardown":
			td.TeardownDuration += float32(logEntry.Data.Value)
		default:
			if logEntry.Data.Tags.Scenario == scenarioName {
				td.ScenarioDuration += float32(logEntry.Data.Value)
			}
		}

	}

	// TODO review this. not entirely sure it makes sense.
	if err := scanner.Err(); err != nil {
		return td, fmt.Errorf("Error reading file: %w", err)
	}

	td.TotalDuration = td.SetupDuration + td.ScenarioDuration + td.TeardownDuration

	return td, nil
}

// dashboard_create.js -> /tmp/dashboard_create.json
func getJsonOutputFilename(filename string) string {
	jsonName := filepath.Base(filename)
	jsonName = strings.TrimSuffix(jsonName, filepath.Ext(jsonName))
	return path.Join("/tmp", jsonName+".json")
}

// prettyMS adds ms suffix to ms float
func prettyMS(ms float32) string {
	duration := time.Duration(ms) * time.Millisecond
	return fmt.Sprintf("%dms", duration.Milliseconds())
}
