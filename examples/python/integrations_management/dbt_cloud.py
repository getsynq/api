"""
Integration management lifecycle using the Coalesce Quality IntegrationsService,
with a dbt Cloud connection as the example type:

    create -> get -> list -> update (with etag) -> disable -> enable -> health -> delete

dbt Cloud is a "managed" type: Coalesce Quality stores the connection and syncs
it in the background. This script works even with an invalid or disabled dbt Cloud
token. Every management call (create, update, enable, disable, delete) still
succeeds, because these calls work on the stored configuration. Only the
background sync fails, and you can see that later with GetIntegrationHealth.

Prerequisites:
- QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET (scope: Edit Integration Settings)
- Optional QUALITY_API_ENDPOINT (defaults to developer.synq.io; api.us.synq.io
  for US, api.au.synq.io for AU)
- Optional DBT_CLOUD_* connection settings (the defaults are placeholders).
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
    dbt_cloud_conf_pb2,
)

load_dotenv()

CLIENT_ID = os.getenv("QUALITY_CLIENT_ID") or os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("QUALITY_CLIENT_SECRET") or os.getenv("SYNQ_CLIENT_SECRET")
API_ENDPOINT = os.getenv("QUALITY_API_ENDPOINT") or os.getenv("API_ENDPOINT") or "developer.synq.io"

if not CLIENT_ID or not CLIENT_SECRET:
    raise SystemExit("QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET must be set (scope: Edit Integration Settings)")

# dbt Cloud connection settings. The defaults are placeholders. The token can
# be invalid or disabled, and the management calls below still succeed.
DBT_ACCOUNT_ID = os.getenv("DBT_CLOUD_ACCOUNT_ID", "12345")
DBT_PROJECT_ID = os.getenv("DBT_CLOUD_PROJECT_ID", "example-dbt-project")
DBT_TOKEN = os.getenv("DBT_CLOUD_TOKEN", "dbtc_disabled-demo-token")
DBT_API_ENDPOINT = os.getenv("DBT_CLOUD_API_ENDPOINT", "cloud.getdbt.com")
# Tracked dbt Cloud job ids (comma-separated). Empty tracks every job the token
# can see.
DBT_JOB_IDS = [j.strip() for j in os.getenv("DBT_CLOUD_JOB_IDS", "100,200").split(",") if j.strip()]


def dbt_config(token=None, project_id=DBT_PROJECT_ID, job_ids=None):
    """Build an IntegrationConfig for dbt Cloud. Omit ``token`` to keep the
    stored secret on update (write-only secret semantics). ``job_ids`` is
    replace-semantics: pass the FULL desired set, not a delta."""
    conf = dbt_cloud_conf_pb2.DbtCloudConf(
        account_id=DBT_ACCOUNT_ID,
        project_id=project_id,
        api_endpoint=DBT_API_ENDPOINT,
        job_ids=job_ids if job_ids is not None else DBT_JOB_IDS,
    )
    if token is not None:
        conf.token = token
    return integration_pb2.IntegrationConfig(dbt_cloud=conf)


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
        print("=== Step 1: Create dbt Cloud integration ===")
        created = stub.CreateIntegration(
            svc.CreateIntegrationRequest(title="Example dbt Cloud", config=dbt_config(token=DBT_TOKEN))
        ).integration
        integration_id = created.id
        print(f"Created id={integration_id} etag={created.etag}")
        print(f"Tracked job ids: {list(created.config.dbt_cloud.job_ids)}")
        # Secret is write-only: masked (empty) on every read.
        print(f'Token in response (masked): "{created.config.dbt_cloud.token}"\n')

        # --- Step 2: Get ---
        print("=== Step 2: Get ===")
        got = stub.GetIntegration(svc.GetIntegrationRequest(integration_id=integration_id)).integration
        print(f'title="{got.title}" disabled={got.disabled} project_id="{got.config.dbt_cloud.project_id}"\n')

        # --- Step 3: List ---
        print("=== Step 3: List all integrations ===")
        listed = stub.ListIntegrations(svc.ListIntegrationsRequest())
        print(f"Workspace has {len(listed.integrations)} integration(s)\n")

        # --- Step 4: Patch the tracked job ids (with optimistic concurrency) ---
        # The server replaces the whole config, so job_ids must hold the full set
        # you want, not only the changes. Here we drop one job and add another.
        # We omit the token to keep the stored one. We pass the etag we last read,
        # so we only update the version we saw (a stale etag gives ABORTED / 409).
        patched_job_ids = ["100", "300"]  # was ["100","200"]: keep 100, drop 200, add 300
        print("=== Step 4: Patch tracked job ids ===")
        updated = stub.UpdateIntegration(
            svc.UpdateIntegrationRequest(
                integration_id=integration_id,
                title="Example dbt Cloud (updated)",
                etag=created.etag,
                config=dbt_config(token=None, job_ids=patched_job_ids),
            )
        ).integration
        print(f"Patched job ids: {DBT_JOB_IDS} -> {list(updated.config.dbt_cloud.job_ids)}")
        print(f"new etag={updated.etag}")

        # The etag from step 1 is now stale, so the server rejects it.
        try:
            stub.UpdateIntegration(
                svc.UpdateIntegrationRequest(
                    integration_id=integration_id, etag=created.etag, config=dbt_config(token=None)
                )
            )
            print("Unexpected: stale etag accepted")
        except grpc.RpcError as e:
            ok = e.code() == grpc.StatusCode.ABORTED
            print(f"Stale etag rejected with {e.code().name} ({'expected' if ok else 'unexpected'})\n")

        # --- Step 5: Disable then Enable ---
        print("=== Step 5: Disable / Enable ===")
        dis = stub.DisableIntegration(svc.DisableIntegrationRequest(integration_id=integration_id)).integration
        print(f"disabled={dis.disabled} can_enable={dis.capabilities.can_enable}")
        en = stub.EnableIntegration(svc.EnableIntegrationRequest(integration_id=integration_id)).integration
        print(f"disabled={en.disabled} can_disable={en.capabilities.can_disable}\n")

        # --- Step 6: Health ---
        # Right after create there are usually no runs yet (UNSPECIFIED). If the
        # token is invalid/disabled, a later sync surfaces ERROR here.
        print("=== Step 6: Health ===")
        health = stub.GetIntegrationHealth(
            svc.GetIntegrationHealthRequest(integration_id=integration_id)
        )
        status_name = svc.HealthStatus.Name(health.health.status)
        print(f'status={status_name} healthy={health.health.healthy} runs={len(health.runs)}\n')

        # --- Step 7: Delete (with etag) ---
        print("=== Step 7: Delete ===")
        stub.DeleteIntegration(
            svc.DeleteIntegrationRequest(integration_id=integration_id, etag=en.etag)
        )
        try:
            stub.GetIntegration(svc.GetIntegrationRequest(integration_id=integration_id))
            print("Unexpected: integration still present")
        except grpc.RpcError as e:
            print(f"Deleted {integration_id}; Get after delete -> {e.code().name}")

        print("\nDone: ran the full management lifecycle.")


if __name__ == "__main__":
    main()
