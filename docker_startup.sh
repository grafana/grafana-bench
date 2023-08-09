#! /usr/bin/env sh
mkdir creds

echo $GCP_KEY_BASE64 | base64 -d >> creds/GCP-infra-manager-828bbfa6f427.json

K6_CLOUD_TOKEN=$(echo $K6_CLOUD_TOKEN | base64 -d)
echo $K6_CLOUD_TOKEN
echo $K6_CLOUD_PROJECT_ID

# execute
grafana-bench $GRAFANA_URL $GRAFANA_PORT $GRAFANA_USER $GRAFANA_PASSWORD $GRAFANA_TEST_SUITE
