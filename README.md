---
title: 'Getting Started'
description: 'Getting started with developer API at Synq'
---

# Overview

The Synq API is available for developers to manage certain functionalities using custom workflows. Synq exposes its API as [gRPC](https://grpc.io/) services. This means that the API are as easy to use as calling functions from your code.

To use the API, you need to do the following.

1. Clone protos from our github repository and generate the client code in the language of your choice.

2. Generate an access token and use it to connect to Synq API.

3. Initialize clients and call functions in your code.

You can find language specific examples [here](https://github.com/getsynq/api/tree/main/examples).

# Client Code

The simplest way to use Synq API is to use the SDKs from our [`buf` repository](https://buf.build/getsynq/api/sdks). Use select the language of your choice and follow the instructions to add the Synq API to your project.


## Generating client code

If you prefer to, the client code can be generated from the protos available at our [github repository](https://github.com/getsynq/api).

```bash
$ git clone git@github.com:getsynq/api.git <synq_api_codebase>
```

[gRPC](https://grpc.io/) supports a wide choice of languages and you can find the necessary guides on how to get started in a language of your choice [here](https://grpc.io/docs/languages/). You don't need to understand it all to get started with Synq API. You just need to find the right tools to build the client code in your language of choice.

Following are some language references.

## Go

You will need the following plugins to generate golang code from the protos.

```bash
$ go install google.golang.org/protobuf/cmd/protoc-gen-go
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

If you are starting off with gRPC and protos, it might be useful to follow the guide [here](https://grpc.io/docs/languages/go/quickstart/).

Run the following command from `<synq_api_codebase>` to generate the code in golang.

```bash
$ protoc --proto_path=./protos --go_out=./gen2 --go-grpc_out=./gen protos/**/*.proto
```

The generated code is added to the `./gen` folder. You can change the location or find more options [here](https://protobuf.dev/reference/go/go-generated/) on how to use the `protoc` generator to suit your project's needs.

## Python

You will need the following tools to generate python code from the protos.

```bash
$ python -m pip install grpcio
$ python -m pip install grpcio-tools
```

If you are starting off with gRPC and protos, it might be useful to follow the guide [here](https://grpc.io/docs/languages/python/quickstart/).

Run the following command from `<synq_api_codebase>` to generate the code in golang.

```bash
$ python3 -m grpc_tools.protoc -Iprotos --python_out=./gen --pyi_out=./gen --grpc_python_out=./gen protos/**/*.proto
```

The generated code is added to the `./gen` folder. You can change the location to suit your project's needs.

# Fetching Access Token

You need a valid access token to communicate with the Synq servers. To generate the access token, you need client credentials.

You can generate an client credentials (`CLIENT_ID` and `CLIENT_SECRET`) from the [Synq application](https://app.synq.io/settings/api). The credentials are scoped so make sure to select the one best suited to execute the RPCs that you wish to.

You can now fetch the token source by making the following `POST` call to our OAuth2 server.

```bash
curl -d "client_id=<CLIENT_ID>&client_secret=<CLIENT_SECRET>&grant_type=client_credentials" -X POST http://api.synq.io/oauth2/token

```

The response will have the following structure.

```json
{
    "access_token": <access_token>,
    "expires_in": <expiry_seconds>,
    ...
}
```

The `<access_token>` thus fetched is a valid JWT token which should be passed on to the calls made to Synq API.

# Examples

The language specific examples to use Synq APIs can be found [here](https://docs.synq.io/api-reference/examples).
