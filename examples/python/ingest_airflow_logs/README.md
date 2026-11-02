# Ingest Airflow logs (Python, AWS Lambda)

This example forwards Airflow task logs from Amazon S3 to Coalesce Quality with
the `IngestLog` REST endpoint. It runs as an AWS Lambda function. S3 sends an
event for each new log file to an SQS queue, and the queue invokes the Lambda.
So logs reach Coalesce Quality soon after Airflow writes them to remote storage.

## How it works

1. Airflow writes task logs to S3 with
   `airflow.providers.amazon.aws.log.s3_task_handler.S3TaskHandler`.
2. Each new log file in S3 sends an `s3:ObjectCreated:*` event notification to
   an SQS queue.
3. The SQS queue invokes this Lambda through an event source mapping. The body
   of each SQS message is the S3 event notification.
4. The Lambda parses the S3 key to get `dag_id`, `run_id`, `task_id` and
   `attempt`. It reads the log content and posts it to the `IngestLog` endpoint
   as one payload.

## Prerequisites

### Airflow configuration

Your Airflow instance must use the S3 task handler with the default
`log_filename_template` (Airflow 2.6 or later):

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

### Logs upload token

Each Airflow integration has a token for uploading logs, so you do not need to
create one. In Coalesce Quality, go to **Settings → Integrations**, find your
Airflow integration, and click **Manage logs upload tokens**. Copy the token and
use it as `QUALITY_TOKEN` below.

This endpoint takes the logs upload token directly as a bearer token. The other
examples in this repository exchange OAuth2 client credentials for a token, but
this one does not.

### AWS IAM permissions

The Lambda execution role needs:

- `s3:GetObject` on the log bucket (for example `arn:aws:s3:::my-airflow-bucket/*`).
- `AWSLambdaSQSQueueExecutionRole`, to read messages from the SQS queue and write
  CloudWatch logs.

## Environment variables

| Variable          | Required | Default                     | Description |
| ----------------- | -------- | --------------------------- | ----------- |
| `QUALITY_TOKEN`   | yes      | (none)                      | The logs upload token of your Airflow integration. `SYNQ_TOKEN` also works. |
| `QUALITY_API_URL` | no       | `https://developer.synq.io` | The Coalesce Quality API base URL: `https://api.us.synq.io` for US, `https://api.au.synq.io` for AU. `SYNQ_API_URL` also works. |
| `LOG_PREFIX`      | no       | `""`                        | S3 key prefix to remove before parsing (for example `logs/my-env`). |

## Deployment

1. Zip `handler.py` on its own. The Lambda Python runtime includes `boto3`, so
   you do not need `pip install`.
2. Create a Lambda function with the Python 3.11 (or later) runtime and the
   handler `handler.handler`.
3. Set the environment variables above.
4. Create an SQS queue, and add it to the Lambda as an event source (trigger).
5. Add an S3 event notification to the log bucket that sends to the SQS queue:
   - Event types: `s3:ObjectCreated:*`
   - Prefix: the log prefix (for example `logs/my-env/`)
   - Suffix: `.log`

## Local testing (PEP 723)

`handler.py` declares its dependencies inline with
[PEP 723](https://peps.python.org/pep-0723/), so you can run it directly with
[`uv`](https://github.com/astral-sh/uv):

```bash
uv run handler.py
```

To call the handler locally with a fake event, write a small test script. It
builds an SQS event whose `Records[].body` holds a JSON-encoded S3 event, and
calls `handler({...}, None)`.

## Troubleshooting

`validation error: log_time: value is required`: the handler sets
`logTime.seconds` from the S3 object's `LastModified`. If you see this error,
check that the object exists and that the handler read it before it called the
API.

`proto: syntax error (line 2:2): invalid value ...`: the request body must be a
JSON-encoded string (the log content in quotes, not raw text). The handler calls
`json.dumps` on the log content to do this.

401 `failed to exchange token`: the token is not valid, or it belongs to a
different region. Check that `QUALITY_API_URL` points to the region where you
created your Airflow integration.

A log file is missing: a log entry with the same DAG, task, run, attempt and
`log_time` as an earlier one replaces it. The handler uses the S3 object's
`LastModified` as `log_time`, so a new version of a log file is stored as a new
entry, and a retry of the same object replaces its earlier upload.
