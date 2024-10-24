package test

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grafana/grafana-bench/pkg/grafana"	
)


func getGrafanaInstance(
	log *slog.Logger,
	config GrafanaConfig,
) (grafana.GrafanaInstance, string, error) {
	grafanaInstance, err := grafana.NewInstance(
		config.Url,
		config.AdminUser,
		config.AdminPassword,
		grafana.WithTimeout(config.Timeout),
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

