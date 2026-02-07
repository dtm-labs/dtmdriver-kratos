package driver

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"os"

	"github.com/pkg/errors"
)

type TlsConfig struct {
	CaPath      string
	CertPath    string
	CertKeyPath string
}

func isTlsEnable(url *url.URL) bool {
	return url.Query().Get("tls") == "true"
}

func paseTargetUrlForTls(url *url.URL) (*TlsConfig, error) {
	var config TlsConfig
	caCrt := url.Query().Get("caPath")
	_, err := os.Stat(caCrt)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err, "ca  file not found")
		}
		return nil, errors.Wrap(err, "read ca file failed, check your permission")
	}
	config.CaPath = caCrt

	//sometimes client just need ca.crt，but if you enable  Two-way Authentication
	//you also need cert.crt and certKey.key
	certPath := url.Query().Get("certPath")
	if certPath == "" {
		return &config, nil
	}
	_, err = os.Stat(certPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err, "cert file not found")
		}
		return nil, errors.Wrap(err, "read cert file failed, check your permission")
	}
	config.CertPath = certPath

	certKeyPath := url.Query().Get("certKeyPath")
	if certKeyPath == "" {
		return &config, nil
	}
	_, err = os.Stat(certKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err, "key file not found")
		}
		return nil, errors.Wrap(err, "read key file failed, check your permission")
	}
	config.CertKeyPath = certKeyPath

	return &config, nil
}

func loadCaPool(tlsConfig *TlsConfig) (*x509.CertPool, error) {
	if tlsConfig.CaPath == "" {
		return nil, errors.New("you enable tls,but  caPath is empty")
	}
	var caPool *x509.CertPool
	caData, err := os.ReadFile(tlsConfig.CaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err, "ca  file not found")
		}
		return nil, errors.Wrap(err, "read ca file failed, check your permission")
	}

	//add ca to x509.CertPool
	caPool = x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caData) {
		return nil, errors.New("append ca to x509.CertPool failed")
	}
	return caPool, nil
}

func loadCertificate(tlsConfig *TlsConfig) (*tls.Certificate, error) {
	if tlsConfig.CertPath != "" && tlsConfig.CertKeyPath != "" {
		clientCert, err := tls.LoadX509KeyPair(tlsConfig.CertPath, tlsConfig.CertKeyPath)
		if err != nil {
			return nil, errors.Wrap(err, "load x509 key pair failed")
		}
		return &clientCert, nil
	}

	return &tls.Certificate{}, nil
}
