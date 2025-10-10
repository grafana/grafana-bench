# Configuring Notifications with Bench

You can configure notifications for Slack anywhere, including rolling release channels. In order to support large repos and provide flexibility we use a combination of GitHub CODEOWNERS file for the team and a codeowners mapping for the Slack channel.

## Prerequisites

To enable notifications, you'll need:
- A Slack token set as the `SLACK_TOKEN` environment variable
- Use the `--slack-notifications` flag when running bench commands

## CODEOWNERS Format

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