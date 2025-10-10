# Configuring a hosted grafana instance for e2e tests

In order to run e2e tests against an instance in hosted grafana, we need to enable auth.
You'll use scripts in the deployment_tools repo.

1. set your stack url as an environment variable
`export STACK_SLUG={YOUR_STACK_SLUG}`

2. Connect to the hosted grafana sql instance where the stack is located. (hg-dev, hg-ops, hg for prod)

```sh
scripts/hg/hg-mysql-dev dev-us-central-0 hosted_grafana
```

3. Fetch the secret based on the slug name

```sql
MySQL [hosted_grafana]> select root_url,secret from instances where slug='{STACK_SLUG}';
+----------------------------------------+------------------------------------------+
| root_url                               | secret                                   |
+----------------------------------------+------------------------------------------+
| https://{STACK_SLUG}.grafana-dev.net | abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123 |
+----------------------------------------+------------------------------------------+
```

4. Create your test user
You'll use the password fetched in the previous step along with the 'admin' user
as credentials in the curl command.

```sh
curl -X POST https://admin:abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123@{STACK_SLUG}.grafana-dev.net/api/admin/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "{YOUR_TEST_USER}",
    "email": "",
    "login": "{YOUR_TEST_USER}",
    "password": "{YOUR_SECURE_PASSWORD}",
  }'

{"id":18,"uid":"dev018jaxqdj4a","message":"User created"}
```

Take note of the ID returned ^^ for below commands:

5. Set your user as an admin (if necessary)

```sh
curl -X PUT https://admin:abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123@{STACK_SLUG}.grafana-dev.net/api/admin/users/{USER_ID}/permissions \
  -H "Content-Type: application/json" \
  -d '{"isGrafanaAdmin": true}'
```

6. Ensure your user has org permissions

```
curl -X PUT https://admin:abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123@{STACK_SLUG}.grafana-dev.net/api/access-control/users/{USER_ID}/roles\?targetOrgId=1 \
  -H "Content-Type: application/json" \
  -d '{"orgId": 1, "roleUids": []}'
```

7. Change the instance config to allow auth with username and password.

By default, the login form is disabled for hosted grafana stacks. The `gcom-dev` script is available [here](https://github.com/grafana/deployment_tools/blob/master/scripts/gcom/gcom-dev).

> Use the gcom command for the environment your stack is in. use gcom for prod, gcom-dev for dev, and gcom-ops for ops

After making this change, you'll need to wait for the pod to reboot, so the changes will take a few minutes to show.

```
gcom-dev /instances/{STACK_SLUG}/config -d 'config[auth][disable_login_form]=false'
```

> [!TIP]
> You can monitor the progress of your restarts
> `kubectl get pods -n hosted-grafana --context={SLUG_REGION} -l slug={STACK_SLUG}`
> Refer to deployment_tools documentation for [setting up your kubernetes config](https://github.com/grafana/deployment_tools/?tab=readme-ov-file#accessing-kubernetes-clusters)
