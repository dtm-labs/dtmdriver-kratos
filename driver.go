package driver

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/dtm-labs/dtmdriver"
	consul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	_ "github.com/go-kratos/kratos/v2/transport/grpc/resolver/direct"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	consulAPI "github.com/hashicorp/consul/api"
	etcdAPI "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

const (
	DriverName    = "dtm-driver-kratos"
	DefaultScheme = "discovery"
	EtcdScheme    = "etcd"
	ConsulScheme  = "consul"
)

var builders sync.Map

type kratosEtcdBuilder struct{}

func (b *kratosEtcdBuilder) newBuilder(endpoint string) resolver.Builder {
	client, _ := etcdAPI.New(etcdAPI.Config{
		Endpoints: strings.Split(endpoint, ","),
	})
	return discovery.NewBuilder(etcd.New(client), discovery.WithInsecure(true))
}

func (b *kratosEtcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	endpoint := target.URL.Host
	builder, ok := builders.Load(endpoint)
	if !ok {
		builder = b.newBuilder(endpoint)
		builders.Store(endpoint, builder)
	}
	return builder.(resolver.Builder).Build(target, cc, opts)
}

func (b *kratosEtcdBuilder) Scheme() string {
	return EtcdScheme
}

type kratosConsulBuilder struct{}

func (b *kratosConsulBuilder) newBuilder(endpoint string) resolver.Builder {
	client, _ := consulAPI.NewClient(&consulAPI.Config{Address: endpoint})
	return discovery.NewBuilder(consul.New(client), discovery.WithInsecure(true))
}

func (b *kratosConsulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	endpoint := target.URL.Host
	builder, ok := builders.Load(endpoint)
	if !ok {
		builder = b.newBuilder(endpoint)
		builders.Store(endpoint, builder)
	}
	reso, err := builder.(resolver.Builder).Build(target, cc, opts)
	return reso, err
}

func (b *kratosConsulBuilder) Scheme() string {
	return ConsulScheme
}

type kratosDefaultBuilder struct {
	builder resolver.Builder
}

func (b *kratosDefaultBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	return b.builder.Build(target, cc, opts)
}

func (b *kratosDefaultBuilder) Scheme() string {
	return DefaultScheme
}

type kratosDriver struct{}

func (k *kratosDriver) GetName() string {
	return DriverName
}

func (k *kratosDriver) RegisterGrpcResolver() {
	resolver.Register(&kratosEtcdBuilder{})
	resolver.Register(&kratosConsulBuilder{})
}

func (k *kratosDriver) RegisterGrpcService(target string, endpoint string) error {
	if target == "" {
		return nil
	}

	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case DefaultScheme:
		fallthrough
	case EtcdScheme:
		// etcd as registry, so register discovery:// to etcd resolver
		// so that dtm can handle discovery://
		defaultBuilder := &kratosDefaultBuilder{builder: &kratosEtcdBuilder{}}
		resolver.Register(defaultBuilder)

		registerInstance := &registry.ServiceInstance{
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: strings.Split(endpoint, ","),
		}
		client, err := etcdAPI.New(etcdAPI.Config{
			Endpoints: strings.Split(u.Host, ","),
		})
		if err != nil {
			return err
		}
		return etcd.New(client).Register(context.Background(), registerInstance)

	case ConsulScheme:
		// etcd as registry, so register discovery:// to consul resolver
		// so that dtm can handle discovery://
		defaultBuilder := &kratosDefaultBuilder{builder: &kratosConsulBuilder{}}
		resolver.Register(defaultBuilder)

		registerInstance := &registry.ServiceInstance{
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: strings.Split(endpoint, ","),
		}
		client, err := consulAPI.NewClient(&consulAPI.Config{Address: u.Host})
		if err != nil {
			return err
		}
		return consul.New(client).Register(context.Background(), registerInstance)
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
