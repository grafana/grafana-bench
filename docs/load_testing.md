# Performing Load Testing on Grafana

## Foreword

Congratulations for making it to the load testing milestone!

As a company, we've reached a level of success and sophistication that delivering quality software to our users depends on the performance of the code we write. This is a huge milestone and you should be proud.

## Load Testing vs. Benchmarking

**Benchmarking** is generally concerned with units of code. Think functions or routines. Most languages offer tools and packages for benchmarking itself. For example, in Go, we use Go benchmarks.

**Load testing** is concerned with looking at a specific path of behavior as a holistic system. For example, hitting `/api/dashboards/create` as an authenticated user and measuring database usage, performance, error rates. In this case, it doesn't help to measure the speed of the create dashboard path when we are almost certainly IO bound.

We've built out most of the tooling you need to perform load testing on your Grafana service, however, it is up to you and your team to determine what aspects of performance are important to you. For example, are you optimizing for latency or throughput?

If you're not sure, reach out in [#grafana-bench](https://grafanalabs.enterprise.slack.com/archives/C069CQCLDCG). We'd love to help facilitate the conversation and help you develop some performance goals.

## Configuring Your Load Testing Instance

### Setting Up Your Load Testing Environment

> **⚠️ IMPORTANT WARNING ⚠️**
>
> **It is VERY important that you are aware of how and where your instance is created. These scripts WILL cause outages on shared database servers and should NEVER be run in production environments. In most cases it is recommended to create a dedicated load testing database instance.**

### Creating Your Load Testing Database Server

We manage our Grafana database servers in deployment tools.

1. **Check out the [deployment_tools](https://github.com/grafana/deployment_tools) repo**
2. **Navigate to the terraform configuration:**
   [`terraform/databases/grafanalabs-dev/cloud_sql_hosted_grafana.tf`](https://github.com/grafana/deployment_tools/blob/d875826af5c5e8d7de55abec1c7c6cfac9a494fb/terraform/databases/grafanalabs-dev/cloud_sql_hosted_grafana.tf#L157)
3. **Add a new entry at the bottom for your new database server:**

```terraform
#make sure you update the name of the instance
module "cloud_sql_dev-us-central-0-hosted-grafana-dedicated-lt" {
  source            = "../../modules/cloud_sql"
  database_version  = "MYSQL_8_0"

  # update the name of the instance
  name              = "dev-us-central-0-hosted-grafana-dedicated-lt"
 
  # make sure the database is in the same region as your test instance
  region            = "us-central1"
  tier              = "db-n1-highmem-16"
  disk_autoresize   = true
  # need larger disk in order to provision more IOPS
  disk_size         = 500
  zone_preference   = "us-central1-a"
  high_availability = false
  private_network   = local.private_network_ops
  database_flags = [
    {
      name  = "innodb_file_per_table"
      value = "off"
    },
    {
      name  = "performance_schema"
      value = "on"
    },
    {
      # we're setting this to the max allowed by MySQL - we don't want this
      # database to be limited by the number of connections server-side
      name  = "max_connections"
      value = 32000
    },
    {
      name  = "slow_query_log"
      value = "on"
    },
    {
      name  = "log_output"
      value = "FILE"
    },
    {
      name = "table_open_cache"
      # max_connections * N where N is the max # of tables per join, plus a
      # few extra for temp tables, etc. We'll assume a max of 6 tables per join
      # as an absolute worst case scenario.
      value = 192000
    },
  ]

  # gives us the ability to inspect sql queries in the google cloud console
  insights_config = {
    query_insights_enabled  = true
    query_plans_per_minute  = 5
    query_string_length     = 2048
    record_application_tags = false
    record_client_address   = false
  }

  users = {
    "admin" = {
      password_vault_path = "secret/hosted-grafana/dev-us-central-0/cloudsql-hosted-grafana-dedicated-lt-admin-password"
    }
  }
}
```

4. **Create a PR and push**
5. **In a comment on the PR, run `atlantis plan`**
6. **Wait for this to finish, review the resulting created database and verify**
7. **Get a review**
8. **Run `atlantis apply`**
9. **Merge the PR**

### Creating Your Instance

1. **Navigate to [grafana-dev.net](https://grafana-dev.net)**
2. **Sign in using your Okta/Google credentials**
3. **This will sign you into the raintank org**
4. **Click `add stack`**
5. **Select your stack identifier and set your region to the same as your database server and click apply**
   
   <img width="1454" height="950" alt="Create stack interface" src="https://github.com/user-attachments/assets/262a8234-cced-4898-9df2-f95012f18a99" />

6. **Wait for your stack to be created and note the ID in the URL for configuring your instance**
   
   Example URL: `https://grafana-dev.com/orgs/raintank/stacks/8182`

<img width="2070" height="756" alt="Stack created confirmation" src="https://github.com/user-attachments/assets/561ffba6-1a8f-406e-b58a-5c554d700dfd" />

### Configuring Your Instance

1. **Substitute the ID of your stack into the admin URL:**
   ```
   https://admin.grafana-dev.com/orgs/raintank/stacks/{YOUR_ID}
   ```

2. **Navigate to that URL and click the edit button next to Grafana**

   <img width="1588" height="1606" alt="Grafana admin interface" src="https://github.com/user-attachments/assets/05ae88ca-7fb0-4dec-bc54-c274f4cfbe53" />

3. **Update the following config sections:**

```ini
[alerting]
enabled = false

[analytics]
enabled = false

[auth]
disable_login_form = false
token_rotation_interval_minutes = 2800 # this is for session timeouts

[database]
conn_max_lifetime = 14400
instrument_queries = true
max_idle_conn = 1200
max_open_conn = 1200

[feature_toggles]
databaseReadReplica = false
legacyImportantFeature = true
ngalert = true
panelTitleSearch = false
publicDashboards = false

[hosted_grafana]
autoscaling = false
cpu_request = 4
memory_limit = 12000Mi
memory_request = 8000Mi
replicas = 10

[log]
level = debug

#[rbac]
#resources_with_managed_permissions_on_creation = []
#resources_with_seeded_wildcard_access = dashboard folder datasource service-account
```

### Migrating the Instance to the New Database

From the deployment tools repo:

1. **[Request timed access](https://timed-access.grafana-ops.net/timed-access/access/request)**

2. **Pause the instance:**
   ```sh
   scripts/gcom/gcom-dev /instances/{YOUR_SLUG}/archive -d ""
   ```

3. **Migrate to new database server:**
   ```sh
   scripts/hg/hg-dev /instances/{YOUR_SLUG}/migrate_db -d targetDbServer={YOUR_DATABASE_SERVER_NAME}
   ```

### Creating a Load Testing User

For creating users and setting up authentication, follow the detailed steps in [Configuring a hosted Grafana instance for e2e tests](configuring_an_instance.md#setup-process).

## Populating Your Instance

The fake-user-generator tool in the simulation directory gives us two options for populating an instance:

1. **Using the API**
2. **Creating SQL scripts and writing to disk**

SQL is quite a bit faster, however, more difficult to push to an instance of Grafana. Currently we use the API as it's fast enough to populate an instance.

### Setup

1. **Navigate to the fake-user-generator directory:**
   ```sh
   cd grafana-api-tests/simulation/fake-user-generator
   ```

2. **Install dependencies:**
   ```sh
   yarn install --immutable
   yarn add ts-node -D
   ```

3. **Create a config file:**
   ```sh
   cp config.example.json config.json
   ```

4. **Edit `config.json` with your instance details:**
   ```json
   {
     "grafanaUrl": "https://{YOUR_INSTANCE}.grafana-dev.net",
     "user": "benchloadtester",
     "password": "{YOUR_SECURE_PASSWORD}",
     "token": ""
   }
   ```

5. **Run the script:**
   ```sh
   ./generateNestedFolders.ts --scenario medium --timeout 10
   ```

**Available scenarios:** `flat`, `tiny`, `small`, `medium`, `big`, `huge`

## Writing and Running Tests

### Writing Tests for Your Instance

The Grafana simulation suite provides a framework for writing load tests using k6. We use a shared library in `{root}/lib` to implement the Grafana API, with domain-specific tests organized in `simulation/src/{your_domain}`.

### Getting Started with the Simulation Suite

1. **Clone and setup the API tests repository:**
   ```sh
   git clone https://github.com/grafana/grafana-api-tests
   cd grafana-api-tests/simulation
   yarn install --immutable
   ```

2. **Build the simulation suite:**
   ```sh
   yarn build:simulation
   ```
   
   This compiles TypeScript to JavaScript in `simulation/dist`

3. **Configure environment variables:**
   
   Create a `.env.{yourinstance}` file:

   ```sh
   export GRAFANA_URL="https://{YOUR_INSTANCE}.grafana-dev.net"
   export GRAFANA_ADMIN_USER="{YOUR_ADMIN_USER}"
   export GRAFANA_ADMIN_PASSWORD="{YOUR_PASSWORD}"

   export K6_CLOUD_PROJECT_ID="{YOUR_PROJECT_ID}"
   export K6_CLOUD_TOKEN="{YOUR_TOKEN}"
   export K6_CLOUD_TRACES_ENABLED=true
   export K6_CLOUD_HOST=https://ingest.staging.k6.io
   export K6_CLOUD_TRACES_HOST=grpc-k6-api-staging-dev-us-east-0.grafana-dev.net:443
   ```

4. **Load environment variables:**
   ```sh
   source .env.{yourinstance}
   ```

### Creating Tests for Your Domain

1. **Create your domain directory:**
   ```sh
   mkdir simulation/src/{your_domain}
   ```

2. **Reference existing examples:**
   - `simulation/src/hosted_grafana/` - Examples for hosted Grafana instances
   - `simulation/src/unified_storage/` - Examples for unified storage testing

3. **Create your test files:**
   
   Follow the patterns in the existing examples to create your domain-specific load tests.

### Test execution patterns

The simulation suite supports different k6 execution patterns:

#### Constant arrival rate

```javascript
export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 100,           // requests per timeUnit
      timeUnit: '1s',      // per second
      duration: '10m',     // test duration
      preAllocatedVUs: 50, // pre-allocated virtual users
      maxVUs: 200,         // maximum virtual users
    }
  }
};
```

#### Ramping arrival rate

```javascript
export const options = {
  scenarios: {
    ramping_load: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 1000,
      stages: [
        { target: 500, duration: '30s' },
        { target: 1000, duration: '60s' },
        { target: 2000, duration: '60s' },
      ],
    }
  }
};
```

## Running tests against your instance

### Running tests from simulation/src/

Execute tests from your domain-specific directory:

```sh
# Run a specific test
k6 run -e GRAFANA_URL=$GRAFANA_URL -e GRAFANA_USERNAME=$GRAFANA_USERNAME -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD dist/tests/dashboard_by_uid_fetch_ap.js

# Run locally
k6 run --vus 50 --duration 5m -e GRAFANA_URL=$GRAFANA_URL -e GRAFANA_USERNAME=$GRAFANA_USERNAME -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD dist/tests/dashboard_by_uid_fetch_ap.js
```

### Cloud execution with k6 Cloud

For running actual load tests, use k6 Cloud:

```sh
k6 cloud -e GRAFANA_URL=$GRAFANA_URL -e GRAFANA_USERNAME=$GRAFANA_USERNAME -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD dist/tests/dashboard_by_uid_fetch_ap.js
```

## Scheduling tests

**Note: When scheduling tests, ensure they don't overlap with each other to avoid resource conflicts and inaccurate results.**

There are two ways to schedule load tests: using k6 Cloud's built-in scheduling or integrating with GitHub Actions.

### k6 Cloud scheduling

You can schedule tests directly in k6 Cloud:

1. **Run your test once to create it in k6 Cloud:**

   ```sh
   k6 cloud -e GRAFANA_URL=$GRAFANA_URL -e GRAFANA_USERNAME=$GRAFANA_USERNAME -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD dist/tests/dashboard_by_uid_fetch_ap.js
   ```

2. **Navigate to k6 Cloud dashboard** and find your test run

3. **Set up scheduling:**
   - Click on your test
   - Go to "Scheduling" tab
   - Configure frequency (daily, weekly, etc.)
   - Set timezone and specific times
   - Save the schedule

### GitHub Actions integration

#### GitHub Actions

Create `.github/workflows/load-test.yml`:

```yaml
name: Load Testing
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:
  push:
    branches: [main]

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          
      - name: Install dependencies
        run: |
          cd simulation
          yarn install --immutable
          
      - name: Build simulation suite
        run: |
          cd simulation
          yarn build:simulation
          
      - name: Run load tests
        uses: grafana/k6-action@v0.2.0
        with:
          filename: simulation/dist/tests/dashboard_by_uid_fetch_ap.js
        env:
          GRAFANA_URL: ${{ secrets.GRAFANA_URL }}
          GRAFANA_USERNAME: ${{ secrets.GRAFANA_USERNAME }}
          GRAFANA_PASSWORD: ${{ secrets.GRAFANA_PASSWORD }}
```

## Tracing and improving performance

### Viewing traces in k6 Cloud

k6 Cloud provides distributed tracing capabilities to help you understand performance bottlenecks in your load tests.

#### Enable tracing in your environment

Make sure your `.env` file includes tracing configuration:

```sh
export K6_CLOUD_TRACES_ENABLED=true
export K6_CLOUD_HOST=https://ingest.staging.k6.io
export K6_CLOUD_TRACES_HOST=grpc-k6-api-staging-dev-us-east-0.grafana-dev.net:443
```

**Important:** If your Grafana instance is in dev, make sure `K6_CLOUD_TRACES_HOST` points to the dev environment as shown above. Otherwise, it will look for traces in prod and you won't see your trace data.

#### Add instrumentation to your k6 test

Import and configure tracing in your test file:

```javascript
import tracing from 'k6/experimental/tracing';

// Configure tracing
tracing.instrumentHTTP({
  propagator: 'w3c',  // possible values: "w3c", "jaeger"
});
```

#### Run tests with tracing

Execute your tests normally - tracing will be automatically enabled:

```sh
k6 cloud -e GRAFANA_URL=$GRAFANA_URL -e GRAFANA_USERNAME=$GRAFANA_USERNAME -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD dist/tests/dashboard_by_uid_fetch_ap.js
```

#### View traces in k6 Cloud

1. **Navigate to k6 Cloud dashboard** after your test completes
2. **Click on your test run**
3. **Go to the "Traces" tab** to view distributed traces
4. **Analyze performance bottlenecks:**
   - View request waterfall charts
   - Identify slow database queries
   - Track request flow across services
   - Monitor error rates and patterns

### Application instrumentation

For detailed information on instrumenting your Grafana application code, see the [Grafana contributor guide for backend instrumentation](https://github.com/grafana/grafana/blob/main/contribute/backend/instrumentation.md).

### Local development setup

For local work, check out the [hosted-grafana](https://github.com/grafana/hosted-grafana) repo and run `common/up.sh` to spin up a Grafana stack for viewing traces while developing Grafana locally.

## Managing your load testing instance

### Monitoring instance status

Review pods:

```sh
kubectl get pods -n hosted-grafana --context=dev-us-central-0 -l slug={YOUR_SLUG}
```

View logs:

```sh
kubectl logs -n hosted-grafana --context=dev-us-central-0 -f {POD_NAME}
```

Check instance database config:

```sh
scripts/hg/hg-dev /instances/{YOUR_SLUG} | jq .database
```

### Instance operations

#### Restart instance

```sh
scripts/gcom/gcom-dev /instances/{YOUR_SLUG}/restart -d ''
```

#### Configure instance via gcom

```sh
scripts/gcom/gcom-dev /instances/{YOUR_SLUG}/config \
  -d 'config[auth][disable_login_form]=false' \
  -d 'config[auth][token_rotation_interval_minutes]=2800' \
  -d 'config[database][conn_max_lifetime]=14400' \
  -d 'config[database][instrument_queries]=true' \
  -d 'config[database][max_idle_conn]=1200' \
  -d 'config[database][max_open_conn]=1200' \
  -d 'config[hosted_grafana][autoscaling]=false' \
  -d 'config[hosted_grafana][cpu_request]=4' \
  -d 'config[hosted_grafana][memory_limit]=12000Mi' \
  -d 'config[hosted_grafana][memory_request]=8000Mi' \
  -d 'config[hosted_grafana][replicas]=10' \
  -d 'config[log][level]=debug'
```

### Restoring database backups

1. [Request timed access](https://internal-ops-us-east-0.grafana.net/timed-access/access/request)

2. Pause instance:

   ```sh
   scripts/gcom/gcom-dev /instances/{YOUR_SLUG}/archive -d ""
   ```

3. Create `~/.boto` file (ignore key values):

   ```ini
   [GSUtil]
   encryption_key=
   decryption_key=
   ```

4. Locate backup in [Google Cloud Storage](https://console.cloud.google.com/storage/browser/hg-databases)
   - Search for your slug
   - Click on the `.sql` file
   - View version history
   - Grab the generation number for the version you want

5. Copy backup (this will boot the instance after):

   ```sh
   scripts/hg/copy-old-archive/copy_archive {YOUR_SLUG} {GENERATION_NUMBER}
   ```

6. Migrate to correct database server:

   ```sh
   scripts/hg/hg-dev /instances/{YOUR_SLUG}/migrate_db -d targetDbServer={YOUR_DATABASE_SERVER}
   ```

### Create alert to prevent instance pausing

Create an alert to keep the instance active:

```sh
POST https://{YOUR_INSTANCE}.grafana-dev.net/api/ruler/grafana/api/v1/rules/{FOLDER_ID}?subtype=cortex
{
  "name": "test",
  "interval": "4h",
  "rules": [{
    "grafana_alert": {
      "title": "test",
      "condition": "C",
      "no_data_state": "NoData",
      "exec_err_state": "OK",
      "data": [
        {
          "refId": "A",
          "datasourceUid": "grafanacloud-prom",
          "queryType": "",
          "relativeTimeRange": {"from": 600, "to": 0},
          "model": {
            "refId": "A",
            "expr": "sum(rate([$__rate_interval]))",
            "range": false,
            "instant": true,
            "editorMode": "builder",
            "legendFormat": "__auto"
          }
        }
      ],
      "is_paused": false,
      "notification_settings": {"receiver": "grafana-default-email"}
    },
    "for": "20h",
    "annotations": {},
    "labels": {"service_name": ""}
  }]
}
```

## Debugging and profiling

### Connect to database directly

```sh
scripts/hg/hg-mysql-dev dev-us-central-0 hg_{YOUR_SLUG}
```

### SSH into pod

```sh
kubectl -n hosted-grafana exec -ti {POD_NAME} -- sh
```

### Local profiling setup

For local development, disable race detector in Makefile and add profiling:

```makefile
run-go: ## Build and run web server immediately.
 $(GO) run $(if $(GO_BUILD_TAGS),-build-tags=$(GO_BUILD_TAGS)) \
  ./pkg/cmd/grafana -- server -packaging=dev cfg:app_mode=development
```

Environment variables:

```sh
export GF_DIAGNOSTICS_PROFILING_ENABLED=1
export GF_DIAGNOSTICS_PROFILING_ADDR=0.0.0.0
export GF_DIAGNOSTICS_PROFILING_PORT=6000
```

View profiling data:

```sh
go tool pprof -http=:6060 http://localhost:6000/debug/pprof/heap
```
