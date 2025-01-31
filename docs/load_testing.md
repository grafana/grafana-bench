# Performing load testing on Grafana

## creating your instance

It is VERY important that you are aware of how and where your instance is created. These scripts WILL cause outages on shared database servers and should NEVER be run in production. In most cases it is recommended to create a dedicated load testing database instance.

### Configuring your instance

```ini
[alerting]
enabled = false

[analytics]
enabled = false

[auth]
disable_login_form = false
token_rotation_interval_mintes = 2800 # this is for session timeouts

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

[rbac]
resources_with_managed_permissions_on_creation = []
resources_with_seeded_wildcard_access = dashboard folder datasource service-account
```

### Create a load testing user

You need to login as the super admin in order to create a user with enough permissions to create everything we need. You can do this with the root secret.

### get the admin secret

Scripts in [deployment_tools](https://github.com/grafana/deployment_tools/tree/master/scripts/hg)

1. connect to the hosted_grafana server in the the appropriate region
```sh
/hg-mysql-dev dev-us-central-0 hosted_grafana
```
2. get the admin secret

```sql
select secret from instances where slug='benchloadtestingxxl';
```

### create the user


## populating instance
[Fake User Generator](https://github.com/grafana/grafana-fake-users-generator) built by @alexanderzobnin gives us two options for populating an instance.

1. using the API
2. creating sql scripts and writing to disk

Sql is quite a bit faster, however, more difficult to push to an instance of Grafana. Currently I just use the API as it's fast enough to populate an instance.

1. pull the repo on the branch jalevin/add_google_script
2. create a config file called <yourinstance>.config.json

```json
{
  "grafanaUrl": "http://localhost:3000",
  "user": "admin",
  "password": "admin",
  "token": ""
}
```

Run the script
```sh
./generateNestedFoldersAPI.js --skipPermissions --scenario google --config <yourinstance>.config.json
```

## Dedicated load testing database instance
