# Configuring a hosted Grafana instance for e2e tests

In order to run e2e tests against an instance in hosted Grafana, we need to enable auth.
You'll use scripts in the deployment_tools repo.

## Prerequisites

**Set your stack URL as an environment variable:**

```bash
export STACK_SLUG={YOUR_STACK_SLUG}
```

## Setup Process

### 1. Connect to the hosted Grafana SQL instance

Connect to the hosted Grafana SQL instance where the stack is located (hg-dev, hg-ops, hg for prod):

```sh
scripts/hg/hg-mysql-dev dev-us-central-0 hosted_grafana
```

### 2. Fetch the secret based on the slug name

```sql
MySQL [hosted_grafana]> select root_url,secret from instances where slug='{STACK_SLUG}';
+----------------------------------------+------------------------------------------+
| root_url                               | secret                                   |
+----------------------------------------+------------------------------------------+
| https://{STACK_SLUG}.grafana-dev.net | abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123 |
+----------------------------------------+------------------------------------------+
```

**Export the secret as an environment variable:**
```bash
export SLUG_SECRET={THE_SECRET_FROM_ABOVE}
```

### 3. Create your test user

You'll use the exported secret with the 'admin' user as credentials in the curl command:

```sh
curl -X POST https://admin:${SLUG_SECRET}@${STACK_SLUG}.grafana-dev.net/api/admin/users \
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

### 4. Set your user as an admin (if necessary)

```sh
curl -X PUT https://admin:${SLUG_SECRET}@${STACK_SLUG}.grafana-dev.net/api/admin/users/{USER_ID}/permissions \
  -H "Content-Type: application/json" \
  -d '{"isGrafanaAdmin": true}'
```

### 5. Ensure your user has org permissions

```sh
curl -X PUT https://admin:${SLUG_SECRET}@${STACK_SLUG}.grafana-dev.net/api/access-control/users/{USER_ID}/roles\?targetOrgId=1 \
  -H "Content-Type: application/json" \
  -d '{"orgId": 1, "roleUids": []}'
```

### 6. Enable login form for the instance

By default, the login form is disabled for hosted Grafana stacks. The `gcom-dev` script is available [here](https://github.com/grafana/deployment_tools/blob/master/scripts/gcom/gcom-dev).

> **Important:** Use the gcom command for the environment your stack is in:
>
> - `gcom` for prod
> - `gcom-dev` for dev  
> - `gcom-ops` for ops

```sh
gcom-dev /instances/{STACK_SLUG}/config -d 'config[auth][disable_login_form]=false'
```

After making this change, you'll need to wait for the pod to reboot, so the changes will take a few minutes to show.

## Monitoring Progress

You can monitor the progress of your restarts:

```sh
kubectl get pods -n hosted-grafana --context={SLUG_REGION} -l slug={STACK_SLUG}
```

> **Tip:** Refer to deployment_tools documentation for [setting up your kubernetes config](https://github.com/grafana/deployment_tools/?tab=readme-ov-file#accessing-kubernetes-clusters)
