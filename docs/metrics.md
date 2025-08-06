
## Generating an ops metrics key

1. Go to <https://grafana-ops.com/orgs/grafana/access-policies> (where the ops instance and its data sources live)
2. Create a new Cloud Access Policy with metrics:write permissions on the ops stack
3. Generate a CAP token for this policy
4. Use the token as the PROMETHEUS_PASSWORD argument to bench
    a. To add the token to vault for use in a workflow: VAULT_INSTANCE=prod ./vault-put ci/repo/grafana/{your-repo}/prometheus_token prometheus_token={token}
5. These variables are also required:
    a. PROMETHEUS_URL: <https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push>
    b. PROMETHEUS_USER: 10428
