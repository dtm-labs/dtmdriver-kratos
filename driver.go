package driver

import (
	"context"
	"fmt"
	"github.com/dtm-labs/dtmdriver"
	consul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	_ "github.com/go-kratos/kratos/v2/transport/grpc/resolver/direct"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	"github.com/google/uuid"
	consulAPI "github.com/hashicorp/consul/api"
	etcdAPI "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
	"net/url"
	"os"
	"strings"
)

const (
	DriverName    = "dtm-driver-kratos2"
	DefaultScheme = "discovery"
	EtcdScheme    = "etcd"
	ConsulScheme  = "consul"
)

type kratosDriver struct{}

func (k *kratosDriver) GetName() string {
	return DriverName
}

func (k *kratosDriver) RegisterAddrResolver() {

}

func (k *kratosDriver) RegisterService(target string, endpoint string) error {
	if target == "" {
		return nil
	}

	u, err := url.Parse(target)
	if err != nil {
		return err
	}

	// --  @# 从环境变量获取pod ip然后组合为endpoint
	theEndPoint := "grpc://" + os.Getenv("POD_IP") + ":36790"

	switch u.Scheme {
	case DefaultScheme:
		fallthrough
	case EtcdScheme:
		registerInstance := &registry.ServiceInstance{
			ID:        uuid.New().String(),
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: []string{theEndPoint},
		}
		client, err := etcdAPI.New(etcdAPI.Config{
			Endpoints: strings.Split(u.Host, ","),
		})
		if err != nil {
			return err
		}
		registry := etcd.New(client)
		//add resolver so that dtm can handle discovery://
		resolver.Register(discovery.NewBuilder(registry, discovery.WithInsecure(true)))
		return registry.Register(context.Background(), registerInstance)

	case ConsulScheme:
		registerInstance := &registry.ServiceInstance{
			ID:        uuid.New().String(),
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: []string{theEndPoint},
		}
		client, err := consulAPI.NewClient(&consulAPI.Config{Address: u.Host})
		if err != nil {
			return err
		}
		registry := consul.New(client)
		//add resolver so that dtm can handle discovery://
		resolver.Register(discovery.NewBuilder(registry, discovery.WithInsecure(true)))
		return registry.Register(context.Background(), registerInstance)
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
