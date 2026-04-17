// Seeds a local Loki with the fixture records needed to exercise
// `grafana-bench analyze` end-to-end.
//
// Usage:
//
//	go run ./test/analyze/seed -loki http://localhost:3100
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type pushReq struct {
	Streams []stream `json:"streams"`
}

type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func main() {
	lokiURL := flag.String("loki", "http://localhost:3100", "Loki base URL")
	service := flag.String("service", "grafana-pro", "value for the service field")
	runStage := flag.String("run-stage", "ci", "value for the runStage field")
	testFile := flag.String("test-file", "permissions.ts", "testFile to simulate a regression on")
	goodVersion := flag.String("good-version", "13.0.0-23563050832", "last-known-good grafanaVersion")
	badVersion := flag.String("bad-version", "13.0.0-23542128402", "first-bad grafanaVersion")
	flag.Parse()

	waitForLoki(*lokiURL)

	start := time.Now().Add(-2 * time.Hour)
	var records []map[string]any
	for i := 0; i < 3; i++ {
		records = append(records, testRunRecord(
			start.Add(time.Duration(i)*10*time.Minute),
			*service, *runStage, *testFile, *goodVersion,
			"passed", "", fmt.Sprintf("run-good-%d", i),
		))
	}
	for i := 0; i < 4; i++ {
		records = append(records, testRunRecord(
			start.Add(time.Duration(i+3)*10*time.Minute),
			*service, *runStage, *testFile, *badVersion,
			"failed",
			fmt.Sprintf("check rate expected 1 got 0.87 pid=%d at %s",
				1000+i, time.Now().Format(time.RFC3339)),
			fmt.Sprintf("run-bad-%d", i),
		))
	}

	if err := push(*lokiURL, *service, records); err != nil {
		log.Fatalf("push: %v", err)
	}
	fmt.Fprintf(os.Stdout, "seeded %d records into %s\n", len(records), *lokiURL)
}

func testRunRecord(ts time.Time, svc, stage, file, version, status, exitMsg, runID string) map[string]any {
	return map[string]any{
		"time":           ts.Format(time.RFC3339Nano),
		"level":          "INFO",
		"msg":            "testRun",
		"tool":           "bench",
		"service":        svc,
		"runStage":       stage,
		"testFile":       file,
		"folder":         "rrc-grafana-api-tests",
		"grafanaVersion": version,
		"grafanaSlug":    "k6testinstant1",
		"grafanaUrl":     "k6testinstant1.grafana-dev.net",
		"status":         status,
		"exitMessage":    exitMsg,
		"runId":          runID,
	}
}

func push(lokiURL, service string, records []map[string]any) error {
	values := make([][]string, 0, len(records))
	for _, rec := range records {
		ts, err := time.Parse(time.RFC3339Nano, rec["time"].(string))
		if err != nil {
			return fmt.Errorf("parse record time: %w", err)
		}
		line, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("encode record: %w", err)
		}
		values = append(values, []string{strconv.FormatInt(ts.UnixNano(), 10), string(line)})
	}
	body, err := json.Marshal(pushReq{
		Streams: []stream{{
			Stream: map[string]string{
				"service_name": service,
				"job":          "bench-analyze-smoke",
			},
			Values: values,
		}},
	})
	if err != nil {
		return err
	}
	resp, err := http.Post(lokiURL+"/loki/api/v1/push", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("loki push returned %s", resp.Status)
	}
	return nil
}

func waitForLoki(base string) {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/ready")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Fatalf("loki at %s not ready after 60s", base)
}
