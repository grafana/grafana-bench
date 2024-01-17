package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
)

type TestRunner struct {
	Log              *slog.Logger
	Type             TestType
	TestRevision     string
	Tests            []string
	K6CloudToken     string
	K6CloudProjectID string
	GrafanaInstance  *provisioner.VMInstance
	MachineSpec      string
}

func NewTestRunner(
	log *slog.Logger,
	testType TestType,
	tests []string,
	testRevision string,
	k6CloudProjectId,
	k6CloudToken string,
	grafanaInstance *provisioner.VMInstance,
	machineSpec string,
) *TestRunner {

	return &TestRunner{
		Log:              log,
		Type:             testType,
		Tests:            tests,
		TestRevision:     testRevision,
		K6CloudToken:     k6CloudToken,
		K6CloudProjectID: k6CloudProjectId,
		GrafanaInstance:  grafanaInstance,
		MachineSpec:      machineSpec,
	}
}

func (t *TestRunner) Exec(ctx context.Context) error {
	log := t.Log.With("svc", "boot-test-runner")

	// TODO implement a timeout of some sort
	WaitForLiveGrafana(ctx, log, t.GrafanaInstance.ServiceAddress())

	grafanaVersion, err := provisioner.GetGrafanaBuildVersion(t.GrafanaInstance)
	if err != nil {
		return fmt.Errorf("getting grafana version %w", err)
	}


	runIdentifier := NewRunIdentifier(t.Type.Name(), grafanaVersion, t.TestRevision)
	t.Log.Info("suite identifier", "identifier", runIdentifier)

	t.Log = log.With("svc", fmt.Sprintf("%s-test-runner", t.Type.Name()))

	envVars := map[string]string{
		"MACHINE_SPEC":        t.MachineSpec,
		"TEST_SUITE_REVISION": t.TestRevision,
		"GT_URL":              t.GrafanaInstance.SchemeServiceAddress(),
		"GT_USERNAME":         t.GrafanaInstance.ServiceUser,
		"GT_PASSWORD":         t.GrafanaInstance.ServicePassword,
		"K6_CLOUD_TOKEN":      t.K6CloudToken,
		"K6_CLOUD_PROJECT_ID": t.K6CloudProjectID,
	}

	// run the tests
	for _, testFile := range t.Tests {
		t.Log.Info("running test file", "file", testFile)
		envVars["SCENARIO_NAME"] = getScenarioName(testFile)

		args := []string{"run", testFile}

		if t.Type == LoadTest {
			args = append(args, "--out", "cloud")
		}

		cmd := exec.Command("k6", args...)

		// k6 run tests/tests/dashboards.js ...args
		err = utils.ExecStdoutWithEnv(cmd, envVars)
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				cmdString := "k6 " + strings.Join(cmd.Args, " ")
				t.Log.Warn("k6 command exited with err", "status", exitError.ExitCode(), "error", err, "testFile", testFile, "cmd", cmdString)
			}
		}
	}

	return nil
}

// GetNewSuitedentifier creates an identifier to be used for
// building dashboards in hosted grafana
//
// smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
// load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func NewRunIdentifier(testType, grafanaVersion, testRevision string) string {
	// {type}-{time}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-api-tests-%s-graf-%s",
		testType,
		time.Now().UTC().Format("15:04:05"),
		testRevision,
		grafanaVersion,
	)
}

// Wait for the server to start up
func WaitForLiveGrafana(ctx context.Context, log *slog.Logger, grafanaAddress string) {
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

// we expect scenarios to be named like the file
// tests/dashboards/dashboard_create.js -> dashboardCreate
func getScenarioName(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(filename, "_")
	for i, p := range parts {
		// don't capitalize the first word
		if i == 0 {
			continue
		}
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}