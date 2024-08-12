module github.com/getsynq/api/examples/golang/ingest_dbt

go 1.22

require (
	buf.build/gen/go/getsynq/api/grpc/go v1.3.0-20240603120535-eeda6bbccb21.3
	buf.build/gen/go/getsynq/api/protocolbuffers/go v1.34.1-20240603120535-eeda6bbccb21.1
	golang.org/x/oauth2 v0.20.0
	google.golang.org/grpc v1.64.0
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)
