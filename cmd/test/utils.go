package test

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grafana/grafana-bench/pkg/grafana"	
)


func getGrafanaInstance(
	log *slog.Logger,
	url string,
	username string,
	password string,
	timeout time.Duration,
) (grafana.GrafanaInstance, string, error) {
	grafanaInstance, err := grafana.NewInstance(
		url,
		username,
		password,
		grafana.WithTimeout(timeout),
	)
	if err != nil {
		return nil, "", err
	}

	log.Info("Waiting for grafana server...", "address", grafanaInstance.Url())

	err = grafanaInstance.WaitForLiveGrafana(context.TODO())
	if err != nil {
		return nil, "", fmt.Errorf("checking Grafana is Live... %w", err)
	}
	log.Debug("Grafana server is ready!")

	grafanaVersion, err := grafanaInstance.GetGrafanaBuildVersion()
	if err != nil {
		return nil, "", fmt.Errorf("getting grafana version %w", err)
	}

	return grafanaInstance, grafanaVersion, nil

}

