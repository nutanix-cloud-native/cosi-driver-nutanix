package admin_test

import (
	"net/http"
	"testing"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validPEMCert = `-----BEGIN CERTIFICATE-----
MIIDVDCCAjygAwIBAgIRALLmZX4DorTkHZzNVhA+RYUwDQYJKoZIhvcNAQELBQAw
FzEVMBMGA1UEChMMTnV0YW5peCBJbmMuMCAXDTI1MDEyNzA4MDU0NVoYDzIyMDQw
NzAzMDgwNTQ1WjAXMRUwEwYDVQQKEwxOdXRhbml4IEluYy4wggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQDXvGBSfAByt1VCvAuaDThq7H+JlK8He8zcjqD7
DGe6Dc1jq9n7+eN6X+cTT85dKPhajjPp9Nc4g1HT0jfWHD4QHhgkS12Ny0Wrmqqr
qx8Fcuzhaz89BFyaI3tOCBx75yZe9zaNSOBoKJqreu2mAjfI8LM7jFl/cON68Kd9
/b6g89+2FME0gFq2p3mabGXGVerFAK0g7TuffhWKyKL/B9equZ2M7CmhAO7wa0ko
CnQJY3XvNk4OUeb7NXBdpswpWD789rXSsSMbxaS9ZmDrqvB7A1IlWjwEUVzxYnPw
21miqfZ5qb2p3gZtSTbDOqmg0l9yfvwIroaGKMLrRCK0Q+IjAgMBAAGjgZgwgZUw
DgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFPE9h85q
h53rKbrBRd+VnUGuAxpKMFMGA1UdEQRMMEqCInJpbXVydS5wcmlzbS1jZW50cmFs
LmNsdXN0ZXIubG9jYWyCJCoucmltdXJ1LnByaXNtLWNlbnRyYWwuY2x1c3Rlci5s
b2NhbDANBgkqhkiG9w0BAQsFAAOCAQEAd3wN98xaTJqBGE2j0qoSQhqMNb5NmBDa
Cp/Pt0mwlAJmv2petz3ON5edt3/yC81vEvWfT+4GpM/6jAHcY9rZ+XQA+ZkSnjsG
itALgSLq77vDYRTHAXfsWPH2DY140IS6OqqTtLPLukHzux5uR2LH1uggU5sARs5l
EBi1znwsnSxrKfqPOurt4oSgW7FougqiaOiK+Vkm+1FtybVlMXH1w5TkePFK/x7B
OkiKpPoALmPy1Y2BxvbpxQYLjZEFMKwIo7G20pl9opFntCBs6GcY7QNesVYKawV1
zQEsbJYBuhj1XgjzRx+6al2Fjf2NFN3I2aCKQZ9oMFsg0R/M0biBjA==
-----END CERTIFICATE-----
`

func TestNew(t *testing.T) {
	// Valid parameters for testing.
	validEndpoint := "https://objectore.example.com"
	validAccessKey := "dummyAccessKey"
	validSecretKey := "dummySecretKey"
	validPCEndpoint := "https://prism.example.com"
	validPCUsername := "admin"
	validPCPassword := "password"
	validAccountName := "custom-account"

	t.Run("TestNew_DefaultAccount", func(t *testing.T) {
		api, err := admin.New(validEndpoint, validAccessKey, validSecretKey, validPCEndpoint, validPCUsername, validPCPassword, "", validPEMCert, false, nil)

		_, ok := api.HTTPClient.(*http.Client)
		require.True(t, ok)
		require.NotNil(t, api)
		require.NoError(t, err)
		assert.Equal(t, validEndpoint, api.Endpoint)
		assert.Equal(t, validAccessKey, api.AccessKey)
		assert.Equal(t, validSecretKey, api.SecretKey)
		assert.Equal(t, validPCEndpoint, api.PCEndpoint)
		assert.Equal(t, validPCUsername, api.PCUsername)
		assert.Equal(t, validPCPassword, api.PCPassword)
		assert.Equal(t, "ntnx-cosi-iam-user", api.AccountName)
	})

	t.Run("TestNew_CustomAccount", func(t *testing.T) {
		api, err := admin.New(validEndpoint, validAccessKey, validSecretKey, validPCEndpoint, validPCUsername, validPCPassword, validAccountName, validPEMCert, false, nil)

		require.NoError(t, err)
		require.NotNil(t, api)
		assert.Equal(t, validAccountName, api.AccountName)
	})

	t.Run("TestNew_EmptyEndpoint", func(t *testing.T) {
		api, err := admin.New("", validAccessKey, validSecretKey, validPCEndpoint, validPCUsername, validPCPassword, validAccountName, validPEMCert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Equal(t, admin.ErrNoEndpoint, err)
	})

	t.Run("TestNew_EmptyAccessKey", func(t *testing.T) {
		api, err := admin.New(validEndpoint, "", validSecretKey, validPCEndpoint, validPCUsername, validPCPassword, validAccountName, validPEMCert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Equal(t, admin.ErrNoAccessKey, err)
	})

	t.Run("TestNew_EmptySecretKey", func(t *testing.T) {
		api, err := admin.New(validEndpoint, validAccessKey, "", validPCEndpoint, validPCUsername, validPCPassword, validAccountName, validPEMCert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Equal(t, admin.ErrNoSecretKey, err)
	})

	t.Run("TestNew_EmptyPCEndpoint", func(t *testing.T) {
		api, err := admin.New(validEndpoint, validAccessKey, validSecretKey, "", validPCUsername, validPCPassword, validAccountName, validPEMCert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Equal(t, admin.ErrNoPCEndpoint, err)
	})

	t.Run("TestNew_EmptyPCUsername", func(t *testing.T) {
		api, err := admin.New(validEndpoint, validAccessKey, validSecretKey, validPCEndpoint, "", validPCPassword, validAccountName, validPEMCert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Equal(t, admin.ErrNoPCUsername, err)
	})

	t.Run("TestNew_EmptyPCPassword", func(t *testing.T) {
		api, err := admin.New(validEndpoint, validAccessKey, validSecretKey, validPCEndpoint, validPCUsername, "", validAccountName, validPEMCert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Equal(t, admin.ErrNoPCPassword, err)
	})

	t.Run("TestNew_BadCACert", func(t *testing.T) {
		invalidCACert := "invalid-cert"
		api, err := admin.New(validEndpoint, validAccessKey, validSecretKey, validPCEndpoint, validPCUsername, validPCPassword, validAccountName, invalidCACert, false, nil)
		require.Error(t, err)
		assert.Nil(t, api)
		assert.Contains(t, err.Error(), "failed to decode CA cert")
	})
}

func TestGetCredsFromPCSecret(t *testing.T) {
	t.Run("TestGetCredsFromPCSecret_Valid", func(t *testing.T) {
		// Using localhost as a resolvable IP
		key := "admin:password"
		username, password, err := admin.GetCredsFromPCSecret(key)
		require.NoError(t, err)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "password", password)
	})

	t.Run("TestGetCredsFromPCSecret_MissingFields", func(t *testing.T) {
		key := "admin"
		username, password, err := admin.GetCredsFromPCSecret(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing information in secret value")
		assert.Empty(t, username)
		assert.Empty(t, password)
	})
}

func TestValidateEndpoint(t *testing.T) {
	t.Run("TestValidateEndpoint_ValidEndpoint", func(t *testing.T) {
		err := admin.ValidateEndpoint("https://127.0.0.1")
		assert.NoError(t, err)
	})

	t.Run("TestValidateEndpoint_EmptyString", func(t *testing.T) {
		err := admin.ValidateEndpoint("")
		require.Error(t, err)
		assert.Equal(t, "endpoint is not specified", err.Error())
	})

	t.Run("TestValidateEndpoint_InvalidEndpoint", func(t *testing.T) {
		err := admin.ValidateEndpoint("https//thisisaninvalid.endpoint")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error while resolving endpoint")
	})
}

func TestGetAccountName(t *testing.T) {
	t.Run("TestGetAccountName_Success", func(t *testing.T) {
		api := admin.API{
			AccountName: "test",
		}
		accountName := api.AccountName
		assert.Equal(t, "test", accountName)
	})

	t.Run("TestGetAccountName_MissingAccountName", func(t *testing.T) {
		api := admin.API{}
		accountName := api.AccountName
		assert.Empty(t, accountName)
	})
}

func TestGetEndpoint(t *testing.T) {
	t.Run("TestGetEndpoint_Success", func(t *testing.T) {
		api := admin.API{
			Endpoint: "test",
		}
		endpoint := api.GetEndpoint()
		assert.Equal(t, "test", endpoint)
	})

	t.Run("TestGetEndpoint_MissingEndpoint", func(t *testing.T) {
		api := admin.API{}
		endpoint := api.GetEndpoint()
		assert.Empty(t, endpoint)
	})
}
