"""
Warehouse-specific IntegrationsService features, using BigQuery as the example:

    create -> read generated outputs -> refresh -> health (+ run history) -> delete

Unlike dbt Cloud, a warehouse integration is "refreshable" (Capabilities.
can_refresh == True) and the server derives read-only Outputs from the supplied
credentials (for BigQuery, the service-account email to grant dataset access to).

The service-account key can be invalid for this demo: create still succeeds and
the management calls work end to end. A bad key simply surfaces later as an
unhealthy status from GetIntegrationHealth.

Prerequisites:
- SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET (scope: Manage Integrations)
- Optionally BIGQUERY_* connection settings (defaults are illustrative).
"""

import os
import sys

import grpc
from dotenv import load_dotenv

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__))))

from auth import TokenSource, TokenAuth
from synq.integrations.v1 import (
    integrations_service_pb2 as svc,
    integrations_service_pb2_grpc as svc_grpc,
    integration_pb2,
    bigquery_conf_pb2,
)

load_dotenv()

CLIENT_ID = os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("SYNQ_CLIENT_SECRET")
API_ENDPOINT = os.getenv("API_ENDPOINT", "developer.synq.io")

if not CLIENT_ID or not CLIENT_SECRET:
    raise SystemExit("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set (scope: Manage Integrations)")

# BigQuery connection settings. Defaults are illustrative placeholders.
BQ_PROJECT_ID = os.getenv("BIGQUERY_PROJECT_ID", "example-project")
BQ_REGION = os.getenv("BIGQUERY_REGION", "EU")


def load_service_account_key():
    """Return the BigQuery service-account key JSON. BIGQUERY_SA_KEY may be a
    file path or inline JSON; an invalid key still lets management calls work."""
    v = os.getenv("BIGQUERY_SA_KEY")
    if not v:
        return '{"type":"service_account","project_id":"example-project"}'
    if os.path.isfile(v):
        with open(v) as f:
            return f.read()
    return v


def main():
    token_source = TokenSource(CLIENT_ID, CLIENT_SECRET, API_ENDPOINT)
    credentials = grpc.composite_channel_credentials(
        grpc.ssl_channel_credentials(),
        grpc.metadata_call_credentials(TokenAuth(token_source)),
    )
    with grpc.secure_channel(
        f"{API_ENDPOINT}:443",
        credentials,
        options=(("grpc.default_authority", API_ENDPOINT),),
    ) as channel:
        grpc.channel_ready_future(channel).result(timeout=10)
        print(f"Connected to {API_ENDPOINT}\n")
        stub = svc_grpc.IntegrationsServiceStub(channel)

        # --- Step 1: Create ---
        print("=== Step 1: Create BigQuery integration ===")
        config = integration_pb2.IntegrationConfig(
            bigquery=bigquery_conf_pb2.BigQueryCloudConf(
                project_id=BQ_PROJECT_ID,
                region=BQ_REGION,
                service_account_key=load_service_account_key(),
                # datasets left empty: discover all visible datasets.
            )
        )
        created = stub.CreateIntegration(
            svc.CreateIntegrationRequest(title="Example BigQuery", config=config)
        ).integration
        integration_id = created.id
        print(f"Created id={integration_id} etag={created.etag}")
        print(f'Service account key in response (masked): "{created.config.bigquery.service_account_key}"\n')

        # --- Step 2: Generated outputs + capabilities ---
        # Outputs are read-only values the server derives on create. For BigQuery
        # that is the service-account email you grant dataset access to.
        print("=== Step 2: Outputs & capabilities ===")
        print(f'Service account to grant access: "{created.outputs.bigquery.service_account_email}"')
        caps = created.capabilities
        print(f"can_refresh={caps.can_refresh} can_disable={caps.can_disable} can_delete={caps.can_delete}\n")

        # --- Step 3: Refresh ---
        # Valid only when Capabilities.can_refresh is true (warehouse types).
        print("=== Step 3: Trigger refresh ===")
        if caps.can_refresh:
            stub.RefreshIntegration(svc.RefreshIntegrationRequest(integration_id=integration_id))
            print("Refresh enqueued\n")
        else:
            print("Type does not support refresh\n")

        # --- Step 4: Health + run history ---
        # Omitting pagination returns a bounded recent window (last 7 days).
        print("=== Step 4: Health ===")
        health = stub.GetIntegrationHealth(
            svc.GetIntegrationHealthRequest(integration_id=integration_id)
        )
        print(f'status={svc.HealthStatus.Name(health.health.status)} '
              f'healthy={health.health.healthy} message="{health.health.message}"')
        for run in health.runs:
            print(f'  run {run.run_id} status={svc.HealthStatus.Name(run.status)} "{run.message}"')
        print()

        # --- Step 5: Delete ---
        print("=== Step 5: Delete ===")
        stub.DeleteIntegration(svc.DeleteIntegrationRequest(integration_id=integration_id))
        try:
            stub.GetIntegration(svc.GetIntegrationRequest(integration_id=integration_id))
            print("Unexpected: integration still present")
        except grpc.RpcError as e:
            print(f"Deleted {integration_id}; Get after delete -> {e.code().name}")

        print("\nDone: warehouse lifecycle (outputs, refresh, health) exercised end to end.")


if __name__ == "__main__":
    main()
