package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/unknwon/log"
)

type TestRunner struct {
	Log *slog.Logger
	// location of test files on disk
	TestDir string
	// location of the .version file within the test repo
	TestVersionFilepath string
	K6CloudToken        string
	K6CloudProjectID    string
	GrafanaInstance     *provisioner.VMInstance

	runIdentifier string
}

func NewTestRunner(ctx context.Context, log *slog.Logger, testDir string, testVersionFilepath string, k6CloudProjectId, k6CloudToken string, grafanaInstance *provisioner.VMInstance) *TestRunner {

	return &TestRunner{
		Log:                 log,
		TestDir:             testDir,
		TestVersionFilepath: testVersionFilepath,
		K6CloudToken:        k6CloudToken,
		K6CloudProjectID:    k6CloudProjectId,
		GrafanaInstance:     grafanaInstance,
	}
}

func (t *TestRunner) Exec(testType TestType) error {
	ctx := context.Background()
	log := t.Log.With("svc", "boot-test-runner")

	// TODO implement a timeout of some sort
	WaitForLiveGrafana(ctx, log, t.GrafanaInstance.ServiceAddress())

	grafanaVersion, err := provisioner.GetGrafanaBuildVersion(t.GrafanaInstance)
	if err != nil {
		t.Log.Error("error getting grafana version", "err", err)
		return fmt.Errorf("Error getting grafana version. exiting.. err: %w", err)
	}

	testVersion, err := t.GetShortTestRevisionFromCompiled(t.TestVersionFilepath)
	if err != nil {
		return err
	}

	t.runIdentifier = NewRunIdentifier(testType.Name(), grafanaVersion, testVersion)
	t.Log.Info("suite identifier", "identifier", t.runIdentifier)

	t.Log = log.With("svc", fmt.Sprintf("%s-test-runner", testType.Name()))

	switch testType {
	case SmokeTest:
		return t.Smoke()
	case LoadTest:
		return t.Load()
	}

	return nil
}

// GetNewSuitedentifier creates an identifier to be used for
// building dashboards in hosted grafana
//
// smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
// load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func NewRunIdentifier(testType, grafanaVersion, testVersion string) string {
	// {type}-{time}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-api-tests-%s-graf-%s",
		testType,
		time.Now().UTC().Format("15:04:05"),
		testVersion,
		grafanaVersion,
	)
}

// read .version from test directory
func (t *TestRunner) GetShortTestRevisionFromCompiled(testVersionFilepath string) (string, error) {
	bytes, err := os.ReadFile(testVersionFilepath)

	if err == nil {
		return strings.TrimSpace(string(bytes)), nil
	}

	if os.IsNotExist(err) {
		t.Log.Warn(fmt.Sprintf("No version file specified at %s", testVersionFilepath))
		return "UNKNOWN", nil
	}

	return "", err
}

// Wait for the server to start up
func WaitForLiveGrafana(ctx context.Context, log *log.Slog, grafanaAddress string) {
	for {
		if IsLive(log, grafanaAddress) {
			log.Info("Grafana server is ready!")
			break
		}
		log.Info("Waiting for grafana server...", "address", grafanaAddress)
		time.Sleep(time.Second)
	}
}

func IsLive(log *slog.Logger, address string) bool {
	_, err := net.Dial("tcp", address)
	if err != nil {
		log.Info("Checking isLive...", "error", err)
	}
	return err == nil
}
