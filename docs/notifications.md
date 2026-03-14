# Setting up Slack notifications

Bench can send Slack notifications when test suites fail (or pass). Notifications are routed to teams based on who owns the tests, using GitHub CODEOWNERS and a channel mapping file.

## Prerequisites

- Test suites with a `.github/CODEOWNERS` file
- A Slack bot token set as the `SLACK_TOKEN` environment variable

## How it works

When a test run completes, bench:

1. Reads the CODEOWNERS file to find who owns the test files
2. Looks up the owning team in `codeowners-mapping.yaml` to find their Slack channel
3. Posts a summary message to that channel

By default, bench only notifies on failure. Use `--slack-passing` to also notify on success.

---

## Step 1: Create a Slack bot and generate an API token

1. Go to [api.slack.com/apps](https://api.slack.com/apps) and click **Create New App → From scratch**.
2. Give it a name (e.g. `bench`) and select your workspace.
3. In the left sidebar, go to **OAuth & Permissions**.
4. Under **Scopes → Bot Token Scopes**, add:
   - `chat:write` — to post messages
   - `channels:read` — to look up public channel IDs
   - `groups:read` — to look up private channel IDs
5. Scroll up and click **Install to Workspace**, then **Allow**.
6. Copy the **Bot User OAuth Token** (starts with `xoxb-`). This is your `SLACK_TOKEN`.

Invite the bot to each channel that should receive notifications:

```text
/invite @your-bot-name
```

Do this for every channel you add to your `codeowners-mapping.yaml`. The bot cannot post to channels it hasn't been invited to.

## Step 2: Store the token in your secrets manager

Do not put the Slack token in your config file or commit it to source control. Store it as a secret and expose it as the `SLACK_TOKEN` environment variable at runtime.

**GitHub Actions example:**

```yaml
- name: Run tests
  env:
    SLACK_TOKEN: ${{ secrets.SLACK_TOKEN }}
  run: |
    bench test \
      --suite-name my-repo/smoke-tests \
      --suite-path ./tests \
      --slack-notifications \
      --slack-codeowners-mapping .github/codeowners-mapping.yaml
```

**Local development using a `.env` file:**

```sh
# .env (never commit this file)
SLACK_TOKEN=xoxb-your-token-here
```

Bench loads `.env` automatically if it exists in the working directory.

## Step 3: Add CODEOWNERS entries for your tests

Add entries to your repository's `.github/CODEOWNERS` file mapping test paths to owning GitHub teams:

```text
# .github/CODEOWNERS
/tests/api/          @your-org/your-team
/tests/e2e/          @your-org/your-other-team
```

## Step 4: Create the codeowners mapping file

Create `codeowners-mapping.yaml` in your test suite directory. This file maps GitHub teams to Slack channel IDs.

> **Note:** Use the Slack channel ID (format: `C0123ABCDEF`), not the channel name. Channel IDs are stable even if the channel is renamed. Find the ID by right-clicking a channel in Slack and selecting **Copy link** — the ID is the last path segment.

```yaml
# codeowners-mapping.yaml
mapping:
  - github_team: "@your-org/your-team"
    slack_channel: "C0123ABCDEF"
  - github_team: "@your-org/your-other-team"
    slack_channel: "C0456GHIJKL"
```

The mapping file can be a local path or a URL. By default bench looks for `codeowners-mapping.yaml` relative to the test suite base directory.

## Step 5: Validate your setup

Before running tests, verify that the bot has access to all channels in your mapping:

```sh
bench validate \
  --check-slack-permissions \
  --slack-codeowners-mapping ./codeowners-mapping.yaml
```

The command uses the `SLACK_TOKEN` environment variable by default. You can also pass the token directly:

```sh
bench validate \
  --check-slack-permissions \
  --slack-token "$SLACK_TOKEN" \
  --slack-codeowners-mapping ./codeowners-mapping.yaml
```

Example output:

```text
+------------------+--------------------------+--------+-------+
| CHANNEL ID       | CHANNEL NAME             | STATUS | ERROR |
+------------------+--------------------------+--------+-------+
| C0123ABCDEF      | my-team-alerts           | ok!    |       |
| C0456GHIJKL      | my-other-team-alerts     | ok!    |       |
+------------------+--------------------------+--------+-------+
```

If a channel shows an error, the bot has not been invited to that channel. Run `/invite @your-bot-name` in that channel and re-validate.

## Step 6: Add Slack flags to your bench command

Add `--slack-notifications` to your `bench test` command:

```sh
bench test \
  --suite-name my-repo/smoke-tests \
  --suite-path ./tests \
  --slack-notifications \
  --slack-codeowners-mapping ./codeowners-mapping.yaml
```

The `SLACK_TOKEN` environment variable is used automatically. To also notify on passing suites:

```sh
bench test \
  --suite-name my-repo/smoke-tests \
  --suite-path ./tests \
  --slack-notifications \
  --slack-passing \
  --slack-codeowners-mapping ./codeowners-mapping.yaml
```

### Configuration file

You can also configure Slack notifications in `bench.yaml` instead of passing flags:

```yaml
# bench.yaml
slack:
  notifications: true
  passing: false           # set to true to notify on success too
  codeowners-mapping: ".github/codeowners-mapping.yaml"
```

> **Note:** Do not put `slack.token` in `bench.yaml`. Use the `SLACK_TOKEN` environment variable instead.

---

## Slack flag reference

| Flag | Default | Description |
|---|---|---|
| `--slack-notifications` | `false` | Enable Slack notifications |
| `--slack-passing` | `false` | Also notify for passing suites |
| `--slack-token` | `$SLACK_TOKEN` | Bot token (prefer env var) |
| `--slack-codeowners-mapping` | `codeowners-mapping.yaml` | Path or URL to mapping file |

---

## Related pages

- [bench test reference](bench_test.md) - Full flag reference for the test command
- [bench validate reference](bench_validate.md) - Validate command details
- [Configuration guide](configuration.md) - bench.yaml configuration
- [GitHub Actions integration](github_actions.md) - CI setup examples
