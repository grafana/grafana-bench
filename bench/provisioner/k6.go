package provisioner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

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
func parseDurationFromJson(scenarioName, jsonFile string) (TestDurations, error) {
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
			fmt.Println("Error unmarshaling JSON:", err)
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

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	td.TotalDuration = td.SetupDuration + td.ScenarioDuration + td.TeardownDuration
	//fmt.Println("setupDuration:", td.SetupDuration)
	//fmt.Println("scenarioDuration", td.ScenarioDuration)
	//fmt.Println("teardownDuration:", td.TeardownDuration)
	//fmt.Println("totalDuration:", td.TotalDuration)

	return td, nil
}
