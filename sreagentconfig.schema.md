# jsonschema-markdown

JSON Schema missing a description, provide it using the `description` key in the root of the JSON document.

### Type: `object(?)`


---

# Definitions

## Config

Config represents the main configuration for the SRE agent

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| synq | `object` |  | [SYNQ](#synq) | SYNQ platform configuration |
| connections | `object` |  | [Connection](#connection) | Map of connection configurations |
| llm | `object` |  | [LlmConfig](#llmconfig) |  |

## BigQueryConf

BigQuery specific configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| project_id | `string` |  | string | GCP project ID |
| service_account_key | `string` |  | string | Service account key JSON |
| service_account_key_file | `string` |  | string | Location of service account key file |
| region | `string` |  | string | Region for BigQuery resources |

## ClickhouseConf

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Host address |
| port | `integer` |  | integer | Port number |
| database | `string` |  | string | Database name |
| username | `string` |  | string | Username for authentication |
| password | `string` |  | string | Password for authentication |
| allow_insecure | `boolean` |  | boolean | Whether to use disable SSL for connection |

## Connection

Connection represents a database connection configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| name | `string` |  | string | Name of the connection |
| disabled | `boolean` |  | boolean |  |
| parallelism | `integer` |  | integer | How many queries to DWH can be executed in parallel, defaults to 2 |
| bigquery | `object` |  | [BigQueryConf](#bigqueryconf) |  |
| clickhouse | `object` |  | [ClickhouseConf](#clickhouseconf) |  |
| databricks | `object` |  | [DatabricksConf](#databricksconf) |  |
| mysql | `object` |  | [MySQLConf](#mysqlconf) |  |
| postgres | `object` |  | [PostgresConf](#postgresconf) |  |
| redshift | `object` |  | [RedshiftConf](#redshiftconf) |  |
| snowflake | `object` |  | [SnowflakeConf](#snowflakeconf) |  |
| trino | `object` |  | [TrinoConf](#trinoconf) |  |

## DatabricksConf

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| workspace_url | `string` |  | string |  |
| auth_token | `string` |  | string |  |
| auth_client | `string` |  | string |  |
| auth_secret | `string` |  | string |  |
| warehouse | `string` |  | string |  |
| refresh_table_metrics | `boolean` |  | boolean |  |
| refresh_table_metrics_use_scan | `boolean` |  | boolean |  |
| fetch_table_tags | `boolean` |  | boolean |  |
| use_show_create_table | `boolean` |  | boolean |  |

## MySQLConf

MySQL specific configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Host address |
| port | `integer` |  | integer | Port number |
| database | `string` |  | string | Database name |
| username | `string` |  | string | Username for authentication |
| password | `string` |  | string | Password for authentication |
| allow_insecure | `boolean` |  | boolean | Whether to allow insecure connections |
| params | `object` |  | object | Additional connection parameters |

## PostgresConf

Postgres specific configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Host address |
| port | `integer` |  | integer | Port number |
| database | `string` |  | string | Database name |
| username | `string` |  | string | Username for authentication |
| password | `string` |  | string | Password for authentication |
| allow_insecure | `boolean` |  | boolean | Whether to allow insecure connections |

## RedshiftConf

Redshift specific configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Host address |
| port | `integer` |  | integer | Port number |
| database | `string` |  | string | Database name |
| username | `string` |  | string | Username for authentication |
| password | `string` |  | string | Password for authentication |
| freshness_from_query_logs | `boolean` |  | boolean | Estimate table freshness based on query logs |

## SnowflakeConf

Snowflake specific configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| account | `string` |  | string | Snowflake account identifier |
| warehouse | `string` |  | string | Virtual warehouse to use |
| role | `string` |  | string | Role to assume |
| username | `string` |  | string | Username for authentication |
| password | `string` |  | string | Password for authentication |
| private_key | `string` |  | string | Content of Private key used for Snowflake authentication |
| private_key_file | `string` |  | string | Location of the file containing Private key used for Snowflake authentication |
| private_key_passphrase | `string` |  | string | Passphrase used to decode Private key |
| databases | `array` |  | string | Database to connect to |
| use_get_ddl | `boolean` |  | boolean | Use GET_DDL to determine queries used for table/view creation |
| account_usage_db | `string` |  | string | Name of the database where ACCOUNT_USAGE schema is present, fallbacks to SNOWFLAKE |
| auth_type | `string` |  | string | Authentication type: empty (default, uses password or private_key), "externalbrowser" (SSO via browser) When set to "externalbrowser", opens browser for SSO login and caches the token locally. |

## TrinoConf

Trino specific configuration

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| host | `string` |  | string | Host address |
| port | `integer` |  | integer | Optional port to use, otherwise it will use the default port 8080 |
| use_plaintext | `boolean` |  | boolean | Use non-SSL connection to Trino. This should only be enabled if the Trino cluster does not support SSL or if the connection is secured through other means (e.g., a VPN). Defaults to false (SSL enabled). |
| username | `string` |  | string | Username for authentication |
| password | `string` |  | string | Password for authentication |
| catalogs | `array` |  | string | To which catalogs to connect |
| no_show_create_view | `boolean` |  | boolean | Use SHOW CREATE VIEW to get views DDLs |
| no_show_create_table | `boolean` |  | boolean | Use SHOW CREATE TABLE to get tables DDLs |
| no_materialized_views | `boolean` |  | boolean | Should it fetch system.metadata.materialized_views to get information about Trino MVs |
| fetch_table_comments | `boolean` |  | boolean | Fetch Trino table comments from system.metadata.table_comments |

## LlmConfig

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| openai | `object` |  | [OpenAIApi](#openaiapi) |  |
| triage_parallelism | `integer` |  | integer | How many Issue triages can run in parallel |
| test_suggestions_parallelism | `integer` |  | integer | How many test suggestions can be processed in parallel |

## OpenAIApi

No description provided for this model.

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| base_url | `string` |  | string | Base URL of the OpenAPI compatible endpoint |
| api_key | `string` |  | string | API key to use for the interaction with the API |
| thinking_model | `string` |  | string | Model to use for complex operations |
| summary_model | `string` |  | string | Model to use for simpler operations, e.g. summary generation or improving the format of messages. |

## SYNQ

SYNQ contains authentication and connection details for the SYNQ platform

#### Type: `object`

| Property | Type | Required | Possible values | Description |
| -------- | ---- | -------- | --------------- | ----------- |
| client_id | `string` |  | string | Client ID for OAuth authentication |
| client_secret | `string` |  | string | Client secret for OAuth authentication |
| endpoint | `string` |  | string | SYNQ API agent endpoint (host:port) |
| ingest_endpoint | `string` |  | string | SYNQ API ingest endpoint (host:port) |
| oauth_url | `string` |  | string | OAuth authentication URL |


---

Markdown generated with [jsonschema-markdown](https://github.com/elisiariocouto/jsonschema-markdown).
