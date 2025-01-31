#

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

