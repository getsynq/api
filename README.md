# Synq API

The Synq API is available for developers to manage certain functionalities using custom workflows. Synq exposes its API as [gRPC](https://grpc.io/) services. This means that the API are as easy to use as calling functions from your code.

To use the API, you need to do the following.

1. Clone protos from our github repository and generate the client code in the language of your choice.

2. Generate an access token and use it to connect to Synq API.

3. Initialize clients and call functions in your code.

You can find language specific examples [here](https://github.com/getsynq/api/tree/main/examples).

## Generating client code

You can generate the code from the protos available at our [github repository](https://github.com/getsynq/api).

```bash
$ git clone git@github.com:getsynq/api.git <synq_api_codebase>
```

[gRPC](https://grpc.io/) supports a wide choice of languages and you can find the necessary guides on how to get started in a language of your choice [here](https://grpc.io/docs/languages/). You don't need to understand it all to get started with Synq API. You just need to find the right tools to build the client code in your language of choice.

Following are some language references.

### Go

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

### Python

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

## Fetching Access Token

You need a valid access token to communicate with the Synq servers. You can generate an access token with limited validity using your client credentials (`CLIENT_ID` and `CLIENT_SECRET`). Reach out to us in order to generate your client credentials.

You can fetch the token source by making the following `POST` call to our OAuth2 server.

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

The `<access_token>` thus fetched is a valid JWT token.

The following example show how to achieve this in golang.

```go
func getToken() (string, error) {
	type oauth2Token struct {
		AccessToken string `json:"access_token"`
	}

	tokenUrl := "http://api.synq.io/oauth2/token"

	v := &url.Values{}
	v.Set("client_id", "xxxxxx") // replace with your CLIENT_ID
	v.Set("client_secret", "0000000000000000") // replace with your CLIENT_SECRET
	v.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenUrl, strings.NewReader(v.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	token := &oauth2Token{}
	if err := json.NewDecoder(res.Body).Decode(token); err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching token. status code %d", res.StatusCode)
	}

	return token.AccessToken, nil
}
```

## Making API calls

The access token fetched above needs to be passed to the calls made to Synq API. The server expects the token set as an [Authorization Bearer](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Authorization) header.

The following example illustrates how to do this in golang.

```go

type bearerAuth struct {
	token string
}

// Convert user credentials to request metadata.
func (b bearerAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + b.token,
	}, nil
}

// Specify whether channel security is required to pass these credentials.
func (b bearerAuth) RequireTransportSecurity() bool {
	return false
}

func main() {
    token, err := getToken() // Check section above on fetching the token
	if err != nil {
		panic(err)
	}
	fmt.Println(token)
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(bearerAuth{
			token: token,
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial("https://api.synq.io", opts...)
	if err != nil {
		panic(err)
	}

    ...
}
```

You can now use the connection to initialize clients and make calls to the API.

```go
func main {
    ...

    client := schemasv1.NewLineageServiceClient(conn)
    resp, err := client.GetLineage(context.Background(), &schemasv1.GetLineageRequest{
		StartPoint: &schemasv1.GetLineageStartPoint{
			From: &schemasv1.GetLineageStartPoint_Entities{
				Entities: &schemasv1.EntitiesStartPoint{
					Entities: []*corev1.EntityRef{
						{
							Path: "xxxxxxxxxxxxxxxxxxxxx", // replace with entity path
							Type: corev1.EntityType_ENTITY_TYPE_UNSPECIFIED, // replace with entity type
						},
					},
				},
			},
		},
	})
}

```
