package coverage_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/grafana/grafana-bench/pkg/coverage"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/k6"
	"github.com/grafana/grafana-bench/pkg/openapi"
	"github.com/grafana/grafana-bench/pkg/recorder"
)

func TestCoverage(t *testing.T) {
	recoderOptions := recorder.ProxyOptions{
		Target: "httpbin.org",
	}
	recorder, err := recorder.NewProxyRecorder(recoderOptions)
	if err != nil {
		t.Fatalf("starting recorder API %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	k6Executor := k6.NewK6TestExecutor(
		log, // logger
		k6.K6ExecutorOptions{},
	)

	_, err = k6Executor.ExecTestSuite(
		context.TODO(),
		executor.TestSuite{
			Name:    "httpbin",
			BaseDir: "./testdata",
			Path:    "httpbin.js",
		},
		map[string]string{
			"HTTPS_PROXY": recorder.ProxyHost(),
		},
	)
	if err != nil {
		t.Fatalf("executing test")
	}

	recording, err := recorder.GetRecording()
	if err != nil {
		t.Fatalf("getting recording")
	}

	httpbinAPI, err := openapi.FromFile("testdata/httpbinV3.json")
	if err != nil {
		t.Fatalf("loading HTTPBIN API %v", err)
	}

	analizer, err := coverage.NewAnalizer("", "", httpbinAPI)
	if err != nil {
		t.Fatalf("loading HTTPBIN API %v", err)
	}

	analizer.Analize(recording)

	report, err := analizer.Coverage("/status")
	if err != nil {
		t.Fatalf("getting coverage %v", err)
	}

	ops, _ := httpbinAPI.GetOperations("/status/{codes}")
	expectedTotal := int32(len(ops))
	if report.Total !=  expectedTotal || report.Covered != 1 {
		t.Fatalf("expected 1/%d  got %d/%d", expectedTotal, report.Covered, report.Total)
	}
}