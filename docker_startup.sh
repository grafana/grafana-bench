#! /usr/bin/env sh
mkdir creds


touch creds/GCP-infra-manager-828bbfa6f427.json
echo $GCP_KEY_BASE64 | base64 -d >> creds/GCP-infra-manager-828bbfa6f427.json
echo $K6_GRAFANA_OPS_KEY | base64 -d >> creds/k6cloud_ops_grafana_ops_net
echo $K6_JEFFLEVINSLUNCH_KEY | base64 -d >>creds/k6cloud_jefflevinslunch_grafana_net

# execute
# grafana-bench HGTest {url} {port} {username} {password} {testsuite}

grafana-bench $GRAFANA_URL $GRAFANA_PORT $GRAFANA_USER $GRAFANA_PASSWORD $GRAFANA_TEST_SUITE
