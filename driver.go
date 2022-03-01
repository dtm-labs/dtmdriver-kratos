package driver

import (
	"context"
	"fmt"
	"github.com/dtm-labs/dtmdriver"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
	"net/url"
	"strings"
)

const (
	DriverName = "dtm-driver-kratos"
	KindEtcd   = "etcd"
)

type kratosDriver struct{}

func (k *kratosDriver) GetName() string {
	return DriverName
}

func (k *kratosDriver) RegisterGrpcResolver() {
	resolver.Register(discovery.NewBuilder(etcd.New(clientv3.NewCtxClient(context.Background()))))
}

func (k *kratosDriver) RegisterGrpcService(target string, endpoint string) error {
	if target == "" {
		return nil
	}

	u, err := url.Parse(target)
	if err != nil {
		return err
	}

	registerInstance := &registry.ServiceInstance{
		Name:      strings.TrimPrefix(u.Path, "/"),
		Endpoints: strings.Split(endpoint, ","),
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(u.Host, ","),
	})

	switch u.Scheme {
	case KindEtcd:
		return etcd.New(client).Register(context.Background(), registerInstance)
	default:
		return fmt.Errorf("unknown scheme: %s", u.Scheme)
	}
}

func (k *kratosDriver) ParseServerMethod(uri string) (server string, method string, err error) {
	if !strings.Contains(uri, "//") {
		sep := strings.IndexByte(uri, '/')
		if sep == -1 {
			return "", "", fmt.Errorf("bad url: '%s'. no '/' found", uri)
		}
		return uri[:sep], uri[sep:], nil

	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", nil
	}
	index := strings.IndexByte(u.Path[1:], '/') + 1
	return u.Scheme + "://" + u.Host + u.Path[:index], u.Path[index:], nil
}

func init() {
	dtmdriver.Register(&kratosDriver{})
}
