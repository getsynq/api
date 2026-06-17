# Integrations Management

This example shows how to manage **integrations** — the connections from Coalesce
Quality to your data systems (warehouses, databases, transformation tools) — using
the `IntegrationsService` API.

Two self-contained programs are included:

- **`dbt_cloud/`** — the full management lifecycle on a *managed* type:
  create → get → list → update (with `etag` optimistic concurrency) →
  disable → enable → health → delete.
- **`bigquery/`** — the extra features *warehouse* types add: server-generated
  outputs (the service-account email to grant access to), `RefreshIntegration`,
  and `GetIntegrationHealth` with run history.

## Prerequisites

Create client credentials with the **Manage Integrations** scope at
[Coalesce Quality → API settings](https://app.synq.io/settings/api) and export them:

```bash
export SYNQ_CLIENT_ID=...
export SYNQ_CLIENT_SECRET=...
# Optional; defaults to the EU endpoint developer.synq.io.
# For the US region use api.us.synq.io.
export API_ENDPOINT=developer.synq.io
```

## Running

```bash
cd dbt_cloud && go run .
cd bigquery && go run .
```

Both programs create a real integration, exercise the API, and delete it again at
the end, so they are safe to re-run.

## Connection settings

The connection details are read from the environment with illustrative defaults,
so the examples run as-is. The credentials may be **invalid or disabled** — every
management call (create / update / enable / disable / delete) still succeeds,
because they operate on the stored configuration. Only the background sync would
fail, which you can then observe via `GetIntegrationHealth`. That is the
end-to-end proof the connection is live.

dbt Cloud (`dbt_cloud/`):

| Variable | Default | Meaning |
|---|---|---|
| `DBT_CLOUD_ACCOUNT_ID` | `12345` | dbt Cloud account id |
| `DBT_CLOUD_PROJECT_ID` | `example-dbt-project` | id used to generate asset identifiers |
| `DBT_CLOUD_TOKEN` | `dbtc_disabled-demo-token` | API token (write-only, masked on reads) |
| `DBT_CLOUD_API_ENDPOINT` | `cloud.getdbt.com` | dbt Cloud API host |
| `DBT_CLOUD_JOB_IDS` | `100,200` | comma-separated job ids to track; Step 4 patches this set (replace-semantics — send the full set, not a delta) |

BigQuery (`bigquery/`):

| Variable | Default | Meaning |
|---|---|---|
| `BIGQUERY_PROJECT_ID` | `example-project` | Google Cloud project id |
| `BIGQUERY_REGION` | `EU` | project region |
| `BIGQUERY_SA_KEY` | placeholder JSON | service-account key — a file path or inline JSON (write-only, masked on reads) |

## Key API concepts

- **Secrets are write-only.** Credential fields (tokens, passwords, keys) are
  masked (returned empty) on every read. On update, omit a secret to keep it,
  send a new value to rotate it.
- **Optimistic concurrency via `etag`.** Read the `etag` from a get/list, pass it
  back on update/delete to ensure you change the version you last saw. A stale
  `etag` is rejected with `ABORTED` (HTTP 409). Omit it for last-write-wins.
- **`Capabilities`** tell you which actions are valid right now (e.g.
  `can_refresh` is true only for warehouse types, `can_enable` only when
  currently disabled).
- **Type is fixed at creation.** The populated `config` variant sets the
  integration's type; it cannot be changed on update.
