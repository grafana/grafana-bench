# Configuring notifications with Bench

COMING SOON

In rolling release channels you can configure notifications for slack. In order to support large repos and provide flexibility we use a combination of github CODEOWNERS file for the team and a codeowners mapping for the slack channel.

Format for codeowners

```yaml
# .github/CODEOWNERS
# <file or directory> <team>
/pdc/loki.ts @grafana/grafana-backend-services-squad
```

```yaml
# .github/codeowners-mapping.yaml
mapping:
  - slack_channel: "#grafana-backend-services-squad"
    github_team: "@grafana/grafana-backend-services-squad"
```