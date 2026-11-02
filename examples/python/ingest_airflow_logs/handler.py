# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "boto3",
# ]
# ///

import json
import os
import re
import urllib.request
import urllib.error
from urllib.parse import unquote_plus, quote

import boto3

S3 = boto3.client("s3")

# The Airflow integration's logs upload token. SYNQ_TOKEN is accepted as an alias.
QUALITY_TOKEN = os.environ.get("QUALITY_TOKEN") or os.environ.get("SYNQ_TOKEN")
if not QUALITY_TOKEN:
    raise RuntimeError("QUALITY_TOKEN must be set")
# https://developer.synq.io (EU), https://api.us.synq.io (US) or
# https://api.au.synq.io (AU). SYNQ_API_URL is accepted as an alias.
QUALITY_API_URL = (
    os.environ.get("QUALITY_API_URL") or os.environ.get("SYNQ_API_URL") or "https://developer.synq.io"
)
LOG_PREFIX = os.environ.get("LOG_PREFIX", "")

# Pattern: dag_id=<v>/run_id=<v>/task_id=<v>/attempt=<n>.log
LOG_PATH_RE = re.compile(
    r"dag_id=(?P<dag_id>[^/]+)/run_id=(?P<run_id>[^/]+)/task_id=(?P<task_id>[^/]+)/attempt=(?P<attempt>\d+)\.log$"
)


def handler(event, _):
    """Read Airflow log objects from S3 and forward them to the Coalesce Quality API.

    An SQS event source mapping invokes this handler. The body of each SQS
    record is a JSON-encoded S3 event notification with its own Records list.
    """
    total = 0
    for sqs_record in event.get("Records", []):
        s3_event = json.loads(sqs_record["body"])
        for record in s3_event.get("Records", []):
            bucket = record["s3"]["bucket"]["name"]
            key = unquote_plus(record["s3"]["object"]["key"])

            log_path = key
            if LOG_PREFIX and log_path.startswith(LOG_PREFIX):
                log_path = log_path[len(LOG_PREFIX) :]

            m = LOG_PATH_RE.search(log_path)
            if not m:
                print(f"Skipping {key}: does not match expected log path pattern")
                continue

            dag_id = m.group("dag_id")
            task_id = m.group("task_id")
            run_id = m.group("run_id")
            attempt = m.group("attempt")

            obj = S3.get_object(Bucket=bucket, Key=key)
            body = obj["Body"].read().decode("utf-8")

            # The object's LastModified time is the log time. A retry of the same
            # object sends the same log_time, so the server replaces the row
            # instead of adding a second one.
            log_time = int(obj["LastModified"].timestamp())
            url = (
                f"{QUALITY_API_URL}/api/ingest/airflow/v1"
                f"/{quote(dag_id, safe='')}"
                f"/{quote(task_id, safe='')}"
                f"/{quote(run_id, safe='')}"
                f"/logs?attempt={attempt}&logTime.seconds={log_time}&logTime.nanos=0"
            )

            payload = json.dumps(body).encode("utf-8")

            req = urllib.request.Request(
                url,
                data=payload,
                headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {QUALITY_TOKEN}",
                },
                method="POST",
            )

            try:
                with urllib.request.urlopen(req) as resp:
                    print(f"Forwarded {key} -> {resp.status}")
            except urllib.error.HTTPError as exc:
                print(f"Failed to forward {key}: {exc.code} {exc.read().decode()}")
                raise

            total += 1

    return {"statusCode": 200, "body": f"Processed {total} records"}
