# Integrations management

This example shows how to manage integrations with the `IntegrationsService` API.
An integration is a connection from Coalesce Quality to one of your data systems,
such as a warehouse, a database or a transformation tool.

There are two programs, and each one runs on its own:

- `dbt_cloud/` runs the full management lifecycle on a managed type:
  create, get, list, update (with `etag` optimistic concurrency), disable, enable,
  health and delete.
- `bigquery/` shows what warehouse types add: outputs that the server generates
  (the service account email to grant access to), `RefreshIntegration`, and
  `GetIntegrationHealth` with the run history.

## Prerequisites

Create client credentials with the `Edit Integration Settings` scope in
Coalesce Quality, on the API settings page ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)), and export them:

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
# Optional. The default is the EU endpoint, developer.synq.io.
# For other regions: api.us.synq.io for US, api.au.synq.io for AU.
export QUALITY_API_ENDPOINT=developer.synq.io
```

## Running

```bash
cd dbt_cloud && go run .
cd bigquery && go run .
```

Both programs create a real integration, call the API, and delete the integration
at the end. You can run them again safely.

## Connection settings

The programs read the connection details from the environment. Each value has an
example default, so the programs run without changes. The credentials can be
invalid or disabled. All management calls (create, update, enable, disable,
delete) still succeed, because they change only the stored configuration. Only
the background sync fails, and `GetIntegrationHealth` shows that failure. A failed
run there tells you that the sync itself runs.

dbt Cloud (`dbt_cloud/`):

| Variable | Default | Meaning |
|---|---|---|
| `DBT_CLOUD_ACCOUNT_ID` | `12345` | dbt Cloud account ID |
| `DBT_CLOUD_PROJECT_ID` | `example-dbt-project` | ID used to generate asset identifiers |
| `DBT_CLOUD_TOKEN` | `dbtc_disabled-demo-token` | API token (write-only, masked on reads) |
| `DBT_CLOUD_API_ENDPOINT` | `cloud.getdbt.com` | dbt Cloud API host |
| `DBT_CLOUD_JOB_IDS` | `100,200` | Comma-separated job IDs to track. Step 4 replaces this set, so always send the full set. |

BigQuery (`bigquery/`):

| Variable | Default | Meaning |
|---|---|---|
| `BIGQUERY_PROJECT_ID` | `example-project` | Google Cloud project ID |
| `BIGQUERY_REGION` | `EU` | Project region |
| `BIGQUERY_SA_KEY` | placeholder JSON | Service account key, as a file path or inline JSON (write-only, masked on reads) |

## API concepts

- Secrets are write-only. Credential fields (tokens, passwords, keys) are
  masked (returned empty) on every read. On update, omit a secret to keep it, or
  send a new value to rotate it.
- `etag` gives optimistic concurrency. Read the `etag` from a get or list call
  and send it back on update or delete. The change then applies only to the
  version you read. A stale `etag` is rejected with `ABORTED` (HTTP 409). Omit
  the `etag` and the last write wins.
- `Capabilities` tell you which actions are valid now. For example,
  `can_refresh` is true only for warehouse types, and `can_enable` is true only
  when the integration is disabled.
- The type is set at creation. The `config` variant that you fill in sets the
  integration's type, and an update cannot change it.
