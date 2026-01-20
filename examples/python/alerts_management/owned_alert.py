"""
Owned Alert Example - Alert with Owner

This example demonstrates creating an alert with owner information:
1. Create an alert with owner (owner path and ownership ID)
2. List alerts filtered by owner
3. Verify the alert appears in the owner's alerts list
4. Clean up by deleting the alert

Prerequisites:
- SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET environment variables
- SLACK_CHANNEL environment variable
"""

import os
import sys
import grpc
from dotenv import load_dotenv

# Add parent directory to path to import auth module
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from auth import TokenSource, TokenAuth
from synq.alerts.services.v1 import (
    alerts_service_pb2,
    alerts_service_pb2_grpc,
)
from synq.alerts.v1 import alerts_pb2, targets_pb2
from synq.entities.v1 import entity_types_pb2
from synq.queries.v1 import query_parts_pb2, query_operand_pb2

load_dotenv()

# Load environment variables
CLIENT_ID = os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("SYNQ_CLIENT_SECRET")
SLACK_CHANNEL = os.getenv("SLACK_CHANNEL")
API_ENDPOINT = os.getenv("API_ENDPOINT", "developer.synq.io")

if CLIENT_ID is None or CLIENT_SECRET is None:
    raise Exception("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set")

if SLACK_CHANNEL is None:
    raise Exception("SLACK_CHANNEL must be set (e.g., 'C1234567890')")

# Define alert properties with owner
ALERT_FQN = "example.alerts.owned-alert"
OWNER_PATH = "team.data-platform"
OWNERSHIP_ID = "example-ownership-id"

# Initialize authorization
token_source = TokenSource(CLIENT_ID, CLIENT_SECRET, API_ENDPOINT)
auth_plugin = TokenAuth(token_source)
grpc_credentials = grpc.metadata_call_credentials(auth_plugin)

# Create and use channel to make requests
with grpc.secure_channel(
    f"{API_ENDPOINT}:443",
    grpc.composite_channel_credentials(
        grpc.ssl_channel_credentials(),
        grpc_credentials,
    ),
    options=(("grpc.default_authority", API_ENDPOINT),),
) as channel:
    grpc.channel_ready_future(channel).result(timeout=10)
    print("Connected to API...\n")

    stub = alerts_service_pb2_grpc.AlertsServiceStub(channel)
    created_alert_id = None

    # ========================================
    # CREATE ALERT WITH OWNER
    # ========================================
    print("=== Creating Alert with Owner ===")
    print(f"FQN: {ALERT_FQN}")
    print(f"Owner Path: {OWNER_PATH}")
    print(f"Ownership ID: {OWNERSHIP_ID}\n")

    # Create entity query that matches ClickHouse tables
    entity_query = alerts_pb2.EntityGroupQuery(
        parts=[
            alerts_pb2.SelectionQuery(
                parts=[
                    alerts_pb2.SelectionQuery.QueryPart(
                        with_type=query_parts_pb2.WithType(
                            types=[
                                query_parts_pb2.WithType.Type(
                                    default=entity_types_pb2.ENTITY_TYPE_CLICKHOUSE_TABLE
                                )
                            ]
                        )
                    )
                ],
                operand=query_operand_pb2.QUERY_OPERAND_AND,
            )
        ]
    )

    # Configure alert for FATAL severity failures
    alert_settings = alerts_pb2.AlertSettings(
        entity_failure=alerts_pb2.EntityFailureAlertSettings(
            severities=[alerts_pb2.EntityFailureAlertSettings.SEVERITY_FATAL],
            notify_upstream=False,
            allow_sql_test_audit_link=True,
            ongoing=alerts_pb2.OngoingAlertsStrategy(
                disabled=alerts_pb2.OngoingAlertsStrategy.Disabled()
            ),
        )
    )

    # Configure Slack target
    targets = [
        targets_pb2.AlertingTarget(
            slack=targets_pb2.SlackTarget(channel=SLACK_CHANNEL)
        )
    ]

    # Create the alert with owner information
    try:
        create_response = stub.Create(
            alerts_service_pb2.CreateRequest(
                name="Owned Alert Example",
                fqn=ALERT_FQN,
                trigger=entity_query,
                targets=targets,
                settings=alert_settings,
                owner=alerts_pb2.Alert.Owner(
                    owner_path=OWNER_PATH,
                    ownership_id=OWNERSHIP_ID,
                ),
            )
        )
        created_alert_id = create_response.alert.id
        print(f"✓ Alert created successfully: {created_alert_id}")

        if not create_response.alert.HasField("owner"):
            print("✗ Alert was created but owner was not set")
            sys.exit(1)

        print(f"  Owner Path: {create_response.alert.owner.owner_path}")
        print(f"  Ownership ID: {create_response.alert.owner.ownership_id}\n")
    except grpc.RpcError as e:
        print(f"Failed to create alert with owner: {e}")
        sys.exit(1)

    # ========================================
    # LIST ALERTS BY OWNER
    # ========================================
    print("=== Listing Alerts by Owner ===")
    print(f"Filtering by owner: {OWNER_PATH} (ownership: {OWNERSHIP_ID})\n")

    try:
        list_response = stub.List(
            alerts_service_pb2.ListRequest(
                owner=alerts_pb2.Alert.Owner(
                    owner_path=OWNER_PATH,
                    ownership_id=OWNERSHIP_ID,
                )
            )
        )

        print(f"Found {len(list_response.alerts_ids)} alert(s) for this owner:")
        for i, alert_id in enumerate(list_response.alerts_ids, 1):
            print(f"  {i}. {alert_id}")
        print()

        # Verify our alert is in the list
        if created_alert_id in list_response.alerts_ids:
            print(f"✓ Our alert '{created_alert_id}' was found in the owner's alerts list\n")
        else:
            print("✗ Our alert was not found in the owner's alerts list")
            sys.exit(1)
    except grpc.RpcError as e:
        print(f"Failed to list alerts by owner: {e}")
        sys.exit(1)

    # ========================================
    # VERIFY ALERT OWNER
    # ========================================
    print("=== Verifying Alert Owner ===")

    try:
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)]
            )
        )

        if ALERT_FQN not in batch_get_response.alerts:
            print("✗ Alert not found")
            sys.exit(1)

        alert = batch_get_response.alerts[ALERT_FQN]

        if not alert.HasField("owner"):
            print("✗ Alert has no owner")
            sys.exit(1)

        if alert.owner.owner_path != OWNER_PATH:
            print(f"✗ Owner path mismatch: expected {OWNER_PATH}, got {alert.owner.owner_path}")
            sys.exit(1)

        if alert.owner.ownership_id != OWNERSHIP_ID:
            print(f"✗ Ownership ID mismatch: expected {OWNERSHIP_ID}, got {alert.owner.ownership_id}")
            sys.exit(1)

        print("✓ Alert owner verified successfully")
        print(f"  Owner Path: {alert.owner.owner_path}")
        print(f"  Ownership ID: {alert.owner.ownership_id}\n")
    except grpc.RpcError as e:
        print(f"Failed to get alert: {e}")
        sys.exit(1)

    # ========================================
    # DELETE ALERT (CLEANUP)
    # ========================================
    print("=== Cleaning Up: Deleting Alert ===")

    try:
        stub.Delete(
            alerts_service_pb2.DeleteRequest(
                identifier=alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)
            )
        )
        print(f"✓ Alert deleted successfully: {ALERT_FQN}")
    except grpc.RpcError as e:
        print(f"Failed to delete alert: {e}")
        sys.exit(1)

    # Verify alert was deleted
    print("\n=== Verifying Deletion ===")

    try:
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)]
            )
        )

        if len(batch_get_response.alerts) == 0 or ALERT_FQN not in batch_get_response.alerts:
            print("✓ Alert successfully deleted (not found in batch get)")
        else:
            print("✗ Warning: Alert may still exist")
    except grpc.RpcError as e:
        print(f"✓ Alert no longer exists (error getting it: {e})")

    print("\nDone! Alert with owner created, verified, and cleaned up successfully.")
