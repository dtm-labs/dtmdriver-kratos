package driver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/url"
	"strings"

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
)

const (
	DriverName    = "dtm-driver-kratos"
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
	//cut line break,support multi line
	target = strings.ReplaceAll(target, "\n", "")
	target = strings.ReplaceAll(target, "\r", "")

	u, err := url.Parse(target)
	if err != nil {
		return err
	}

	tlsConf := &TlsConfig{}
	tlsEnable := isTlsEnable(u)
	if tlsEnable {
		tlsConf, err = paseTargetUrlForTls(u)
		if err != nil {
			return err
		}
	}

	switch u.Scheme {
	case DefaultScheme:
		fallthrough
	case EtcdScheme:
		newUUID, err := uuid.NewUUID()
		if err != nil {
			return err
		}
		registerInstance := &registry.ServiceInstance{
			ID:        newUUID.String(),
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: strings.Split(endpoint, ","),
		}
		var Certificates []tls.Certificate
		var caPool = &x509.CertPool{}
		var client *etcdAPI.Client
		if tlsEnable {
			caPool, err = loadCaPool(tlsConf)
			if err != nil {
				return err
			}

			cert := &tls.Certificate{}
			cert, err = loadCertificate(tlsConf)
			if err != nil {
				return err
			}
			Certificates = append(Certificates, *cert)
			client, err = etcdAPI.New(etcdAPI.Config{
				Endpoints: strings.Split(u.Host, ","),
				TLS: &tls.Config{
					RootCAs:      caPool,
					Certificates: Certificates,
				},
			})
			if err != nil {
				return err
			}
		} else {
			client, err = etcdAPI.New(etcdAPI.Config{
				Endpoints: strings.Split(u.Host, ","),
			})
		}

		registry := etcd.New(client)
		//add resolver so that dtm can handle discovery://
		resolver.Register(discovery.NewBuilder(registry, discovery.WithInsecure(true)))
		err = registry.Register(context.Background(), registerInstance)
		if err != nil {
			log.Println("register instance error: %v", err)
			return err
		}
		return nil

	case ConsulScheme:
		registerInstance := &registry.ServiceInstance{
			Name:      strings.TrimPrefix(u.Path, "/"),
			Endpoints: strings.Split(endpoint, ","),
		}

		var client *consulAPI.Client
		if tlsEnable {
			client, err = consulAPI.NewClient(&consulAPI.Config{
				Address: u.Host,
				TLSConfig: consulAPI.TLSConfig{
					CAFile:   tlsConf.CaPath,
					CertFile: tlsConf.CertPath,
					KeyFile:  tlsConf.CertKeyPath,
				},
			})
		} else {
			client, err = consulAPI.NewClient(&consulAPI.Config{
				Address: u.Host,
			})
		}
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
