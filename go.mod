module github.com/dtm-labs/dtmdriver-kratos

go 1.15

require (
	github.com/dtm-labs/dtmdriver v0.0.1
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20220410081856-3990d91b9bd3
	github.com/go-kratos/kratos/contrib/registry/etcd/v2 v2.0.0-20220301040457-03ad2b663624
	github.com/go-kratos/kratos/v2 v2.2.1
	github.com/hashicorp/consul/api v1.9.1
	go.etcd.io/etcd/client/v3 v3.5.2
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	google.golang.org/genproto v0.0.0-20220228195345-15d65a4533f7 // indirect
	google.golang.org/grpc v1.44.0
)
