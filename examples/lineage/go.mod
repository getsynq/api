module lineage

go 1.21.1

replace github.com/getsynq/api => ../../gen/

require (
	github.com/getsynq/api v0.0.0-20240118130644-9d44c1f0e96d
	golang.org/x/oauth2 v0.13.0
	google.golang.org/grpc v1.60.1
)

require (
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.16.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)
