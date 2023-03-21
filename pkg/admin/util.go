package admin

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

var (
	errNoEndpoint   = errors.New("Nutanix object store instance endpoint not set")
	errNoAccessKey  = errors.New("Admin IAM access key for Nutanix Objects not set")
	errNoSecretKey  = errors.New("Admin IAM secret key for Nutanix Objects not set")
	errNoPCEndpoint = errors.New("Prism Central endpoint for IAM user management not set")
	errNoPCUsername = errors.New("Prism Central username for IAM user management not set")
	errNoPCPassword = errors.New("Prism Central password for IAM user management not set")
)

// HTTPClient interface that conforms to that of the http package's Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// API struct for New Client
type API struct {
	AccessKey   string
	SecretKey   string
	Endpoint    string
	PCEndpoint  string
	PCUsername  string
	PCPassword  string
	AccountName string
	HTTPClient  HTTPClient
}

// New returns client for Nutanix object store
func New(endpoint, accessKey, secretKey, pcEndpoint, pcUsername, pcPassword, accountName string, httpClient HTTPClient) (*API, error) {
	// validate endpoint
	if endpoint == "" {
		return nil, errNoEndpoint
	}

	// validate access key
	if accessKey == "" {
		return nil, errNoAccessKey
	}

	// validate secret key
	if secretKey == "" {
		return nil, errNoSecretKey
	}

	// validate pc endpoint
	if pcEndpoint == "" {
		return nil, errNoPCEndpoint
	}

	// validate pc username
	if pcUsername == "" {
		return nil, errNoPCUsername
	}

	// validate pc password
	if pcPassword == "" {
		return nil, errNoPCPassword
	}

	// set default account_name when empty
	if accountName == "" {
		accountName = "ntnx-cosi-iam-user"
	}

	// If no client is passed initialize it
	if httpClient == nil {
		// SSL certificate verification turned off
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: tr}
	}

	return &API{
		Endpoint:    endpoint,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		PCEndpoint:  pcEndpoint,
		PCUsername:  pcUsername,
		PCPassword:  pcPassword,
		AccountName: accountName,
		HTTPClient:  httpClient,
	}, nil
}

func GetCredsFromPCSecret(key string) (string, string, string, error) {

	// Split using ":" as delimiter
	creds := strings.SplitN(string(key), ":", 4)
	if len(creds) != 4 {
		return "", "", "", fmt.Errorf("missing information in secret value '<prism-ip>:<prism-port>:<pc_user>:<pc_password>'")
	}

	// Validate Prism Endpoint
	err := ValidateEndpoint(creds[0])
	if err != nil {
		return "", "", "", err
	}

	return "https://" + creds[0] + ":" + creds[1], creds[2], creds[3], nil
}

// Validate endpoint is of form <ip or hostname>:<port>
func ValidateEndpoint(endpoint string) error {
	if len(endpoint) == 0 {
		return fmt.Errorf("endpoint is not specified")
	}

	// epList[0] should be an IP v4 address
	if _, err := net.ResolveIPAddr("ip", endpoint); err != nil {
		return fmt.Errorf("error while resolving endpoint %s, err: %s", endpoint, err)
	}

	return nil
}
