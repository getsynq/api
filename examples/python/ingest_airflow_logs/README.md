# Ingest Airflow Logs (Python / AWS Lambda)

This example demonstrates how to forward Airflow task logs from Amazon S3 to
SYNQ via the `IngestLog` REST endpoint. It is packaged as an AWS Lambda
function triggered by S3 object creation events, so logs are shipped to SYNQ
as soon as Airflow flushes them to remote storage.

## How it works

1. Airflow is configured to write task logs to S3 via
   `airflow.providers.amazon.aws.log.s3_task_handler.S3TaskHandler`.
2. Each new log file in S3 fires an `s3:ObjectCreated:*` event that invokes
   this Lambda.
3. The Lambda parses the S3 key to extract `dag_id`, `run_id`, `task_id`, and
   `attempt`, reads the log content, and POSTs it to the SYNQ IngestLog API
   as a single payload.

## Prerequisites

### Airflow configuration

Your Airflow instance must use the S3 task handler with the default
`log_filename_template` (Airflow 2.6+):

```
dag_id={{ ti.dag_id }}/run_id={{ ti.run_id }}/task_id={{ ti.task_id }}/attempt={{ try_number }}.log
```

Example `airflow.cfg`:

```ini
[logging]
remote_logging = True
remote_base_log_folder = s3://my-airflow-bucket/logs/my-env
remote_log_conn_id = my_s3_conn
```

### SYNQ ingest token

Each Airflow integration has a pre-provisioned token for uploading logs —
you don't need to create one manually. In SYNQ, go to
**Settings → Integrations**, find your Airflow integration, and click
**Manage logs upload tokens**. Copy the token from that page and use it as
`SYNQ_TOKEN` below.

Unlike the other examples in this repository, this endpoint uses the
integration's logs upload token directly as a bearer token — there is no
OAuth2 client-credentials exchange.

### AWS IAM permissions

The Lambda execution role needs:

- `s3:GetObject` on the log bucket (e.g. `arn:aws:s3:::my-airflow-bucket/*`)
- The standard `AWSLambdaBasicExecutionRole` for CloudWatch logs

## Environment variables

| Variable       | Required | Default                     | Description                                  |
| -------------- | -------- | --------------------------- | -------------------------------------------- |
| `SYNQ_TOKEN`   | yes      | —                           | Bearer token for the SYNQ IngestLog API      |
| `SYNQ_API_URL` | no       | `https://developer.synq.io` | SYNQ API base URL (`https://api.us.synq.io` for US) |
| `LOG_PREFIX`   | no       | `""`                        | S3 key prefix to strip before parsing (e.g. `logs/my-env`) |

## Deployment

1. Package `handler.py` together with `boto3` (boto3 is bundled in the Lambda
   runtime, so no `pip install` is needed — zipping `handler.py` on its own
   is sufficient).
2. Create a Lambda function with Python 3.11+ runtime and handler
   `handler.handler`.
3. Set the environment variables above.
4. Attach an S3 event notification to the log bucket:
   - Event types: `s3:ObjectCreated:*`
   - Prefix: the log prefix (e.g. `logs/my-env/`)
   - Suffix: `.log`

## Local testing (PEP 723)

`handler.py` declares its dependencies inline via
[PEP 723](https://peps.python.org/pep-0723/), so you can run it directly with
[`uv`](https://github.com/astral-sh/uv):

```bash
uv run handler.py
```

To drive the handler locally with a fake S3 event, wrap it in a small test
script that constructs a `Records` payload and calls `handler({...}, None)`.

## Troubleshooting

**`validation error: log_time: value is required`** — the Lambda should
always set `logTime.seconds` from the S3 object's `LastModified`. If you see
this error, verify the object exists and that the handler read it before
calling the API.

**`proto: syntax error (line 2:2): invalid value ...`** — the request body
must be a JSON-encoded string (i.e. the log content wrapped in quotes, not
raw text). The handler uses `json.dumps(log_content)` to produce this.

**401 `failed to exchange token`** — the `SYNQ_TOKEN` is invalid or was
issued for the wrong region. Make sure `SYNQ_API_URL` points to the region
where your Airflow integration was created.

**Log file missing in ClickHouse** — SYNQ deduplicates log rows by the
tuple `(workspace, integration_id, dag_id, task_id, log_time)`. If multiple
writes share the same `log_time`, only the last one survives. The handler
uses the S3 object's `LastModified` as `log_time` to ensure each write is
unique.
