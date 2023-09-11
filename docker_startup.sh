#! /usr/bin/env sh
mkdir creds

echo $GCP_KEY_BASE64 | base64 -d >> creds/gcp.json

#echo $K6_CLOUD_TOKEN
echo $K6_CLOUD_PROJECT_ID

echo $GRAFANA_URL
echo $GRAFANA_USER
#echo $GRAFANA_PASSWORD

# execute
grafana-bench $GRAFANA_URL $GRAFANA_PORT $GRAFANA_USER $GRAFANA_PASSWORD $GRAFANA_TEST_SUITE
