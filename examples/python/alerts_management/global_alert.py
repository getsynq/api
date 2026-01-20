"""
Global Alert Example - Complete Alert Lifecycle

This example demonstrates the complete alert lifecycle in a single script:
1. Create an alert for critical failures
2. Update the alert to include ERROR severity
3. Toggle the alert on/off
4. Delete the alert

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

# Define alert FQN
ALERT_FQN = "example.alerts.critical-failures"

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
    alert_id = None
    original_alert = None

    # ========================================
    # STEP 1: CREATE ALERT
    # ========================================
    print("=== Step 1: Creating Alert ===")

    # Create entity query to match ClickHouse tables
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

    # Configure alert for FATAL severity failures only
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

    try:
        create_response = stub.Create(
            alerts_service_pb2.CreateRequest(
                name="Critical Failures Alert",
                fqn=ALERT_FQN,
                trigger=entity_query,
                targets=targets,
                settings=alert_settings,
            )
        )
        alert_id = create_response.alert.id
        print(f"✓ Created alert: {alert_id}")
    except grpc.RpcError as e:
        print(f"Failed to create alert: {e}")
        sys.exit(1)

    # List all alerts and find the one we created
    print("\n--- Listing All Alerts ---")
    try:
        list_response = stub.List(alerts_service_pb2.ListRequest())
        if alert_id in list_response.alerts_ids:
            print(f"✓ Found created alert in list: {alert_id}")
    except grpc.RpcError as e:
        print(f"Failed to list alerts: {e}")

    # ========================================
    # STEP 2: UPDATE ALERT
    # ========================================
    print("\n=== Step 2: Updating Alert ===")

    # First, get the alert by FQN
    try:
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[
                    alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)
                ]
            )
        )
        if ALERT_FQN not in batch_get_response.alerts:
            print(f"No alert found with FQN: {ALERT_FQN}")
            sys.exit(1)
        original_alert = batch_get_response.alerts[ALERT_FQN]
        print(f"✓ Retrieved alert: {original_alert.id}")
    except grpc.RpcError as e:
        print(f"Failed to get alert: {e}")
        sys.exit(1)

    # Update to include both FATAL and ERROR severities
    updated_settings = alerts_pb2.EntityFailureAlertSettings()
    updated_settings.CopyFrom(original_alert.settings.entity_failure)
    updated_settings.severities.append(alerts_pb2.EntityFailureAlertSettings.SEVERITY_ERROR)

    updated_name = "Critical and Error Failures Alert"

    try:
        update_response = stub.Update(
            alerts_service_pb2.UpdateRequest(
                identifier=alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN),
                name=updated_name,
                settings=alerts_pb2.AlertSettings(entity_failure=updated_settings),
            )
        )
        updated_alert = update_response.alert
        print(f"✓ Updated alert: {updated_alert.id}")

        # Validate that updated alert has both severities
        has_fatal = alerts_pb2.EntityFailureAlertSettings.SEVERITY_FATAL in updated_alert.settings.entity_failure.severities
        has_error = alerts_pb2.EntityFailureAlertSettings.SEVERITY_ERROR in updated_alert.settings.entity_failure.severities
        
        if not has_fatal or not has_error:
            print("✗ Updated alert does not have both FATAL and ERROR severities")
            sys.exit(1)
        print("✓ Verified alert now has both FATAL and ERROR severities")
    except grpc.RpcError as e:
        print(f"Failed to update alert: {e}")
        sys.exit(1)

    # ========================================
    # STEP 3: TOGGLE ALERT (DISABLE/ENABLE)
    # ========================================
    print("\n=== Step 3: Toggling Alert ===")

    # Disable the alert
    print("\n--- Disabling Alert ---")
    try:
        stub.ToggleEnabled(
            alerts_service_pb2.ToggleEnabledRequest(
                identifier=alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN),
                is_enabled=False,
            )
        )

        # Verify disabled
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)]
            )
        )
        alert = batch_get_response.alerts[ALERT_FQN]
        if alert.is_disabled:
            print("✓ Alert is disabled")
        else:
            print("✗ Alert is still enabled")
    except grpc.RpcError as e:
        print(f"Failed to disable alert: {e}")

    # Re-enable the alert
    print("\n--- Re-enabling Alert ---")
    try:
        stub.ToggleEnabled(
            alerts_service_pb2.ToggleEnabledRequest(
                identifier=alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN),
                is_enabled=True,
            )
        )

        # Verify enabled
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)]
            )
        )
        alert = batch_get_response.alerts[ALERT_FQN]
        if not alert.is_disabled:
            print("✓ Alert is enabled")
        else:
            print("✗ Alert is still disabled")
    except grpc.RpcError as e:
        print(f"Failed to enable alert: {e}")

    # ========================================
    # STEP 4: DELETE ALERT
    # ========================================
    print("\n=== Step 4: Deleting Alert ===")

    # Verify alert exists before deletion
    try:
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)]
            )
        )
        if len(batch_get_response.alerts) == 0:
            print("Alert not found - may have been already deleted")
            sys.exit(0)
    except grpc.RpcError as e:
        print(f"Failed to verify alert: {e}")

    # Delete the alert by FQN
    try:
        stub.Delete(
            alerts_service_pb2.DeleteRequest(
                identifier=alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)
            )
        )
        print("✓ Alert deleted successfully")

        # Verify alert is deleted
        batch_get_response = stub.BatchGet(
            alerts_service_pb2.BatchGetRequest(
                identifiers=[alerts_service_pb2.AlertIdentifier(fqn=ALERT_FQN)]
            )
        )
        if len(batch_get_response.alerts) == 0:
            print("✓ Verified alert deletion (not found in batch get)")
        else:
            print("✗ Warning: Alert may still exist")
    except grpc.RpcError as e:
        print(f"Failed to delete alert: {e}")
        sys.exit(1)

    print("\n" + "=" * 40)
    print("Done! Complete alert lifecycle demonstrated:")
    print("  1. Created alert")
    print("  2. Updated alert settings")
    print("  3. Toggled alert on/off")
    print("  4. Deleted alert")
    print("=" * 40)
