# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [example/v2/example_service.proto](#example_v2_example_service-proto)
    - [AddExampleRequest](#example-v2-AddExampleRequest)
    - [AddExampleResponse](#example-v2-AddExampleResponse)
    - [ExampleResponse](#example-v2-ExampleResponse)
    - [GetExampleRequest](#example-v2-GetExampleRequest)
    - [GetExampleResponse](#example-v2-GetExampleResponse)
    - [ListExamplesRequest](#example-v2-ListExamplesRequest)
    - [RemoveExampleRequest](#example-v2-RemoveExampleRequest)
    - [RemoveExampleResponse](#example-v2-RemoveExampleResponse)
    - [UpdateExampleRequest](#example-v2-UpdateExampleRequest)
    - [UpdateExampleResponse](#example-v2-UpdateExampleResponse)
  
    - [ExampleService](#example-v2-ExampleService)
  
- [example/v2/example.proto](#example_v2_example-proto)
    - [Example](#example-v2-Example)
  
- [Scalar Value Types](#scalar-value-types)



<a name="example_v2_example_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## example/v2/example_service.proto



<a name="example-v2-AddExampleRequest"></a>

### AddExampleRequest
AddExampleRequest contains the parameters required to create an example.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| text | [string](#string) |  | The example itself. |
| description | [string](#string) | optional | Optional description of the example. |
| is_visible | [bool](#bool) |  | Whether the example is visible to users. |






<a name="example-v2-AddExampleResponse"></a>

### AddExampleResponse
Response contains the created example.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| example | [Example](#example-v2-Example) |  |  |






<a name="example-v2-ExampleResponse"></a>

### ExampleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| example | [Example](#example-v2-Example) |  |  |






<a name="example-v2-GetExampleRequest"></a>

### GetExampleRequest
GetExampleRequest gets an example by its ID.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="example-v2-GetExampleResponse"></a>

### GetExampleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| example | [Example](#example-v2-Example) |  |  |






<a name="example-v2-ListExamplesRequest"></a>

### ListExamplesRequest
ListExamplesRequest streams examples from the server.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ids | [string](#string) | repeated |  |






<a name="example-v2-RemoveExampleRequest"></a>

### RemoveExampleRequest
RemoveExampleRequest removes an example by its ID.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="example-v2-RemoveExampleResponse"></a>

### RemoveExampleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |






<a name="example-v2-UpdateExampleRequest"></a>

### UpdateExampleRequest
UpdateExampleRequest contains updateable fields inside an example.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| text | [string](#string) | optional | Optional update of the example text. |
| description | [string](#string) | optional | Optional update of the example description. |
| is_visible | [bool](#bool) | optional | Optional update of example visibility. |






<a name="example-v2-UpdateExampleResponse"></a>

### UpdateExampleResponse
Response with updated example.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| example | [Example](#example-v2-Example) |  |  |





 

 

 


<a name="example-v2-ExampleService"></a>

### ExampleService
ExamplesService is a sample service to display how documentation in protos will work.
The service implements a basic CRUD API and contains server streaming example.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| AddExample | [AddExampleRequest](#example-v2-AddExampleRequest) | [AddExampleResponse](#example-v2-AddExampleResponse) | AddExample adds the given example to the system. |
| UpdateExample | [UpdateExampleRequest](#example-v2-UpdateExampleRequest) | [UpdateExampleResponse](#example-v2-UpdateExampleResponse) | UpdateExample updates the given example. Throws an error if the example was not found. |
| GetExample | [GetExampleRequest](#example-v2-GetExampleRequest) | [GetExampleResponse](#example-v2-GetExampleResponse) | GetExample returns the requested example. |
| ListExamples | [ListExamplesRequest](#example-v2-ListExamplesRequest) | [ExampleResponse](#example-v2-ExampleResponse) stream | ListExamples streams the requested examples. |
| RemoveExample | [RemoveExampleRequest](#example-v2-RemoveExampleRequest) | [RemoveExampleResponse](#example-v2-RemoveExampleResponse) | RemoveExample removes the provided example from the system. Fails silently if the example was not found. |

 



<a name="example_v2_example-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## example/v2/example.proto



<a name="example-v2-Example"></a>

### Example
Example is the core model of this package.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | `id` is generated by the system when a new example is created. It is required to refer to an example in any future operations (like update or remove). |
| text | [string](#string) |  | The example itself. |
| description | [string](#string) | optional | Optional description of the example. |
| is_visible | [bool](#bool) |  | Whether the example is visible to users. |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Example creation time. |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Example last update time. |





 

 

 

 



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

