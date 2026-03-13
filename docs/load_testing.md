# Performing Load Testing on Grafana

## Load Testing vs. Benchmarking

**Benchmarking** is generally concerned with units of code. Think functions or routines. Most languages offer tools and packages for benchmarking itself. For example, in Go, we use Go benchmarks.

**Load testing** is concerned with looking at a specific path of behavior as a holistic system. For example, hitting `/api/dashboards/create` as an authenticated user and measuring database usage, performance, error rates. In this case, it doesn't help to measure the speed of the create dashboard path when we are almost certainly IO bound.

We've built out most of the tooling you need to perform load testing on your Grafana service, however, it is up to you and your team to determine what aspects of performance are important to you. For example, are you optimizing for latency or throughput?

## Configuring Your Load Testing Instance

### Setting Up Your Load Testing Environment

> **Important:** It is recommended to use a dedicated database instance for load testing.
> Running load tests against a shared database server can cause outages and should be avoided.

### Creating a Load Testing User

For creating users and setting up authentication, follow the steps in [Configuring a Grafana instance for e2e tests](configuring_an_instance.md).

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
     "grafanaUrl": "https://{YOUR_GRAFANA_URL}",
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
   export GRAFANA_URL="https://{YOUR_GRAFANA_URL}"
   export GRAFANA_ADMIN_USER="{YOUR_ADMIN_USER}"
   export GRAFANA_ADMIN_PASSWORD="{YOUR_PASSWORD}"

   export K6_CLOUD_PROJECT_ID="{YOUR_PROJECT_ID}"
   export K6_CLOUD_TOKEN="{YOUR_TOKEN}"
   export K6_CLOUD_TRACES_ENABLED=true
   export K6_CLOUD_HOST=https://ingest.k6.io
   export K6_CLOUD_TRACES_HOST={YOUR_K6_TRACES_HOST}
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
export K6_CLOUD_HOST=https://ingest.k6.io
export K6_CLOUD_TRACES_HOST={YOUR_K6_TRACES_HOST}
```

> **Note:** Set `K6_CLOUD_TRACES_HOST` to the appropriate k6 Cloud ingest endpoint for your environment.

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

## Debugging and profiling

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
