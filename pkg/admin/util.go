package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/transport"
)

var (
	ErrNoEndpoint   = errors.New("Nutanix object store instance endpoint not set")
	ErrNoAccessKey  = errors.New("Admin IAM access key for Nutanix Objects not set")
	ErrNoSecretKey  = errors.New("Admin IAM secret key for Nutanix Objects not set")
	ErrNoPCEndpoint = errors.New("Prism Central endpoint for IAM user management not set")
	ErrNoPCUsername = errors.New("Prism Central username for IAM user management not set")
	ErrNoPCPassword = errors.New("Prism Central password for IAM user management not set")
	ErrNoPCCreds = errors.New("Prism Central credentials not set: provide either a Service Account API key or a username/password pair")
)

// pcAPIKeyHeader is the HTTP header used by PC Service Account
// authentication. When PCAPIKey is non-empty on the API struct, every
// outgoing request to the IAM proxy carries this header instead of HTTP
// Basic Auth.
const pcAPIKeyHeader = "X-ntnx-api-key"

// HTTPClient interface that conforms to that of the http package's Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type IAMiface interface {
	CreateUser(ctx context.Context, username string, display_name string) (UserCredentials, error)
	RemoveUser(ctx context.Context, uuid string) error
	GetAccountName() string
	GetEndpoint() string
}

// API struct for New Client
type API struct {
	AccessKey   string
	SecretKey   string
	Endpoint    string
	PCEndpoint  string
	PCUsername  string
	PCPassword  string
	PCAPIKey    string
	AccountName string
	HTTPClient  HTTPClient
}

// New returns client for Nutanix object store
func New(endpoint, accessKey, secretKey, pcEndpoint, pcUsername, pcPassword, pcAPIKey, accountName, caCert string, insecure bool, httpClient HTTPClient) (*API, error) {
	// validate endpoint
	if endpoint == "" {
		return nil, ErrNoEndpoint
	}

	// validate access key
	if accessKey == "" {
		return nil, ErrNoAccessKey
	}

	// validate secret key
	if secretKey == "" {
		return nil, ErrNoSecretKey
	}

	// validate pc endpoint
	if pcEndpoint == "" {
		return nil, ErrNoPCEndpoint
	}

	// validate pc credentials: either an API key (preferred) or a
	// username/password pair must be supplied. When both are absent we
	// cannot authenticate to the IAM proxy at all, so this is fatal.
	if pcAPIKey == "" && pcUsername == "" && pcPassword == "" {
		return nil, ErrNoPCCreds
	}
	// When no API key is provided we fall back to Basic Auth and require
	// both halves of the credential pair to be present.
	if pcAPIKey == "" {
		if pcUsername == "" {
			return nil, ErrNoPCUsername
		}
		if pcPassword == "" {
			return nil, ErrNoPCPassword
		}
	}

	// set default account_name when empty
	if accountName == "" {
		accountName = "ntnx-cosi-iam-user"
	}

	tlsConfig := transport.TlsConfig{
		CACert:   caCert,
		Insecure: insecure,
		Endpoint: pcEndpoint,
	}

	transport, err := transport.BuildTransportTLS(tlsConfig)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: transport,
	}

	return &API{
		Endpoint:    endpoint,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		PCEndpoint:  pcEndpoint,
		PCUsername:  pcUsername,
		PCPassword:  pcPassword,
		PCAPIKey:    pcAPIKey,
		AccountName: accountName,
		HTTPClient:  client,
	}, nil
}

// Authenticate attaches the appropriate authentication header to req
// based on which PC credential is populated on the API struct. The PC
// Service Account API key wins when set; otherwise we fall back to HTTP
// Basic Auth using PCUsername/PCPassword.
//
// This is the single source of truth for PC auth on outgoing requests
// so callers in user.go and the e2e helpers don't have to know which
// credential is in play.
func (api *API) Authenticate(req *http.Request) {
	if api.PCAPIKey != "" {
		req.Header.Set(pcAPIKeyHeader, api.PCAPIKey)
		return
	}
	req.SetBasicAuth(api.PCUsername, api.PCPassword)
}

func GetCredsFromPCSecret(key string) (string, string, error) {

	// Split using ":" as delimiter
	creds := strings.SplitN(string(key), ":", 2)
	if len(creds) != 2 {
		return "", "", fmt.Errorf("missing information in secret value '<pc_user>:<pc_password>'")
	}

	return creds[0], creds[1], nil
}

// Validate endpoint
func ValidateEndpoint(endpoint string) error {
	if len(endpoint) == 0 {
		return fmt.Errorf("endpoint is not specified")
	}

	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return fmt.Errorf("error while resolving endpoint %s, err: %s", endpoint, err)
	}

	return nil
}

func (api *API) GetAccountName() string {
	return api.AccountName
}

func (api *API) GetEndpoint() string {
	return api.Endpoint
}
