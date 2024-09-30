# jsonschema-markdown

JSON Schema missing a description, provide it using the `description` key in the root of the JSON document.

### Type: `object(?)`


---

# Definitions

## Event

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| workspace | `string` |  | string |  |  |  |  |
| event_id | `string` |  | string |  |  |  |  |
| event_time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) |  |  |  |  |
| event_type | `string` |  | `EVENT_TYPE_UNSPECIFIED` `EVENT_TYPE_PING` `EVENT_TYPE_ISSUE_CREATED` `EVENT_TYPE_ISSUE_UPDATED` `EVENT_TYPE_ISSUE_STATUS_UPDATED` `EVENT_TYPE_ISSUE_CLOSED` |  |  |  |  |
| ping | `object` |  | [synq.webhooks.v1.Ping](#synq.webhooks.v1.ping) |  |  |  |  |
| issue_created | `object` |  | [synq.webhooks.v1.IssueCreated](#synq.webhooks.v1.issuecreated) |  |  |  |  |
| issue_updated | `object` |  | [synq.webhooks.v1.IssueUpdated](#synq.webhooks.v1.issueupdated) |  |  |  |  |
| issue_status_updated | `object` |  | [synq.webhooks.v1.IssueStatusUpdated](#synq.webhooks.v1.issuestatusupdated) |  |  |  |  |
| issue_closed | `object` |  | [synq.webhooks.v1.IssueClosed](#synq.webhooks.v1.issueclosed) |  |  |  |  |
| callbacks | `array` |  | [synq.webhooks.v1.Callback](#synq.webhooks.v1.callback) |  |  |  |  |

## synq.entities.v1.AirflowDagIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| integration_id | `string` |  | string |  |  | Synq integration_id that identifies the Airflow instance |  |
| dag_id | `string` |  | string |  |  | Airflow dag_id that identifies the DAG |  |

## synq.entities.v1.AirflowTaskIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| integration_id | `string` |  | string |  |  | Synq integration_id that identifies the Airflow instance |  |
| dag_id | `string` |  | string |  |  | Airflow dag_id that identifies the DAG |  |
| task_id | `string` |  | string |  |  | Airflow task_id that identifies the task within the DAG |  |

## synq.entities.v1.BigqueryTableIdentifier

DATA PLATFORMS

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| project | `string` |  | string |  |  | BigQuery project |  |
| dataset | `string` |  | string |  |  | BigQuery dataset id |  |
| table | `string` |  | string |  |  | BigQuery table name |  |

## synq.entities.v1.ClickhouseTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| host | `string` |  | string |  |  | Clickhouse host inclusive of port |  |
| schema | `string` |  | string |  |  | Clickhouse database |  |
| table | `string` |  | string |  |  | Clickhouse table |  |

## synq.entities.v1.CustomIdentifier

CUSTOM

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| id | `string` |  | string |  |  | Id that identifies the custom entity The Id should be unique within the custom entity Identifier. |  |

## synq.entities.v1.DatabricksTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| workspace | `string` |  | string |  |  | URL of Databricks workspace |  |
| catalog | `string` |  | string |  |  | Databricks catalog |  |
| schema | `string` |  | string |  |  | Databricks schema |  |
| table | `string` |  | string |  |  | Databricks table or view |  |

## synq.entities.v1.DataproductIdentifier

DATAPRODUCT

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| id | `string` |  | string |  |  | Dataproduct id that identifies the Dataproduct |  |

## synq.entities.v1.DbtCloudNodeIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| project_id | `string` |  | string |  |  | Your dbt Cloud project id |  |
| account_id | `string` |  | string |  |  | Your dbt Cloud account id |  |
| node_id | `string` |  | string |  |  | Dbt node_id that identifies one of dbt DAG nodes (model, test, etc) |  |

## synq.entities.v1.DbtCoreNodeIdentifier

DBT

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| integration_id | `string` |  | string |  |  | Synq integration_id that identifies the dbt Core project |  |
| node_id | `string` |  | string |  |  | Dbt node_id that identifies one of dbt DAG nodes (model, test, etc) |  |

## synq.entities.v1.Identifier

Identifier is a unique reference to an entity in Synq system. Entity identifiers are designed to closely mimic identifiers used by data platforms and tools. To construct an identifier, you need to know the kind of the entity and the ids that you would normally use to identify it in the data platform or tool. For example, to identify a table in BigQuery, you would need to know the project, dataset, and table names.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| dbt_core_node | `object` |  | [synq.entities.v1.DbtCoreNodeIdentifier](#synq.entities.v1.dbtcorenodeidentifier) |  |  | Dbt node that identifies one of dbt DAG nodes (model, test, etc) in dbt Core project |  |
| dbt_cloud_node | `object` |  | [synq.entities.v1.DbtCloudNodeIdentifier](#synq.entities.v1.dbtcloudnodeidentifier) |  |  | Dbt node that identifies one of dbt DAG nodes (model, test, etc) in dbt Cloud project |  |
| bigquery_table | `object` |  | [synq.entities.v1.BigqueryTableIdentifier](#synq.entities.v1.bigquerytableidentifier) |  |  | BigQuery table identifier |  |
| snowflake_table | `object` |  | [synq.entities.v1.SnowflakeTableIdentifier](#synq.entities.v1.snowflaketableidentifier) |  |  | Snowflake table identifier |  |
| redshift_table | `object` |  | [synq.entities.v1.RedshiftTableIdentifier](#synq.entities.v1.redshifttableidentifier) |  |  | Redshift table identifier |  |
| postgres_table | `object` |  | [synq.entities.v1.PostgresTableIdentifier](#synq.entities.v1.postgrestableidentifier) |  |  | Postgres table identifier |  |
| mysql_table | `object` |  | [synq.entities.v1.MysqlTableIdentifier](#synq.entities.v1.mysqltableidentifier) |  |  | Mysql table identifier |  |
| clickhouse_table | `object` |  | [synq.entities.v1.ClickhouseTableIdentifier](#synq.entities.v1.clickhousetableidentifier) |  |  | Clickhouse table identifier |  |
| databricks_table | `object` |  | [synq.entities.v1.DatabricksTableIdentifier](#synq.entities.v1.databrickstableidentifier) |  |  | Databricks table identifier |  |
| airflow_dag | `object` |  | [synq.entities.v1.AirflowDagIdentifier](#synq.entities.v1.airflowdagidentifier) |  |  | Airflow DAG identifier |  |
| airflow_task | `object` |  | [synq.entities.v1.AirflowTaskIdentifier](#synq.entities.v1.airflowtaskidentifier) |  |  | Airflow task identifier within a given DAG |  |
| custom | `object` |  | [synq.entities.v1.CustomIdentifier](#synq.entities.v1.customidentifier) |  |  | Custom identifier to be used with all custom created entities |  |
| dataproduct | `object` |  | [synq.entities.v1.DataproductIdentifier](#synq.entities.v1.dataproductidentifier) |  |  | Dataproduct identifier |  |
| synq_path | `object` |  | [synq.entities.v1.SynqPathIdentifier](#synq.entities.v1.synqpathidentifier) |  |  | SynqPath identifier |  |

## synq.entities.v1.MysqlTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| host | `string` |  | string |  |  | Mysql host inclusive of port |  |
| schema | `string` |  | string |  |  | Mysql database |  |
| table | `string` |  | string |  |  | Mysql table |  |

## synq.entities.v1.PostgresTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| host | `string` |  | string |  |  | Postgres host inclusive of port |  |
| database | `string` |  | string |  |  | Postgres database |  |
| schema | `string` |  | string |  |  | Postgres schema |  |
| table | `string` |  | string |  |  | Postgres table |  |

## synq.entities.v1.RedshiftTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| cluster | `string` |  | string |  |  | Redshift cluster |  |
| database | `string` |  | string |  |  | Redshift database |  |
| schema | `string` |  | string |  |  | Redshift schema |  |
| table | `string` |  | string |  |  | Redshift table |  |

## synq.entities.v1.SnowflakeTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| account | `string` |  | string |  |  | Snowflake account |  |
| database | `string` |  | string |  |  | Snowflake database |  |
| schema | `string` |  | string |  |  | Snowflake schema |  |
| table | `string` |  | string |  |  | Snowflake table |  |

## synq.entities.v1.SynqPathIdentifier

SYNQ PATH

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| path | `string` |  | string |  |  | Synq path that identifies the Synq entity, needs to be one of supported paths |  |

## synq.issues.actor.v1.Actor

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| name | `string` |  | string |  |  |  |  |
| slack | `object` |  | [synq.issues.actor.v1.SlackUser](#synq.issues.actor.v1.slackuser) |  |  |  |  |
| email | `object` |  | [synq.issues.actor.v1.EmailUser](#synq.issues.actor.v1.emailuser) |  |  |  |  |
| pagerduty | `object` |  | [synq.issues.actor.v1.PagerdutyUser](#synq.issues.actor.v1.pagerdutyuser) |  |  |  |  |

## synq.issues.actor.v1.EmailUser

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| user_email | `string` |  | string |  |  |  |  |

## synq.issues.actor.v1.PagerdutyUser

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| user_id | `string` |  | string |  |  |  |  |

## synq.issues.actor.v1.SlackUser

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| user_id | `string` |  | string |  |  |  |  |

## synq.issues.commands.v1.IssuesCommand

Not to be used directly. Use the IssuesService instead when calling via API.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| workspace | `string` |  | string |  |  |  |  |
| mark_investigating | `object` |  | [synq.issues.issues.v1.MarkInvestigatingRequest](#synq.issues.issues.v1.markinvestigatingrequest) |  |  |  |  |
| mark_fixed | `object` |  | [synq.issues.issues.v1.MarkFixedRequest](#synq.issues.issues.v1.markfixedrequest) |  |  |  |  |
| mark_expected | `object` |  | [synq.issues.issues.v1.MarkExpectedRequest](#synq.issues.issues.v1.markexpectedrequest) |  |  |  |  |
| mark_no_action_needed | `object` |  | [synq.issues.issues.v1.MarkNoActionNeededRequest](#synq.issues.issues.v1.marknoactionneededrequest) |  |  |  |  |
| post_comment | `object` |  | [synq.issues.issues.v1.PostCommentRequest](#synq.issues.issues.v1.postcommentrequest) |  |  |  |  |

## synq.issues.issues.v1.MarkExpectedRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue_id | `string` |  | string |  |  | ID of the issue to mark as expected. |  |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) |  |  | Actor marking the issue as expected. |  |

## synq.issues.issues.v1.MarkFixedRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue_id | `string` |  | string |  |  | ID of the issue to mark as fixed. |  |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) |  |  | Actor marking the issue as fixed. |  |

## synq.issues.issues.v1.MarkInvestigatingRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue_id | `string` |  | string |  |  | ID of the issue to mark as investigating. |  |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) |  |  | Actor marking the issue as investigating. |  |

## synq.issues.issues.v1.MarkNoActionNeededRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue_id | `string` |  | string |  |  | ID of the issue to mark as no action needed. |  |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) |  |  | Actor marking the issue as no action needed. |  |

## synq.issues.issues.v1.PostCommentRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue_id | `string` |  | string |  |  | ID of the issue to post a comment on. |  |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) |  |  | Actor posting the comment. |  |
| comment | `string` |  | string |  |  | Comment to post. |  |

## synq.webhooks.v1.Callback

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| url | `string` |  | string |  |  |  |  |
| action_name | `string` |  | string |  |  |  |  |
| issues_command | `object` |  | [synq.issues.commands.v1.IssuesCommand](#synq.issues.commands.v1.issuescommand) |  |  |  |  |

## synq.webhooks.v1.IssueClosed

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue | `object` |  | [synq.webhooks.v1.IssueSummary](#synq.webhooks.v1.issuesummary) |  |  |  |  |

## synq.webhooks.v1.IssueCreated

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue | `object` |  | [synq.webhooks.v1.IssueSummary](#synq.webhooks.v1.issuesummary) |  |  |  |  |

## synq.webhooks.v1.IssueStatusUpdated

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue | `object` |  | [synq.webhooks.v1.IssueSummary](#synq.webhooks.v1.issuesummary) |  |  |  |  |

## synq.webhooks.v1.IssueSummary

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue_id | `string` |  | string |  |  |  |  |
| issue_group_id | `string` |  | string |  |  |  |  |
| issue_url | `string` |  | string |  |  |  |  |
| title | `string` |  | string |  |  | Summary of the issue, what happened and where. |  |
| description | `string` |  | string |  |  | Detailed description of the issue. |  |
| trigger_entity | `object` |  | [synq.webhooks.v1.IssueSummary.IssueEntity](#synq.webhooks.v1.issuesummary.issueentity) |  |  | Entity which triggered the issue. |  |
| directly_affected_entities | `array` |  | [synq.webhooks.v1.IssueSummary.IssueEntity](#synq.webhooks.v1.issuesummary.issueentity) |  |  | Entities directly affected by the issue, not considering downstream ones. |  |
| started_at | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) |  |  | Time when the issue was triggered. |  |
| ended_at | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) |  |  | Time when the issue was closed. |  |
| trigger_run_id | `string` |  | string |  |  |  |  |
| trigger_name | `string` |  | string |  |  |  |  |
| trigger_message | `string` |  | string |  |  |  |  |
| status | `string` |  | `ISSUE_STATUS_UNSPECIFIED` `ISSUE_STATUS_INVESTIGATING` `ISSUE_STATUS_EXPECTED` `ISSUE_STATUS_FIXED` `ISSUE_STATUS_NO_ACTION_NEEDED` |  |  |  |  |

## synq.webhooks.v1.IssueSummary.IssueEntity

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| name | `string` |  | string |  |  |  |  |
| type_name | `string` |  | string |  |  |  |  |
| identifier | `object` |  | [synq.entities.v1.Identifier](#synq.entities.v1.identifier) |  |  |  |  |
| folder | `string` |  | string |  |  |  |  |
| entity_url | `string` |  | string |  |  |  |  |

## synq.webhooks.v1.IssueUpdated

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| issue | `object` |  | [synq.webhooks.v1.IssueSummary](#synq.webhooks.v1.issuesummary) |  |  |  |  |

## synq.webhooks.v1.Ping

Test event sent during a webhook setup.

#### Type: `object`

| Property | Type | Required | Possible values | Deprecated | Default | Description | Examples |
| -------- | ---- | -------- | --------------- | ---------- | ------- | ----------- | -------- |
| message | `string` |  | string |  |  |  |  |


---

Markdown generated with [jsonschema-markdown](https://github.com/elisiariocouto/jsonschema-markdown).
