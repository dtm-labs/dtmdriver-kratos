package driver

import (
	"context"
	"errors"
	"fmt"
	"github.com/dtm-labs/dtmdriver"
	consul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	nacos "github.com/go-kratos/kratos/contrib/registry/nacos/v2"
	"github.com/go-kratos/kratos/v2/registry"
	_ "github.com/go-kratos/kratos/v2/transport/grpc/resolver/direct"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/nacos-group/nacos-sdk-go/clients"
	nacosconstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	etcdAPI "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
	"net/url"
	"strconv"
	"strings"
)

const (
	DriverName    = "dtm-driver-kratos"
	DefaultScheme = "discovery"
	EtcdScheme    = "etcd"
	ConsulScheme  = "consul"
	NacosScheme   = "nacos"
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
	switch u.Scheme {
	case DefaultScheme:
		fallthrough
	case EtcdScheme:
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
		registry := etcd.New(client)
		//add resolver so that dtm can handle discovery://
		resolver.Register(discovery.NewBuilder(registry, discovery.WithInsecure(true)))
		return registry.Register(context.Background(), registerInstance)
	case ConsulScheme:
		registerInstance := &registry.ServiceInstance{
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: strings.Split(endpoint, ","),
		}
		client, err := consulAPI.NewClient(&consulAPI.Config{Address: u.Host})
		if err != nil {
			return err
		}
		registry := consul.New(client)
		//add resolver so that dtm can handle discovery://
		resolver.Register(discovery.NewBuilder(registry, discovery.WithInsecure(true)))
		return registry.Register(context.Background(), registerInstance)
	case NacosScheme:
		return k.nacosRegister(u, endpoint)
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

func (k *kratosDriver) nacosRegister(url *url.URL, endpoint string) (err error) {
	registerInstance := &registry.ServiceInstance{
		Name:      strings.TrimPrefix(url.Path, "/"),
		Endpoints: strings.Split(endpoint, ","),
	}

	hostSplit := strings.Split(url.Host, ":")
	if len(hostSplit) != 2 {
		return errors.New("nacos host format error")
	}
	ipAddr, port := hostSplit[0], uint64(8848)
	port, err = strconv.ParseUint(hostSplit[1], 10, 64)
	if err != nil {
		return
	}

	sc := []nacosconstant.ServerConfig{
		*nacosconstant.NewServerConfig(ipAddr, port),
	}

	namespaceId := url.Query().Get("namespaceId")
	if namespaceId == "" {
		namespaceId = "public"
	}

	timeoutMs := uint64(5000)
	if url.Query().Get("timeoutMs") != "" {
		timeoutMs, err = strconv.ParseUint(url.Query().Get("timeoutMs"), 10, 64)
		if err != nil {
			return
		}
	}

	notLoadCacheAtStart := strings.ToLower(url.Query().Get("notLoadCacheAtStart")) != "false"

	cc := nacosconstant.NewClientConfig(
		nacosconstant.WithNamespaceId(namespaceId),
		nacosconstant.WithTimeoutMs(timeoutMs),
		nacosconstant.WithNotLoadCacheAtStart(notLoadCacheAtStart),
	)

	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ServerConfigs: sc,
			ClientConfig:  cc,
		},
	)
	if err != nil {
		return err
	}

	r := nacos.New(client)

	resolver.Register(discovery.NewBuilder(r, discovery.WithInsecure(true)))

	return r.Register(context.Background(), registerInstance)
}

func init() {
	dtmdriver.Register(&kratosDriver{})
}
