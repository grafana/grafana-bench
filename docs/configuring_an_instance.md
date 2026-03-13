# Configuring a Grafana instance for e2e tests

In order to run e2e tests against a Grafana instance, you need a test user with appropriate
permissions and the login form enabled.

## Prerequisites

You need:
- A running Grafana instance
- Admin credentials for that instance

## Setup Process

### 1. Create your test user

Use the Grafana Admin API to create a test user:

```sh
curl -X POST https://admin:${ADMIN_PASSWORD}@${GRAFANA_URL}/api/admin/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "{YOUR_TEST_USER}",
    "email": "",
    "login": "{YOUR_TEST_USER}",
    "password": "{YOUR_SECURE_PASSWORD}"
  }'
```

**Response:**

```json
{"id":18,"uid":"dev018jaxqdj4a","message":"User created"}
```

> **Note:** Take note of the ID returned for the commands below.

### 2. Set your user as an admin (if necessary)

```sh
curl -X PUT https://admin:${ADMIN_PASSWORD}@${GRAFANA_URL}/api/admin/users/{USER_ID}/permissions \
  -H "Content-Type: application/json" \
  -d '{"isGrafanaAdmin": true}'
```

### 3. Ensure your user has org permissions

```sh
curl -X PUT https://admin:${ADMIN_PASSWORD}@${GRAFANA_URL}/api/access-control/users/{USER_ID}/roles\?targetOrgId=1 \
  -H "Content-Type: application/json" \
  -d '{"orgId": 1, "roleUids": []}'
```

### 4. Enable the login form

If your Grafana instance has the login form disabled, enable it in your `grafana.ini` or
via the Grafana API:

```ini
[auth]
disable_login_form = false
```

After making this change, restart Grafana for it to take effect.

## Next Steps

- [Writing K6 API Tests](writing_k6_api_tests.md)
- [Writing Playwright Tests](writing_pw_tests.md)
- [GitHub Actions Integration](github_actions.md)
