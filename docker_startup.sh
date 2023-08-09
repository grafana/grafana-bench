#! /usr/bin/env sh
mkdir creds

echo $GCP_KEY_BASE64 | base64 -d >> creds/GCP-infra-manager-828bbfa6f427.json

# vault already decoding from base64 for some reason
echo $K6_GRAFANA_OPS_KEY >> creds/k6cloud_ops_grafana_ops_net
echo $K6_JEFFLEVINSLUNCH_KEY >> creds/k6cloud_jefflevinslunch_grafana_net

# execute
grafana-bench $GRAFANA_URL $GRAFANA_PORT $GRAFANA_USER $GRAFANA_PASSWORD $GRAFANA_TEST_SUITE
