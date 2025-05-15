package transport

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const invalidPEMCert = `-----BEGIN CERTIFICATE-----
MIIBszCCAVmgAwIBAgIUQWnMEj6V3RQI9z5Hc3+qGKDdJOQwCgYIKoZIzj0EAwIw
EjEQMA4GA1UEAwwHcm9vdENBMB4XDTIxMDUyMTA5MjEyOFoXDTMxMDUxOTA5MjEy
OFowEjEQMA4GA1UEAwwHcm9vdENBMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAExHa7
XjgUacgMRR9w+OXYy1Pu67n0LlgFzqX+gDVoEWSODVd19Bx8I59M/TJjSzE0sF8T
twUjg6ezd9bDSR2FyyHRt4Nuv6R/kD3b9M+oH10wczELMAkGA1UdEwQCMAAwDgYD
VR0PAQH/BAQDAgeAMB0GA1UdDgQWBBR29oUT2Yw+xzd6OIp5A/HVnhNhYjAfBgNV
HSMEGDAWgBR29oUT2Yw+xzd6OIp5A/HVnhNhYjAKBggqhkjOPQQDAgNHADBEAiAk
F4eGc8JHSKZ6KU9QHfn3ev95tr6NV0uXwB+5Ciu4kwIgCS48S/jHcUuR+EoXfzX2
Q6KhqCcBnBJoBfaAsV7tSFA=
-----END CERTIFICATE-----`

const validPEMCert = `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUHUPkerrOvHfqMIQRJOXMyt9Db90wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI1MDUwNDEwMjI1MFoXDTI2MDUw
NDEwMjI1MFowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEA1mPxcihKFcjNUsQCx/3EsXib2tdRnfXdwhBeGtfPEfPD
6PRFwK1BaJECv7h5MYr3lmQlSYXaM69SFcE6jBb/SOFOTknbViHS18AnoGVCMb9w
nJP2cosv00b+Do1Lkh3vHlhn4KovlT1xquNkHDDRe1TjKec0aZcXVF4kkebGIXEs
FH01nItvaB/msKrKSZZVvELrgsJdB5WQwUOB4Eq/WlCDvdtHK+TxgJ676iJl/Cz8
8ZaFxz2DWQrPdqQ78BU0Min6XbqLg+kE+NXJ4nglWGmPxlatyNrelF9ZLssBr2Mo
Rmce6d9WneN7iJkuWn2YKPdUMpGN+h5LolfXmD4psQIDAQABo1MwUTAdBgNVHQ4E
FgQUXk6pM2YNBr9XpdV2ELAzcdakzKIwHwYDVR0jBBgwFoAUXk6pM2YNBr9XpdV2
ELAzcdakzKIwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAXsoZ
yzhrtsKyHx8XZHXlQySIqudnE90jHBw//okxZdQazK2YyKcMbXZy5723HNvnpmaj
QnkRvTDATv4pBcjJvAjlImo3C2qG96Fv6ySuuhFZcjqi2syPXCq/U9krcf6khtEs
N0EJZ6zIKotGnGLsKH62OZr60eY7IXbPqCUugp7h+RUNIoowh6tb526/2OiSef4S
4YEuOW4Mt3f9OUSw5efr2OoJRC44nZjeIKSqhs6gZJ33DstUQcP7gkQfmo113I9h
lP+COEFL7FKshkfwdns1T5lzmL6fxIghDdX1Wv5TF22qH2Iuz/AuZDAzrfzOk+yL
xsuQAC8/6BO0r72KGw==
-----END CERTIFICATE-----
`

func TestBuildTransportTLS(t *testing.T) {

	t.Run("BuildTransportTLS_InsecureTransport", func(t *testing.T) {
		cfg := TlsConfig{
			Insecure: true,
			Endpoint: "https://example.com",
		}
		transport, err := BuildTransportTLS(cfg)
		require.NoError(t, err)
		require.NotNil(t, transport)
		require.NotNil(t, transport.TLSClientConfig)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify, "InsecureSkipVerify should be true")
	})

	t.Run("BuildTransportTLS_ValidPEM", func(t *testing.T) {
		cfg := TlsConfig{
			Insecure: false,
			CACert:   validPEMCert,
			Endpoint: "https://example.com",
		}
		transport, err := BuildTransportTLS(cfg)
		require.NoError(t, err)
		require.NotNil(t, transport)
		require.NotNil(t, transport.TLSClientConfig)
		assert.False(t, transport.TLSClientConfig.InsecureSkipVerify, "InsecureSkipVerify should be false")
	})

	t.Run("BuildTransportTLS_EncodedPEM", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte(validPEMCert))
		cfg := TlsConfig{
			Insecure: false,
			CACert:   encoded,
			Endpoint: "https://example.com",
		}
		transport, err := BuildTransportTLS(cfg)
		require.NoError(t, err)
		require.NotNil(t, transport)
		require.NotNil(t, transport.TLSClientConfig)
		assert.False(t, transport.TLSClientConfig.InsecureSkipVerify, "InsecureSkipVerify should be false")
	})

	t.Run("BuildTransportTLS_InvalidBase64", func(t *testing.T) {
		cfg := TlsConfig{
			Insecure: false,
			CACert:   "not-base64",
			Endpoint: "https://example.com",
		}
		transport, err := BuildTransportTLS(cfg)
		require.Error(t, err)
		assert.Nil(t, transport)
		assert.Contains(t, err.Error(), "failed to decode CA cert")
	})

	t.Run("BuildTransportTLS_InvalidPEM", func(t *testing.T) {
		invalidPEM := invalidPEMCert
		encoded := base64.StdEncoding.EncodeToString([]byte(invalidPEM))
		cfg := TlsConfig{
			Insecure: false,
			CACert:   encoded,
			Endpoint: "https://example.com",
		}
		transport, err := BuildTransportTLS(cfg)
		require.Error(t, err)
		assert.Nil(t, transport)
		assert.Contains(t, err.Error(), "failed to append CA cert")
	})

	t.Run("BuildTransportTLS_MissingPEM", func(t *testing.T) {
		cfg := TlsConfig{
			Insecure: false,
			Endpoint: "https://example.com",
		}
		transport, err := BuildTransportTLS(cfg)
		require.Error(t, err)
		assert.Nil(t, transport)
		assert.Contains(t, err.Error(), "failed to append CA cert")
	})
}
