# Configuring a hosted grafana instance for e2e tests

In order to run e2e tests against an instance in hosted grafana, we need to enable auth.
You'll use scripts in the deployment_tools repo.

1. Connect to hg sql instance (use prod if your instance is in prod)

```sh
scripts/hg/hg-mysql-dev dev-us-central-0 hosted_grafana
```

1. Fetch the secret based on the slug name

```sql
MySQL [hosted_grafana]> select root_url,secret from instances where slug='fsperf';
+----------------------------------------+------------------------------------------+
| root_url                               | secret                                   |
+----------------------------------------+------------------------------------------+
| https://fsperf.grafana-dev.net | abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123 |
+----------------------------------------+------------------------------------------+
```

1. Create your test user
You'll use the password fetched in the previous step along with the 'admin' user
as credentials in the curl command.

```sh
curl -X POST https://admin:abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123@fsperf.grafana-dev.net/api/admin/users \
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

1. Set your user as an admin (if necessary)

```sh
curl -X PUT https://admin:abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123@fsperf.grafana-dev.net/api/admin/users/{USER_ID}/permissions \
  -H "Content-Type: application/json" \
  -d '{"isGrafanaAdmin": true}'
```

1. Ensure your user has org permissions

```
curl -X PUT https://admin:abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx123@fsperf.grafana-dev.net/api/access-control/users/{USER_ID}/roles\?targetOrgId=1 \
  -H "Content-Type: application/json" \
  -d '{"orgId": 1, "roleUids": []}'
```
