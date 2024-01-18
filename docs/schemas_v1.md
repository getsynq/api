# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [schemas/v1/lineage_service.proto](#schemas_v1_lineage_service-proto)
    - [EntitiesStartPoint](#schemas-v1-EntitiesStartPoint)
    - [EntityColumnsStartPoint](#schemas-v1-EntityColumnsStartPoint)
    - [GetLineageRequest](#schemas-v1-GetLineageRequest)
    - [GetLineageResponse](#schemas-v1-GetLineageResponse)
    - [GetLineageStartPoint](#schemas-v1-GetLineageStartPoint)
  
    - [LineageDirection](#schemas-v1-LineageDirection)
  
    - [LineageService](#schemas-v1-LineageService)
  
- [schemas/v1/lineage.proto](#schemas_v1_lineage-proto)
    - [CllDetails](#schemas-v1-CllDetails)
    - [Column](#schemas-v1-Column)
    - [ColumnDependency](#schemas-v1-ColumnDependency)
    - [Lineage](#schemas-v1-Lineage)
    - [LineageNode](#schemas-v1-LineageNode)
    - [NodeDependency](#schemas-v1-NodeDependency)
  
    - [CllState](#schemas-v1-CllState)
    - [NodePosition](#schemas-v1-NodePosition)
  
- [Scalar Value Types](#scalar-value-types)



<a name="schemas_v1_lineage_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## schemas/v1/lineage_service.proto



<a name="schemas-v1-EntitiesStartPoint"></a>

### EntitiesStartPoint



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | [core.v1.EntityRef](#core-v1-EntityRef) | repeated |  |






<a name="schemas-v1-EntityColumnsStartPoint"></a>

### EntityColumnsStartPoint



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitiy | [core.v1.EntityRef](#core-v1-EntityRef) |  |  |
| column_ids | [string](#string) | repeated |  |






<a name="schemas-v1-GetLineageRequest"></a>

### GetLineageRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| lineage_direction | [LineageDirection](#schemas-v1-LineageDirection) |  |  |
| start_point | [GetLineageStartPoint](#schemas-v1-GetLineageStartPoint) |  |  |
| max_depth | [int32](#int32) | optional |  |






<a name="schemas-v1-GetLineageResponse"></a>

### GetLineageResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| lineage | [Lineage](#schemas-v1-Lineage) |  |  |






<a name="schemas-v1-GetLineageStartPoint"></a>

### GetLineageStartPoint
Possible starting points to get lineage from.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | [EntitiesStartPoint](#schemas-v1-EntitiesStartPoint) |  |  |
| entity_columns | [EntityColumnsStartPoint](#schemas-v1-EntityColumnsStartPoint) |  |  |





 


<a name="schemas-v1-LineageDirection"></a>

### LineageDirection
Direction of the lineage to query.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LINEAGE_DIRECTION_UNSPECIFIED | 0 |  |
| LINEAGE_DIRECTION_UPSTREAM | 1 |  |
| LINEAGE_DIRECTION_DOWNSTREAM | 2 |  |
| LINEAGE_DIRECTION_UPSTREAM_DOWNSTREAM | 3 |  |


 

 


<a name="schemas-v1-LineageService"></a>

### LineageService
LineageService allows you to fetch:
* Entity level lineage from a starting point of one or more entities.
* Column Level lineage from a starting point of multiple columns of a single entity.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetLineage | [GetLineageRequest](#schemas-v1-GetLineageRequest) | [GetLineageResponse](#schemas-v1-GetLineageResponse) |  |

 



<a name="schemas_v1_lineage-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## schemas/v1/lineage.proto



<a name="schemas-v1-CllDetails"></a>

### CllDetails



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| columns | [Column](#schemas-v1-Column) | repeated | Column details for CLL. |
| cll_state | [CllState](#schemas-v1-CllState) |  | State of the CLL parse. UNSPECIFIED if CLL was not requested. |
| cll_messages | [string](#string) | repeated | Messages related to CLL. e.g. Description of parse errors, etc. |






<a name="schemas-v1-Column"></a>

### Column
Column in a table-like asset (used in CLL mode).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| column_id | [string](#string) |  | ID string for the column. This is the parsed column name. |
| name | [string](#string) | optional | Original column name as fetched from the table. |
| native_type | [string](#string) | optional | Column type as fetched from the table. |






<a name="schemas-v1-ColumnDependency"></a>

### ColumnDependency
Indicates data flow between columns.
Source columns are used to compute value of target columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_node_idx | [uint32](#uint32) |  | Index of source node in the lineage nodes list. |
| source_node_column_id | [string](#string) |  |  |
| target_node_idx | [uint32](#uint32) |  | Index of target node in the lineage nodes list. |
| target_node_column_id | [string](#string) |  |  |






<a name="schemas-v1-Lineage"></a>

### Lineage
Lineage defines the lineage of table-like entities.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nodes | [LineageNode](#schemas-v1-LineageNode) | repeated | Nodes in the lineage with their identities and columns. |
| node_dependencies | [NodeDependency](#schemas-v1-NodeDependency) | repeated | All edges in the lineage between nodes. This can be parsed to create a graph of all the nodes. |
| is_cll | [bool](#bool) |  | Indicates whether the lineage was filtered for column level lineage (CLL). |
| column_dependencies | [ColumnDependency](#schemas-v1-ColumnDependency) | repeated | Dependencies between columns. Populated only for CLL. |






<a name="schemas-v1-LineageNode"></a>

### LineageNode
Node in a lineage graph representing one or more entiities (e.g. database table).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | [core.v1.EntityRef](#core-v1-EntityRef) | repeated | All entities which have the same identity as this node. Must be at least one item. These are sorted by closeness to the type of the start point entities. e.g. if requesting lineage of a DBT source, first entity should be from DBT, similarly when viewing table it will be other tables. |
| position | [NodePosition](#schemas-v1-NodePosition) |  | Position of the node in the lineage. |
| cll_details | [CllDetails](#schemas-v1-CllDetails) | optional | Populated only for Column Level Lineage (CLL). |






<a name="schemas-v1-NodeDependency"></a>

### NodeDependency
Indicates data flow between nodes.
Source nodes are used to compute value of target nodes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_node_idx | [uint32](#uint32) |  | Index of source node in the lineage nodes list. |
| target_node_idx | [uint32](#uint32) |  | Index of target node in the lineage nodes list. |





 


<a name="schemas-v1-CllState"></a>

### CllState


| Name | Number | Description |
| ---- | ------ | ----------- |
| CLL_STATE_UNSPECIFIED | 0 | Unspecified state. |
| CLL_STATE_PARSE_FAILED | 1 | Parsing of the asset SQL failed. No upstream dependencies can be found. |
| CLL_STATE_EXTRACTION_FAILED | 2 | Extraction of the asset SQL failed. Some unsupported SQL features may be used. Some details might be missing. |
| CLL_STATE_RESOLUTION_FAILED | 3 | Not all columns or tables were found upstream, lineage is not complete. |
| CLL_STATE_OK | 10 | No known issues present. |



<a name="schemas-v1-NodePosition"></a>

### NodePosition


| Name | Number | Description |
| ---- | ------ | ----------- |
| NODE_POSITION_UNSPECIFIED | 0 |  |
| NODE_POSITION_START_NODE | 1 | Node is one of the requested start point. |
| NODE_POSITION_UPSTREAM | 2 | Node is upstream of the requested start point. |
| NODE_POSITION_DOWNSTREAM | 3 | Node is downstream of the requested start point. |


 

 

 



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

