# Alerts management

This example shows how to manage alert configurations with the Coalesce Quality Alerts API (v2). It creates, updates, lists, gets, toggles and deletes alerts from code.

## Prerequisites

### Client credentials

Go to Coalesce Quality ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)) and create client credentials with the `Edit Alerts` and `Read Alerts` scopes. Set them in the environment as `QUALITY_CLIENT_ID` and `QUALITY_CLIENT_SECRET`.

### Slack channel

The alerts send their notifications to Slack, so your Coalesce Quality workspace needs a connected Slack channel. Set the channel name or ID in the environment as `SLACK_CHANNEL` (for example `#alerts` or `C1234567890`).

## Environment variables

Set these environment variables:
1. `QUALITY_CLIENT_ID`: your client ID.
2. `QUALITY_CLIENT_SECRET`: your client secret.
3. `SLACK_CHANNEL`: the Slack channel that receives the alerts (for example `C1234567890`).

## Running the examples

There are two examples.

### 1. Global alert (complete lifecycle)

One program that creates, updates, toggles and deletes an alert.

```bash
cd global_alert
go run main.go
```

Or with the environment variables from a `.env` file:
```bash
export $(cat .env | xargs) && go run global_alert/main.go
```

### 2. Owned alert (self-contained)

Creates an alert with an owner, lists alerts by owner, checks the result and deletes the alert.

```bash
cd owned_alert
go run main.go
```

Or with the environment variables from a `.env` file:
```bash
export $(cat .env | xargs) && go run owned_alert/main.go
```

## Selecting entities: the trigger

An issue alert fires on the entities that its `Trigger` selects. A trigger holds the
selection in one of two forms:

- A structured `Query`, built from parts such as `WithType`. The `global_alert`
  example uses this form to match ClickHouse tables.
- A ResolverQL string. ResolverQL can express selections that a structured query
  cannot, such as lineage walks and column predicates. The `owned_alert` example
  uses `with_type("clickhouse_table")`. When you set a ResolverQL string, it takes
  precedence over the structured query.

In both cases, the created alert returns a canonical `rendered_resolver_ql`, so you
can see each selection written as ResolverQL.

## Updating an alert

There is one RPC for each kind of change:

- `Rename` changes the alert's name.
- `UpdateSettings` changes the trigger, the settings of the alert kind (for
  example the severities), or both.
- `UpdateTargets` replaces the delivery targets.
- `ToggleEnabled` enables or disables the alert.

The `global_alert` example renames the alert and then adds the `ERROR` severity
with `UpdateSettings`.

## Alert configuration

### Global alert

The example creates an alert with:
- A structured query trigger that matches ClickHouse tables.
- The `FATAL` severity at first. The update step adds `ERROR`.
- The configured Slack channel as the target.
- The FQN `example.alerts.critical-failures`, which the later steps use to find the alert.
- The ongoing strategy disabled, so an open issue does not send repeated alerts.

### Owned alert

The example creates an alert with:
- A ResolverQL trigger that matches ClickHouse tables.
- The `FATAL` severity only.
- The configured Slack channel as the target.
- The FQN `example.alerts.owned-alert`, which the later steps use to find the alert.
- An owner on the alert kind: the owner path `team.data-platform` and an ownership ID.
- The ongoing strategy disabled, so an open issue does not send repeated alerts.

## What the examples do

### Global alert (`global_alert`)

Step 1, create the alert:
- Creates an issue lifecycle alert with an FQN.
- Uses a structured trigger that selects ClickHouse tables.
- Alerts on the `FATAL` severity.
- Sends notifications to Slack.
- Prints the canonical ResolverQL form of the selection.
- Lists all alerts to check that the new alert is there.

Step 2, update the alert:
- Gets the alert by FQN with `BatchGet`.
- Renames the alert.
- Adds the `ERROR` severity with `UpdateSettings`.
- Checks that the update was applied.

Step 3, toggle the alert:
- Disables the alert.
- Checks that it is disabled.
- Enables the alert again.
- Checks that it is enabled.

Step 4, delete the alert:
- Checks that the alert exists.
- Deletes the alert by FQN.
- Checks that the alert is gone.

### Owned alert (`owned_alert`)

- Creates a new alert with an owner on the alert kind (owner path and ownership ID).
- Lists the alerts of that owner.
- Checks that the new alert is in the list.
- Checks that the owner is set correctly.
- Deletes the alert.
- Checks that the alert is gone.

This example runs on its own. It uses a different FQN (`example.alerts.owned-alert`), so it does not conflict with the global alert example.
