package transport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

type TlsConfig struct {
	CACert   string
	Insecure bool
	Endpoint string
}

func BuildTransportTLS(tlsConfig TlsConfig) (*http.Transport, error) {
	var transport *http.Transport

	if tlsConfig.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		klog.InfoS("insecure connection made.", "insecure", tlsConfig.Insecure, "endpoint", tlsConfig.Endpoint)

		return transport, nil
	}

	if tlsConfig.CACert == "" {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		}

		klog.InfoS("secure connection made without CA certs.", "insecure", tlsConfig.Insecure, "endpoint", tlsConfig.Endpoint)

		return transport, nil
	}

	var rootCAs []byte
	if strings.Contains(tlsConfig.CACert, "-----BEGIN CERTIFICATE-----") && strings.Contains(tlsConfig.CACert, "-----END CERTIFICATE-----") {
		rootCAs = []byte(tlsConfig.CACert)
	} else {
		// Decode base64 CA cert
		_rootCAs, err := base64.StdEncoding.DecodeString(tlsConfig.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CA cert: %v", err)
		}

		rootCAs = _rootCAs
	}

	// Create cert pool and add our CA
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(rootCAs) {
		return nil, fmt.Errorf("failed to append CA cert: %s", tlsConfig.CACert)
	}

	transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: false,
		},
	}

	klog.InfoS("secure connection made with CA certs.", "insecure", tlsConfig.Insecure, "endpoint", tlsConfig.Endpoint)

	return transport, nil
}
