---
title: "Getting Started"
description: "Getting started with the Coalesce Quality developer API"
---

# Overview

You can use the Coalesce Quality API to automate tasks from your own code. The API is exposed as [gRPC](https://grpc.io/) services, so a call is like a function call in your code.

To use the API, do the following:

1. Install the prebuilt SDK for your language, or clone the protos from our GitHub repository and generate the client code.

2. Create API client credentials, fetch an access token and use it to connect to the API.

3. Initialize clients and call functions in your code.

You can find language specific examples [here](https://github.com/getsynq/api/tree/main/examples).

# Client code

The simplest way to get a client is to use the prebuilt SDKs from our [`buf` repository](https://buf.build/getsynq/api/sdks). Select your language and follow the instructions there.

## Generating client code

You can also generate the code from the protos in our [GitHub repository](https://github.com/getsynq/api).

```bash
$ git clone git@github.com:getsynq/api.git <synq_api_codebase>
```

[gRPC](https://grpc.io/) supports many languages. The [gRPC language guides](https://grpc.io/docs/languages/) list the tools to generate client code in each one.

You also need [protovalidate](https://github.com/bufbuild/protovalidate) for your language.

Following are some language references.

## Go

You need the following plugins to generate Go code from the protos.

```bash
$ go install google.golang.org/protobuf/cmd/protoc-gen-go
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

If you are new to gRPC and protos, the [Go quick start](https://grpc.io/docs/languages/go/quickstart/) is a good place to begin.

Run the following command from `<synq_api_codebase>` to generate the code in Go.

```bash
$ protoc --proto_path=./protos --go_out=./gen --go-grpc_out=./gen protos/**/*.proto
```

The generated code goes to the `./gen` folder. To change the location or other options, see the [Go generated code reference](https://protobuf.dev/reference/go/go-generated/).

## Python

You need the following tools to generate Python code from the protos.

```bash
$ python -m pip install grpcio
$ python -m pip install grpcio-tools
```

If you are new to gRPC and protos, the [Python quick start](https://grpc.io/docs/languages/python/quickstart/) is a good place to begin.

Run the following command from `<synq_api_codebase>` to generate the code in Python.

```bash
$ python3 -m grpc_tools.protoc -Iprotos --python_out=./gen --pyi_out=./gen --grpc_python_out=./gen protos/**/*.proto
```

The generated code goes to the `./gen` folder. You can change the location to suit your project.

# Fetching an access token

You need a valid access token to call the API. You fetch the token with the client credentials (`CLIENT_ID` and `CLIENT_SECRET`) of an API client.

Create an API client under **Settings → API**, in the **API clients** section ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)). The client secret is shown only once, when you create the client. Creating an API client requires the Admin role.

The same page also has **API tokens**. They are long-lived and meant for cases like status badges. For API calls from your own systems, use an API client.

You select the scopes of the client when you create it. A call succeeds when the client has at least one of the scopes its method requires. [API scopes](https://docs.synq.io/api-reference/scopes) lists every scope and the methods it allows. Pick the smallest set that covers the RPCs you call.

Fetch the token with the following `POST` call to the OAuth2 server of your region.

| Region | OAuth2 token URL |
|---|---|
| EU | `https://developer.synq.io/oauth2/token` |
| US | `https://api.us.synq.io/oauth2/token` |
| AU | `https://api.au.synq.io/oauth2/token` |

```bash
curl -d "client_id=<CLIENT_ID>&client_secret=<CLIENT_SECRET>&grant_type=client_credentials" -X POST https://developer.synq.io/oauth2/token
```

The response has the following structure.

```json
{
    "access_token": <access_token>,
    "expires_in": <expiry_seconds>,
    ...
}
```

The `<access_token>` is a JWT. Send it with every API call as an [Authorization Bearer](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Authorization) header. Fetch a new token before `expires_in` seconds pass.

# Connecting to the API

Connect to the API host of your region on port `443` with TLS:

* EU: `developer.synq.io:443`
* US: `api.us.synq.io:443`
* AU: `api.au.synq.io:443`

Set the gRPC authority to the host name without the port. If you do not set it, the port goes into the `Host` header and every call returns `UNIMPLEMENTED` with no message.

In Go:

```go
conn, err := grpc.NewClient("developer.synq.io:443",
	grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	grpc.WithPerRPCCredentials(oauthTokenSource),
	grpc.WithAuthority("developer.synq.io"),
)
```

In Python:

```python
channel = grpc.secure_channel(
    "developer.synq.io:443",
    credentials,
    options=(("grpc.default_authority", "developer.synq.io"),),
)
```

# Examples

You can find language specific examples of API calls [here](https://docs.synq.io/api-reference/examples).

# Troubleshooting

* If connecting to the gRPC services is very slow, flush your DNS cache. This happens most often behind a VPN. On macOS:

  ```
  dscacheutil -flushcache
  killall -HUP mDNSResponder
  ```
