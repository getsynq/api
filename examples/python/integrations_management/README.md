# Integrations management

This example shows how to manage integrations with the `IntegrationsService`
API. An integration is a connection from Coalesce Quality to one of your data
systems (a warehouse, a database or a transformation tool).

There are two scripts:

- `dbt_cloud.py` runs the full management lifecycle on a managed type: create,
  get, list, update (with `etag` optimistic concurrency), disable, enable,
  health, delete.
- `bigquery.py` shows what warehouse types add: outputs that the server
  generates (the service-account email to grant access to),
  `RefreshIntegration`, and `GetIntegrationHealth` with run history.

## Prerequisites

Create client credentials with the `Edit Integration Settings` scope in
Coalesce Quality → API settings ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)) and export them:

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
# Optional. The default is the EU endpoint, developer.synq.io.
# Other regions: api.us.synq.io for US, api.au.synq.io for AU.
export QUALITY_API_ENDPOINT=developer.synq.io
```

## Setup and running

```bash
pip install -r requirements.txt
python dbt_cloud.py
python bigquery.py
```

Both scripts create a real integration, call the API, and delete the
integration at the end, so you can run them again safely.

## Connection settings

The scripts read the connection details from the environment, with placeholder
defaults, so they run as they are. The credentials can be invalid or disabled.
Every management call (create, update, enable, disable, delete) still succeeds,
because these calls work on the stored configuration. Only the background sync
fails, and you can see that with `GetIntegrationHealth`. A successful sync there
shows that the connection works.

dbt Cloud (`dbt_cloud.py`):

| Variable | Default | Meaning |
|---|---|---|
| `DBT_CLOUD_ACCOUNT_ID` | `12345` | dbt Cloud account id |
| `DBT_CLOUD_PROJECT_ID` | `example-dbt-project` | id used to generate asset identifiers |
| `DBT_CLOUD_TOKEN` | `dbtc_disabled-demo-token` | API token (write-only, masked on reads) |
| `DBT_CLOUD_API_ENDPOINT` | `cloud.getdbt.com` | dbt Cloud API host |
| `DBT_CLOUD_JOB_IDS` | `100,200` | comma-separated job ids to track. Step 4 replaces this set, so send the full set, not only the changes |

BigQuery (`bigquery.py`):

| Variable | Default | Meaning |
|---|---|---|
| `BIGQUERY_PROJECT_ID` | `example-project` | Google Cloud project id |
| `BIGQUERY_REGION` | `EU` | project region |
| `BIGQUERY_SA_KEY` | placeholder JSON | service-account key, as a file path or inline JSON (write-only, masked on reads) |

## API concepts

- Secrets are write-only. Credential fields (tokens, passwords, keys) are masked
  (returned empty) on every read. On update, omit a secret to keep it, or send a
  new value to rotate it.
- Optimistic concurrency with `etag`. Read the `etag` from a get or list, and
  pass it back on update or delete. Then you only change the version you last
  read. A stale `etag` is rejected with `ABORTED` (HTTP 409). Omit it if the
  last write should win.
- `Capabilities` tell you which actions are valid now. For example,
  `can_refresh` is true only for warehouse types, and `can_enable` is true only
  when the integration is disabled.
- The type is fixed at creation. The `config` variant you set decides the
  integration's type, and you cannot change it on update.
