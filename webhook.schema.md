# jsonschema-markdown

JSON Schema missing a description, provide it using the `description` key in the root of the JSON document.

### Type: `object(?)`


---

# Definitions

## Event

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| workspace | `string` |  | string |  |
| event_id | `string` |  | string |  |
| event_time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) |  |
| event_type | `string` |  | `EVENT_TYPE_UNSPECIFIED` `EVENT_TYPE_PING` `EVENT_TYPE_ISSUE_CREATED` `EVENT_TYPE_ISSUE_UPDATED` `EVENT_TYPE_ISSUE_STATUS_UPDATED` `EVENT_TYPE_ISSUE_CLOSED` `EVENT_TYPE_INCIDENT_OPEN` `EVENT_TYPE_INCIDENT_CLOSED` `EVENT_TYPE_INCIDENT_CANCELLED` |  |
| ping | `object` |  | [Ping](#ping) |  |
| issue_created | `object` |  | [IssueCreated](#issuecreated) |  |
| issue_updated | `object` |  | [IssueUpdated](#issueupdated) |  |
| issue_status_updated | `object` |  | [IssueStatusUpdated](#issuestatusupdated) |  |
| issue_closed | `object` |  | [IssueClosed](#issueclosed) |  |
| incident_open | `object` |  | [IncidentOpen](#incidentopen) |  |
| incident_closed | `object` |  | [IncidentClosed](#incidentclosed) |  |
| incident_cancelled | `object` |  | [IncidentCancelled](#incidentcancelled) |  |
| callbacks | `array` |  | [Callback](#callback) |  |

## synq.entities.v1.AirflowDagIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| integration_id | `string` |  | string | SYNQ integration_id that identifies the Airflow instance |
| dag_id | `string` |  | string | Airflow dag_id that identifies the DAG |

## synq.entities.v1.AirflowTaskIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| integration_id | `string` |  | string | SYNQ integration_id that identifies the Airflow instance |
| dag_id | `string` |  | string | Airflow dag_id that identifies the DAG |
| task_id | `string` |  | string | Airflow task_id that identifies the task within the DAG |

## synq.entities.v1.BigqueryTableIdentifier

DATA PLATFORMS

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| project | `string` |  | string | BigQuery project |
| dataset | `string` |  | string | BigQuery dataset id |
| table | `string` |  | string | BigQuery table name |

## synq.entities.v1.ClickhouseTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Clickhouse hostname without port |
| schema | `string` |  | string | Clickhouse database |
| table | `string` |  | string | Clickhouse table |

## synq.entities.v1.CustomIdentifier

CUSTOM

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| id | `string` |  | string | Id that identifies the custom entity The Id should be unique within the custom entity Identifier. |

## synq.entities.v1.DatabricksTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| workspace | `string` |  | string | URL of Databricks workspace |
| catalog | `string` |  | string | Databricks catalog |
| schema | `string` |  | string | Databricks schema |
| table | `string` |  | string | Databricks table or view |

## synq.entities.v1.DataproductIdentifier

DATAPRODUCT

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| id | `string` |  | string | Dataproduct id that identifies the Dataproduct |

## synq.entities.v1.DbtCloudNodeIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| project_id | `string` |  | string | Your dbt Cloud project id |
| account_id | `string` |  | string | Your dbt Cloud account id |
| node_id | `string` |  | string | Dbt node_id that identifies one of dbt DAG nodes (model, test, etc) |

## synq.entities.v1.DbtCoreNodeIdentifier

DBT

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| integration_id | `string` |  | string | SYNQ integration_id that identifies the dbt Core project |
| node_id | `string` |  | string | Dbt node_id that identifies one of dbt DAG nodes (model, test, etc) |

## synq.entities.v1.Identifier

Identifier is a unique reference to an entity in SYNQ system. Entity identifiers are designed to closely mimic identifiers used by data platforms and tools. To construct an identifier, you need to know the kind of the entity and the ids that you would normally use to identify it in the data platform or tool. For example, to identify a table in BigQuery, you would need to know the project, dataset, and table names.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| dbt_core_node | `object` |  | [synq.entities.v1.DbtCoreNodeIdentifier](#synq.entities.v1.dbtcorenodeidentifier) | Dbt node that identifies one of dbt DAG nodes (model, test, etc) in dbt Core project |
| dbt_cloud_node | `object` |  | [synq.entities.v1.DbtCloudNodeIdentifier](#synq.entities.v1.dbtcloudnodeidentifier) | Dbt node that identifies one of dbt DAG nodes (model, test, etc) in dbt Cloud project |
| bigquery_table | `object` |  | [synq.entities.v1.BigqueryTableIdentifier](#synq.entities.v1.bigquerytableidentifier) | BigQuery table identifier |
| snowflake_table | `object` |  | [synq.entities.v1.SnowflakeTableIdentifier](#synq.entities.v1.snowflaketableidentifier) | Snowflake table identifier |
| redshift_table | `object` |  | [synq.entities.v1.RedshiftTableIdentifier](#synq.entities.v1.redshifttableidentifier) | Redshift table identifier |
| postgres_table | `object` |  | [synq.entities.v1.PostgresTableIdentifier](#synq.entities.v1.postgrestableidentifier) | Postgres table identifier |
| mysql_table | `object` |  | [synq.entities.v1.MysqlTableIdentifier](#synq.entities.v1.mysqltableidentifier) | Mysql table identifier |
| clickhouse_table | `object` |  | [synq.entities.v1.ClickhouseTableIdentifier](#synq.entities.v1.clickhousetableidentifier) | Clickhouse table identifier |
| airflow_dag | `object` |  | [synq.entities.v1.AirflowDagIdentifier](#synq.entities.v1.airflowdagidentifier) | Airflow DAG identifier |
| airflow_task | `object` |  | [synq.entities.v1.AirflowTaskIdentifier](#synq.entities.v1.airflowtaskidentifier) | Airflow task identifier within a given DAG |
| custom | `object` |  | [synq.entities.v1.CustomIdentifier](#synq.entities.v1.customidentifier) | Custom identifier to be used with all custom created entities |
| dataproduct | `object` |  | [synq.entities.v1.DataproductIdentifier](#synq.entities.v1.dataproductidentifier) | Dataproduct identifier |
| synq_path | `object` |  | [synq.entities.v1.SynqPathIdentifier](#synq.entities.v1.synqpathidentifier) | SynqPath identifier |
| databricks_table | `object` |  | [synq.entities.v1.DatabricksTableIdentifier](#synq.entities.v1.databrickstableidentifier) | Databricks table identifier |
| trino_table | `object` |  | [synq.entities.v1.TrinoTableIdentifier](#synq.entities.v1.trinotableidentifier) | Trino table identifier |
| sql_mesh_model | `object` |  | [synq.entities.v1.SqlMeshModelIdentifier](#synq.entities.v1.sqlmeshmodelidentifier) | SQLMesh Model identifier |
| sql_mesh_audit | `object` |  | [synq.entities.v1.SqlMeshAuditIdentifier](#synq.entities.v1.sqlmeshauditidentifier) | SQLMesh Audit identifier |
| monitor | `object` |  | [synq.entities.v1.MonitorIdentifier](#synq.entities.v1.monitoridentifier) | Monitor identifier |

## synq.entities.v1.MonitorIdentifier

MONITORS

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| monitored_id | `object` |  | [synq.entities.v1.Identifier](#synq.entities.v1.identifier) | Identifier of the monitored entity |
| monitor_id | `string` |  | string | Identifier of the monitor |
| segment | `string` |  | string | Optional monitor segmentation identifier |
| integration_id | `string` |  | string | SYNQ integration_id of the monitored identifier |

## synq.entities.v1.MysqlTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Mysql hostname without port |
| schema | `string` |  | string | Mysql database |
| table | `string` |  | string | Mysql table |

## synq.entities.v1.PostgresTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Postgres hostname without port |
| database | `string` |  | string | Postgres database |
| schema | `string` |  | string | Postgres schema |
| table | `string` |  | string | Postgres table |

## synq.entities.v1.RedshiftTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| cluster | `string` |  | string | Redshift cluster |
| database | `string` |  | string | Redshift database |
| schema | `string` |  | string | Redshift schema |
| table | `string` |  | string | Redshift table |

## synq.entities.v1.SnowflakeTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| account | `string` |  | string | Snowflake account |
| database | `string` |  | string | Snowflake database |
| schema | `string` |  | string | Snowflake schema |
| table | `string` |  | string | Snowflake table |

## synq.entities.v1.SqlMeshAuditIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| integration_id | `string` |  | string | SYNQ integration_id that identifies the dbt Core project |
| fqn | `string` |  | string | SQLMesh model fully qualified name |
| audit_id | `string` |  | string | Identifier of the audit |

## synq.entities.v1.SqlMeshModelIdentifier

SQLMesh

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| integration_id | `string` |  | string | SYNQ integration_id that identifies the dbt Core project |
| fqn | `string` |  | string | SQLMesh model fully qualified name |

## synq.entities.v1.SynqPathIdentifier

SYNQ PATH

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| path | `string` |  | string | SYNQ path that identifies the SYNQ entity, needs to be one of supported paths |

## synq.entities.v1.TrinoTableIdentifier

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Hostname of the Trino instance |
| catalog | `string` |  | string | Trino catalog |
| schema | `string` |  | string | Trino schema |
| table | `string` |  | string | Trino table or view |

## synq.issues.actor.v1.Actor

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| name | `string` |  | string |  |
| slack | `object` |  | [synq.issues.actor.v1.SlackUser](#synq.issues.actor.v1.slackuser) |  |
| email | `object` |  | [synq.issues.actor.v1.EmailUser](#synq.issues.actor.v1.emailuser) |  |
| pagerduty | `object` |  | [synq.issues.actor.v1.PagerdutyUser](#synq.issues.actor.v1.pagerdutyuser) |  |

## synq.issues.actor.v1.EmailUser

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| user_email | `string` |  | string |  |

## synq.issues.actor.v1.PagerdutyUser

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| user_id | `string` |  | string |  |

## synq.issues.actor.v1.SlackUser

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| user_id | `string` |  | string |  |

## synq.issues.commands.v1.IssuesCommand

Not to be used directly. Use the IssuesService instead when calling via API.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| workspace | `string` |  | string |  |
| mark_investigating | `object` |  | [synq.issues.issues.v1.MarkInvestigatingRequest](#synq.issues.issues.v1.markinvestigatingrequest) |  |
| mark_fixed | `object` |  | [synq.issues.issues.v1.MarkFixedRequest](#synq.issues.issues.v1.markfixedrequest) |  |
| mark_expected | `object` |  | [synq.issues.issues.v1.MarkExpectedRequest](#synq.issues.issues.v1.markexpectedrequest) |  |
| mark_no_action_needed | `object` |  | [synq.issues.issues.v1.MarkNoActionNeededRequest](#synq.issues.issues.v1.marknoactionneededrequest) |  |
| post_comment | `object` |  | [synq.issues.issues.v1.PostCommentRequest](#synq.issues.issues.v1.postcommentrequest) |  |

## synq.issues.issues.v1.MarkExpectedRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue_id | `string` |  | string | ID of the issue to mark as expected. |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) | Actor marking the issue as expected. |
| time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time at which the issue was marked as expected. Defaults to the current time. |
| require_no_existing_status | `boolean` |  | boolean | Ignore status change if the issue already has a status. |

## synq.issues.issues.v1.MarkFixedRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue_id | `string` |  | string | ID of the issue to mark as fixed. |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) | Actor marking the issue as fixed. |
| time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time at which the issue was marked as fixed. Defaults to the current time. |
| require_no_existing_status | `boolean` |  | boolean | Ignore status change if the issue already has a status. |

## synq.issues.issues.v1.MarkInvestigatingRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue_id | `string` |  | string | ID of the issue to mark as investigating. |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) | Actor marking the issue as investigating. |
| time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time at which the issue was marked as investigating. Defaults to the current time. |
| require_no_existing_status | `boolean` |  | boolean | Ignore status change if the issue already has a status. |

## synq.issues.issues.v1.MarkNoActionNeededRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue_id | `string` |  | string | ID of the issue to mark as no action needed. |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) | Actor marking the issue as no action needed. |
| time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time at which the issue was marked as no action needed. Defaults to the current time. |
| require_no_existing_status | `boolean` |  | boolean | Ignore status change if the issue already has a status. |

## synq.issues.issues.v1.PostCommentRequest

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue_id | `string` |  | string | ID of the issue to post a comment on. |
| actor | `object` |  | [synq.issues.actor.v1.Actor](#synq.issues.actor.v1.actor) | Actor posting the comment. |
| comment | `string` |  | string | Comment to post. |
| time | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time at which the comment was posted. Defaults to the current time. |

## Callback

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| url | `string` |  | string |  |
| action_name | `string` |  | string |  |
| issues_command | `object` |  | [synq.issues.commands.v1.IssuesCommand](#synq.issues.commands.v1.issuescommand) |  |

## IncidentCancelled

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| incident | `object` |  | [IncidentSummary](#incidentsummary) |  |

## IncidentClosed

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| incident | `object` |  | [IncidentSummary](#incidentsummary) |  |

## IncidentOpen

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| incident | `object` |  | [IncidentSummary](#incidentsummary) |  |

## IncidentSummary

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| incident_id | `string` |  | string |  |
| incident_url | `string` |  | string |  |
| title | `string` |  | string |  |
| description | `string` |  | string |  |
| description_html | `string` |  | string |  |
| started_at | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) |  |
| ended_at | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) |  |

## IssueClosed

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue | `object` |  | [IssueSummary](#issuesummary) |  |

## IssueCreated

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue | `object` |  | [IssueSummary](#issuesummary) |  |

## IssueStatusUpdated

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue | `object` |  | [IssueSummary](#issuesummary) |  |

## IssueSummary

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue_id | `string` |  | string |  |
| issue_group_id | `string` |  | string |  |
| issue_url | `string` |  | string |  |
| title | `string` |  | string | Summary of the issue, what happened and where. |
| description | `string` |  | string | Detailed description of the issue. In the Markdown format. |
| description_html | `string` |  | string | Detailed description of the issue. In the HTML format. |
| trigger_entity | `object` |  | [IssueSummary.IssueEntity](#issuesummary.issueentity) | Entity which triggered the issue. |
| directly_affected_entities | `array` |  | [IssueSummary.IssueEntity](#issuesummary.issueentity) | Entities directly affected by the issue, not considering downstream ones. |
| started_at | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time when the issue was triggered. |
| ended_at | `string` |  | Format: [`date-time`](https://json-schema.org/understanding-json-schema/reference/string#built-in-formats) | Time when the issue was closed. |
| trigger_run_id | `string` |  | string |  |
| trigger_name | `string` |  | string |  |
| trigger_message | `string` |  | string |  |
| status | `string` |  | `ISSUE_STATUS_UNSPECIFIED` `ISSUE_STATUS_INVESTIGATING` `ISSUE_STATUS_EXPECTED` `ISSUE_STATUS_FIXED` `ISSUE_STATUS_NO_ACTION_NEEDED` |  |

## IssueSummary.IssueEntity

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| name | `string` |  | string |  |
| type_name | `string` |  | string |  |
| identifier | `object` |  | [synq.entities.v1.Identifier](#synq.entities.v1.identifier) |  |
| folder | `string` |  | string |  |
| entity_url | `string` |  | string |  |

## IssueUpdated

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| issue | `object` |  | [IssueSummary](#issuesummary) |  |

## Ping

Test event sent during a webhook setup.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| message | `string` |  | string |  |


---

Markdown generated with [jsonschema-markdown](https://github.com/elisiariocouto/jsonschema-markdown).
