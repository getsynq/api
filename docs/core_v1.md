# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [core/v1/entity_type.proto](#core_v1_entity_type-proto)
    - [EntityType](#core-v1-EntityType)
  
- [core/v1/platforms.proto](#core_v1_platforms-proto)
    - [Platform](#core-v1-Platform)
  
- [core/v1/entity.proto](#core_v1_entity-proto)
    - [Asset](#core-v1-Asset)
    - [Entity](#core-v1-Entity)
    - [EntityRef](#core-v1-EntityRef)
    - [Monitor](#core-v1-Monitor)
    - [MonitorSegment](#core-v1-MonitorSegment)
  
- [Scalar Value Types](#scalar-value-types)



<a name="core_v1_entity_type-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1/entity_type.proto


 


<a name="core-v1-EntityType"></a>

### EntityType
Type of Entity.
This enum lists all the types currently supported by Synq.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITY_TYPE_UNSPECIFIED | 0 |  |
| ENTITY_TYPE_FOLDER | 1 |  |
| ENTITY_TYPE_BQ_PROJECT | 101 |  |
| ENTITY_TYPE_BQ_DATASET | 102 |  |
| ENTITY_TYPE_BQ_TABLE | 103 |  |
| ENTITY_TYPE_BQ_VIEW | 105 |  |
| ENTITY_TYPE_LOOKER_LOOK | 201 |  |
| ENTITY_TYPE_LOOKER_EXPLORE | 203 |  |
| ENTITY_TYPE_LOOKER_VIEW | 207 |  |
| ENTITY_TYPE_LOOKER_DASHBOARD | 208 |  |
| ENTITY_TYPE_DBT_MODEL | 301 |  |
| ENTITY_TYPE_DBT_TEST | 302 |  |
| ENTITY_TYPE_DBT_SOURCE | 303 |  |
| ENTITY_TYPE_DBT_MACRO | 304 |  |
| ENTITY_TYPE_DBT_PROJECT | 306 |  |
| ENTITY_TYPE_DBT_METRIC | 307 |  |
| ENTITY_TYPE_DBT_SNAPSHOT | 310 |  |
| ENTITY_TYPE_DBT_SEED | 311 |  |
| ENTITY_TYPE_DBT_ANALYSIS | 312 |  |
| ENTITY_TYPE_DBT_EXPOSURE | 313 |  |
| ENTITY_TYPE_DBT_GROUP | 314 |  |
| ENTITY_TYPE_DBT_CLOUD_ACCOUNT | 351 |  |
| ENTITY_TYPE_DBT_CLOUD_PROJECT | 352 |  |
| ENTITY_TYPE_DBT_CLOUD_JOB | 353 |  |
| ENTITY_TYPE_DBT_CLOUD_JOB_STEP | 354 |  |
| ENTITY_TYPE_SNOWFLAKE_PROJECT | 501 |  |
| ENTITY_TYPE_SNOWFLAKE_DATASET | 502 |  |
| ENTITY_TYPE_SNOWFLAKE_TABLE | 503 |  |
| ENTITY_TYPE_SNOWFLAKE_ACCOUNT | 505 |  |
| ENTITY_TYPE_SNOWFLAKE_DATABASE | 506 |  |
| ENTITY_TYPE_SNOWFLAKE_SCHEMA | 507 |  |
| ENTITY_TYPE_SNOWFLAKE_VIEW | 508 |  |
| ENTITY_TYPE_REDSHIFT_DATABASE | 801 |  |
| ENTITY_TYPE_REDSHIFT_SCHEMA | 802 |  |
| ENTITY_TYPE_REDSHIFT_TABLE | 803 |  |
| ENTITY_TYPE_REDSHIFT_VIEW | 805 |  |
| ENTITY_TYPE_TABLEAU_EMBEDDED | 1101 |  |
| ENTITY_TYPE_TABLEAU_PUBLISHED | 1102 |  |
| ENTITY_TYPE_TABLEAU_CUSTOM_SQL | 1103 |  |
| ENTITY_TYPE_TABLEAU_TABLE | 1104 |  |
| ENTITY_TYPE_TABLEAU_SHEET | 1105 |  |
| ENTITY_TYPE_TABLEAU_DASHBOARD | 1106 |  |
| ENTITY_TYPE_AIRFLOW_DAG | 1201 |  |
| ENTITY_TYPE_AIRFLOW_TASK | 1202 |  |
| ENTITY_TYPE_CLICKHOUSE_DATABASE | 1301 |  |
| ENTITY_TYPE_CLICKHOUSE_SCHEMA | 1302 |  |
| ENTITY_TYPE_CLICKHOUSE_TABLE | 1303 |  |
| ENTITY_TYPE_CLICKHOUSE_VIEW | 1305 |  |
| ENTITY_TYPE_ANOMALY_INTEGRATION | 1401 |  |
| ENTITY_TYPE_ANOMALY_MONITOR | 1403 |  |
| ENTITY_TYPE_ANOMALY_MONITOR_SEGMENT | 1404 |  |
| ENTITY_TYPE_POSTGRES_DATABASE | 1601 |  |
| ENTITY_TYPE_POSTGRES_SCHEMA | 1602 |  |
| ENTITY_TYPE_POSTGRES_TABLE | 1603 |  |
| ENTITY_TYPE_POSTGRES_VIEW | 1605 |  |
| ENTITY_TYPE_MYSQL_DATABASE | 1701 |  |
| ENTITY_TYPE_MYSQL_SCHEMA | 1702 |  |
| ENTITY_TYPE_MYSQL_TABLE | 1703 |  |
| ENTITY_TYPE_MYSQL_VIEW | 1705 |  |


 

 

 



<a name="core_v1_platforms-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1/platforms.proto


 


<a name="core-v1-Platform"></a>

### Platform
Platforms supported by Synq.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PLATFORM_UNSPECIFIED | 0 |  |
| PLATFORM_BIGQUERY | 10 |  |
| PLATFORM_LOOKER | 20 |  |
| PLATFORM_DBT | 30 |  |
| PLATFORM_DBT_CLOUD | 31 |  |
| PLATFORM_DBT_SELF_HOSTED | 32 |  |
| PLATFORM_SNOWFLAKE | 50 |  |
| PLATFORM_REDSHIFT | 80 |  |
| PLATFORM_TABLEAU | 110 |  |
| PLATFORM_AIRFLOW | 120 |  |
| PLATFORM_CLICKHOUSE | 130 |  |
| PLATFORM_POSTGRES | 160 |  |
| PLATFORM_MYSQL | 170 |  |


 

 

 



<a name="core_v1_entity-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1/entity.proto



<a name="core-v1-Asset"></a>

### Asset
Entity: Asset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset_path | [string](#string) |  |  |
| platform | [Platform](#core-v1-Platform) |  |  |






<a name="core-v1-Entity"></a>

### Entity
Entity in Synq.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset | [Asset](#core-v1-Asset) |  |  |
| monitor | [Monitor](#core-v1-Monitor) |  |  |
| monitor_segment | [MonitorSegment](#core-v1-MonitorSegment) |  |  |






<a name="core-v1-EntityRef"></a>

### EntityRef
Lightweight reference to an Entity


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | Unique path for entity. |
| type | [EntityType](#core-v1-EntityType) |  | Type of entity. |
| name | [string](#string) | optional | Human friendly name. |






<a name="core-v1-Monitor"></a>

### Monitor
Entity: Monitor


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| monitor_path | [string](#string) |  |  |






<a name="core-v1-MonitorSegment"></a>

### MonitorSegment
Entity: Monitor Segment


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| monitor_path | [string](#string) |  |  |
| metric_id | [string](#string) |  |  |
| segment | [string](#string) |  |  |





 

 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

