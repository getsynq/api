# Alerts management (Python)

This example shows how to manage alert configurations with the Coalesce Quality Alerts API (v2). The scripts create, update, list, get, toggle and delete alerts from Python.

## Prerequisites

### Python environment

This example needs Python 3.7 or later. We recommend a virtual environment:

```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### Client credentials

Create client credentials in Coalesce Quality ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)) with the `Edit Alerts` and `Read Alerts` scopes. Set them in the environment as `QUALITY_CLIENT_ID` and `QUALITY_CLIENT_SECRET`.

### Slack channel

The alerts send notifications to a Slack channel. Your Coalesce Quality workspace must have Slack connected. Set the channel in the environment as `SLACK_CHANNEL` (for example `C1234567890`).

## Environment variables

Set these variables in the environment or in a `.env` file:

- `QUALITY_CLIENT_ID`: your client ID.
- `QUALITY_CLIENT_SECRET`: your client secret.
- `SLACK_CHANNEL`: the Slack channel that receives the alerts (for example `C1234567890`).
- `QUALITY_API_ENDPOINT` (optional): `developer.synq.io` (EU, the default), `api.us.synq.io` (US) or `api.au.synq.io` (AU).

## Running the examples

There are two scripts.

### Global alert (full lifecycle)

Creates, updates, toggles and deletes one alert.

```bash
python global_alert.py
```

### Owned alert

Creates an alert with an owner, lists alerts by owner, checks the result and deletes the alert.

```bash
python owned_alert.py
```

## Selecting entities: the trigger

An issue alert fires on the entities that its `Trigger` selects. A trigger holds the selection in one of two forms:

- A structured `Query`, built from parts such as `WithType`. `global_alert.py` uses this form to match ClickHouse tables.
- A ResolverQL string. ResolverQL can express selections that a structured query cannot, such as lineage walks and column predicates. `owned_alert.py` uses `with_type("clickhouse_table")`. When both are set, the ResolverQL string takes precedence.

In both cases, the alert you read back has a canonical `rendered_resolver_ql`, so you can see any selection as ResolverQL.

## Updating an alert

Each kind of change has its own RPC:

- `Rename` changes the alert's name.
- `UpdateSettings` changes the trigger, the kind-specific settings (for example the severities), or both.
- `UpdateTargets` replaces the delivery targets.
- `ToggleEnabled` enables or disables the alert.

`global_alert.py` renames the alert and then adds the `ERROR` severity with `UpdateSettings`.

## Alert configuration

### Global alert

The example creates an alert with:

- Trigger: a structured query that matches ClickHouse tables.
- Severity: FATAL failures at first, then ERROR failures too.
- Target: the Slack channel from `SLACK_CHANNEL`.
- FQN: `example.alerts.critical-failures`, used to refer to the alert.
- Ongoing strategy: disabled, so an ongoing issue does not send repeated alerts.

### Owned alert

The example creates an alert with:

- Trigger: a ResolverQL selection that matches ClickHouse tables.
- Severity: FATAL failures only.
- Target: the Slack channel from `SLACK_CHANNEL`.
- FQN: `example.alerts.owned-alert`, used to refer to the alert.
- Owner: an owner path (`team.data-platform`) and an ownership ID, set on the alert kind.
- Ongoing strategy: disabled.

## What the examples do

### Global alert (`global_alert.py`)

Step 1, create the alert:

- Creates an issue lifecycle alert with an FQN.
- Sets a structured trigger that matches ClickHouse tables.
- Sets the FATAL severity.
- Sets Slack as the notification target.
- Prints the canonical ResolverQL form of the selection.
- Lists all alerts to check that the new one is there.

Step 2, update the alert:

- Gets the alert by FQN with `BatchGet`.
- Renames the alert.
- Adds the ERROR severity with `UpdateSettings`.
- Checks that the update worked.

Step 3, toggle the alert:

- Disables the alert and checks that it is disabled.
- Enables the alert again and checks that it is active.

Step 4, delete the alert:

- Checks that the alert exists.
- Deletes the alert by FQN.
- Checks that the alert is gone.

### Owned alert (`owned_alert.py`)

- Creates an alert with an owner path and an ownership ID on the alert kind.
- Lists alerts filtered by that owner.
- Checks that the new alert is in the list.
- Checks that the owner fields are set correctly.
- Deletes the alert and checks that it is gone.

This script runs on its own. It uses a different FQN (`example.alerts.owned-alert`), so it does not conflict with `global_alert.py`.

## Project structure

```
alerts_management/
├── auth.py                    # Authentication helper (OAuth2 token management)
├── requirements.txt           # Python dependencies
├── .env                       # Environment variables (create it before running)
├── global_alert.py            # Full alert lifecycle example
└── owned_alert.py             # Alert with owner example
```

## Troubleshooting

### Import errors

Check that you installed all dependencies:

```bash
pip install -r requirements.txt
```

### Authentication errors

Check that `QUALITY_CLIENT_ID` and `QUALITY_CLIENT_SECRET` are correct and that the credentials have the `Edit Alerts` and `Read Alerts` scopes. Also check that `QUALITY_API_ENDPOINT` points to the region of your workspace.

### gRPC connection issues

Check that you can reach the API endpoint on port 443 (`developer.synq.io:443` by default) and that certificate validation works.
