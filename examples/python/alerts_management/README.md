# Alerts Management (Python)

This example demonstrates how to manage alert configurations using the SYNQ Alerts API. The examples show how to create, update, list, get, toggle, and delete alerts programmatically using Python.

## Prerequisites

### Python Environment

This example requires Python 3.7 or later. We recommend using a virtual environment:

```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### SYNQ Client Credentials

Head over to [SYNQ](https://app.synq.io/settings/api) to create client credentials with the `Manage Alerts` scope. Set the credentials in the environment as `SYNQ_CLIENT_ID` and `SYNQ_CLIENT_SECRET`.

### Slack Channel (for notifications)

To receive alert notifications, you'll need a Slack channel configured in your SYNQ workspace. Set the channel name in the environment as `SLACK_CHANNEL` (e.g. `C1234567890`).

## Environment Variables

Set the following environment variables in the `.env` file:
1. `SYNQ_CLIENT_ID` - Your SYNQ client ID
2. `SYNQ_CLIENT_SECRET` - Your SYNQ client secret
3. `SLACK_CHANNEL` - The Slack channel where alerts should be sent (e.g., `C1234567890`)

## Installation

1. Create and activate a virtual environment:
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

2. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

3. Configure your environment variables in the `.env` file

## Running the Examples

There are two main examples:

### 1. Global Alert (Complete Lifecycle)
Demonstrates the complete alert lifecycle in a single script: create, update, toggle, and delete.

```bash
python global_alert.py
```

### 2. Owned Alert (Self-Contained)
Creates an alert with owner information, lists by owner, verifies, and cleans up.

```bash
python owned_alert.py
```

## Alert Configuration

### Global Alert
The example creates an alert that:
- **Trigger**: Monitors ClickHouse tables for failures
- **Severity**: FATAL failures (initially), then updated to include ERROR failures
- **Target**: Sends notifications to the configured Slack channel
- **FQN**: Uses `example.alerts.critical-failures` to reference the alert
- **Ongoing Strategy**: Disabled to prevent alert spam

### Owned Alert
The example creates an alert that:
- **Trigger**: Monitors ClickHouse tables for failures
- **Severity**: FATAL failures only
- **Target**: Sends notifications to the configured Slack channel
- **FQN**: Uses `example.alerts.owned-alert` to reference the alert
- **Owner**: Demonstrates setting owner with `team.data-platform` path and `example-ownership-id`
- **Ongoing Strategy**: Disabled to prevent alert spam

## What the Examples Demonstrate

### Global Alert (`global_alert`)
A complete alert lifecycle demonstration in a single script:

**Step 1: Create Alert**
- Creates an alert configuration with FQN for easy reference
- Sets up entity query to monitor ClickHouse tables
- Configures FATAL severity monitoring
- Sets Slack as the notification target
- Lists all alerts to verify creation

**Step 2: Update Alert**
- Retrieves the alert by FQN using BatchGet
- Updates the alert to include both FATAL and ERROR severities
- Changes the alert name to reflect the update
- Validates the update succeeded

**Step 3: Toggle Alert**
- Toggles the alert off (disables it)
- Verifies the disabled state
- Re-enables the alert
- Confirms the alert is active again

**Step 4: Delete Alert**
- Verifies the alert exists before deletion
- Deletes the alert by FQN
- Confirms successful deletion

### Owned Alert (`owned_alert`)
A self-contained example demonstrating alert ownership:

- Creates a new alert with owner information (owner path and ownership ID)
- Lists alerts filtered by owner
- Verifies the alert appears in the owner's alerts list
- Confirms the owner information is correctly set
- Cleans up by deleting the alert
- Verifies deletion

**Note:** This example is self-contained and can be run independently. It uses a different FQN (`example.alerts.owned-alert`) to avoid conflicts with the global alert example.

## Project Structure

```
alerts_management/
├── auth.py                    # Authentication helper (OAuth2 token management)
├── requirements.txt           # Python dependencies
├── .env                       # Environment variables (configure before running)
├── global_alert.py            # Complete alert lifecycle example
└── owned_alert.py             # Alert with owner example
```

## Troubleshooting

### Import Errors
If you encounter import errors, ensure you've installed all dependencies:
```bash
pip install -r requirements.txt
```

### Authentication Errors
Verify that your `SYNQ_CLIENT_ID` and `SYNQ_CLIENT_SECRET` are correct and have the `Manage Alerts` scope.

### gRPC Connection Issues
Ensure you can reach `developer.synq.io:443` and that the certificate validation is working correctly.
