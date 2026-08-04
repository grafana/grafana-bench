// Seeds a local Loki with the fixture records needed to exercise
// `grafana-bench analyze` end-to-end.
//
// Usage:
//
//	go run ./test/analyze/seed -loki http://localhost:3100
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
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

type fixtureRecord struct {
	ts     time.Time
	attrs  []any
	status string
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

	// maxAttempts=2 simulates `--k6-retries 1`. Passing runs succeed on the
	// first attempt; failing runs burn the whole budget (attempts == maxAttempts)
	// so the analyzer's v2 retry rule labels them retryExhausted=true.
	const maxAttempts = 2
	start := time.Now().Add(-2 * time.Hour)
	var fixtures []fixtureRecord
	for i := 0; i < 3; i++ {
		fixtures = append(fixtures, fixtureRecord{
			ts:     start.Add(time.Duration(i) * 10 * time.Minute),
			status: "passed",
			attrs: commonAttrs(*service, *runStage, *testFile, *goodVersion,
				"passed", "", fmt.Sprintf("run-good-%d", i), 1, maxAttempts),
		})
	}
	for i := 0; i < 4; i++ {
		fixtures = append(fixtures, fixtureRecord{
			ts:     start.Add(time.Duration(i+3) * 10 * time.Minute),
			status: "failed",
			attrs: commonAttrs(*service, *runStage, *testFile, *badVersion,
				"failed",
				fmt.Sprintf("check rate expected 1 got 0.87 pid=%d at %s",
					1000+i, time.Now().Format(time.RFC3339)),
				fmt.Sprintf("run-bad-%d", i), maxAttempts, maxAttempts),
		})
	}

	values := make([][]string, 0, len(fixtures))
	for _, f := range fixtures {
		line, err := renderLogfmt(f.ts, f.attrs)
		if err != nil {
			log.Fatalf("render record: %v", err)
		}
		values = append(values, []string{strconv.FormatInt(f.ts.UnixNano(), 10), line})
	}
	if err := push(*lokiURL, *service, values); err != nil {
		log.Fatalf("push: %v", err)
	}
	fmt.Fprintf(os.Stdout, "seeded %d records into %s\n", len(fixtures), *lokiURL)
}

func commonAttrs(svc, stage, file, version, status, exitMsg, runID string, attempts, maxAttempts int) []any {
	return []any{
		"tool", "bench",
		"service", svc,
		"runStage", stage,
		"testFile", file,
		"folder", "rrc-grafana-api-tests",
		"grafanaVersion", version,
		"grafanaSlug", "k6testinstant1",
		"grafanaUrl", "k6testinstant1.grafana-dev.net",
		"status", status,
		"attempts", attempts,
		"maxAttempts", maxAttempts,
		"exitMessage", exitMsg,
		"runId", runID,
	}
}

// renderLogfmt matches exactly what bench's LogReporter produces via
// slog.NewTextHandler: `time=... level=... msg=testRun key=value ...`.
// Using slog directly (not a hand-rolled formatter) guarantees the smoke
// environment stays in lockstep with whatever slog formats today.
func renderLogfmt(ts time.Time, attrs []any) (string, error) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, ts.Format(time.RFC3339Nano))
			}
			return a
		},
	})
	logger := slog.New(h)
	logger.LogAttrs(context.Background(), slog.LevelInfo, "testRun", toSlogAttrs(attrs)...)
	line, err := bufio.NewReader(&buf).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read rendered line: %w", err)
	}
	return line[:len(line)-1], nil // strip trailing newline
}

func toSlogAttrs(kv []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		out = append(out, slog.Any(key, kv[i+1]))
	}
	return out
}

func push(lokiURL, service string, values [][]string) error {
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
