module github.com/fulltimelink/dtmdriver-kratos

go 1.15

require (
	github.com/dtm-labs/dtmdriver v0.0.6
	github.com/go-kratos/kratos/contrib/polaris/v2 v2.0.0-20240918015945-e1f5dc42b1e5
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20220414054820-d0b704b8f38d
	github.com/go-kratos/kratos/contrib/registry/etcd/v2 v2.0.0-20220301040457-03ad2b663624
	github.com/go-kratos/kratos/v2 v2.8.0
	github.com/google/uuid v1.4.0
	github.com/hashicorp/consul/api v1.12.0
	github.com/polarismesh/polaris-go v1.3.0
	go.etcd.io/etcd/client/v3 v3.5.2
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	google.golang.org/grpc v1.61.1
)
