package admin_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	mocks "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/fakes"

	"github.com/stretchr/testify/assert"
)

var (
	ctx     = context.Background()
	baseApi = admin.API{
		PCEndpoint: "https://pc.example.com",
		PCUsername: "admin",
		PCPassword: "password",
	}
)

const (
	listUsersPath  = "/api/iam/v4.0/authn/users"
	createUserPath = "/api/iam/v4.0/authn/users"
	deleteUserBase = "/api/iam/v4.0/authn/users/"

	// mockEtag is the value emitted by the fake ETag header on every
	// GET-by-id response served in the unit tests. Its exact value is
	// irrelevant to the server-side contract; what we assert on is that
	// deleteAccessKey / RemoveUser round-trip it back as If-Match.
	mockEtag = "test-etag"
)

func userPathFor(extID string) string {
	return "/api/iam/v4.0/authn/users/" + extID
}

func keysPathFor(extID string) string {
	return "/api/iam/v4.0/authn/users/" + extID + "/keys"
}

func keyDeletePathFor(extID, keyExtID string) string {
	return "/api/iam/v4.0/authn/users/" + extID + "/keys/" + keyExtID
}

// etagResponse builds a mock 200 response that carries the Etag header
// callers echo back as If-Match. Used for GET-for-etag hops on both the
// user and key endpoints.
func etagResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Etag": []string{mockEtag}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

// listKeysWithOneBody is the v4 list-keys payload returned for an
// existing user that already has a single OBJECT_KEY on file, i.e. one
// that is still below the per-user key cap.
const listKeysWithOneBody = `{
	"metadata": {},
	"data": [
		{
			"extId": "old-key-ext-id",
			"name": "testuser",
			"keyType": "OBJECT_KEY",
			"createdTime": "2024-12-01T00:00:00Z",
			"keyDetails": {"$objectType": "iam.v4.authn.ObjectKeyDetails"}
		}
	]
}`

// listKeysAtCapBody is the v4 list-keys payload for a user that has hit
// the cap of 5 OBJECT_KEYs per user that Nutanix Objects enforces. Only
// in this state does CreateUser evict a key to make room for the new
// one, so the rotation tests need a full set rather than a single key.
// old-key-ext-id is first in the list because that is the entry
// CreateUser evicts.
const listKeysAtCapBody = `{
	"metadata": {},
	"data": [
		{"extId": "old-key-ext-id", "name": "testuser", "keyType": "OBJECT_KEY", "createdTime": "2024-12-01T00:00:00Z"},
		{"extId": "key-2", "name": "testuser", "keyType": "OBJECT_KEY", "createdTime": "2024-12-02T00:00:00Z"},
		{"extId": "key-3", "name": "testuser", "keyType": "OBJECT_KEY", "createdTime": "2024-12-03T00:00:00Z"},
		{"extId": "key-4", "name": "testuser", "keyType": "OBJECT_KEY", "createdTime": "2024-12-04T00:00:00Z"},
		{"extId": "key-5", "name": "testuser", "keyType": "OBJECT_KEY", "createdTime": "2024-12-05T00:00:00Z"}
	]
}`

// listKeysEmptyBody is the v4 list-keys payload returned for an
// existing user that has no OBJECT_KEYs on file.
const listKeysEmptyBody = `{
	"metadata": {},
	"data": []
}`

// listUsersEmptyBody is a v4 user-list payload that does NOT contain
// "testuser", forcing the create path to POST a new user.
const listUsersEmptyBody = `{
	"metadata": {},
	"data": [
		{"extId": "other-uuid", "username": "someoneelse", "userType": "EXTERNAL"}
	]
}`

// listUsersWithTestUserBody returns a v4 user-list payload that already
// contains "testuser" with extId existing-user-uuid.
const listUsersWithTestUserBody = `{
	"metadata": {},
	"data": [
		{"extId": "other-uuid", "username": "someoneelse", "userType": "EXTERNAL"},
		{"extId": "existing-user-uuid", "username": "testuser", "userType": "EXTERNAL"}
	]
}`

const createUserRespBody = `{
	"metadata": {},
	"data": {
		"extId": "user-uuid",
		"username": "testuser",
		"userType": "EXTERNAL",
		"displayName": "Test User"
	}
}`

const createKeyRespBody = `{
	"metadata": {},
	"data": {
		"extId": "key-ext-id",
		"name": "testuser",
		"keyType": "OBJECT_KEY",
		"createdTime": "2025-01-01T00:00:00Z",
		"keyDetails": {
			"$objectType": "iam.v4.authn.ObjectKeyDetails",
			"accessKey": "access-key",
			"secretKey": "secret-key"
		}
	}
}`

const createKeyRespBodyExistingUser = `{
	"metadata": {},
	"data": {
		"extId": "key-ext-id-2",
		"name": "testuser",
		"keyType": "OBJECT_KEY",
		"createdTime": "2026-05-06T08:48:19Z",
		"keyDetails": {
			"$objectType": "iam.v4.authn.ObjectKeyDetails",
			"accessKey": "new-access-key",
			"secretKey": "new-secret-key"
		}
	}
}`

func TestCreateUser(t *testing.T) {
	mockUsername := "testuser"
	mockDisplayName := "Test User"

	t.Run("TestCreateUser_Success", func(t *testing.T) {
		var sawList, sawCreateUser, sawCreateKey bool
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					sawList = true
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersEmptyBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == createUserPath:
					sawCreateUser = true
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createUserRespBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("user-uuid"):
					sawCreateKey = true
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createKeyRespBody)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		resp, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.True(t, sawList, "list users endpoint should have been called")
		assert.True(t, sawCreateUser, "create user endpoint should have been called")
		assert.True(t, sawCreateKey, "create key endpoint should have been called")
		assert.Equal(t, "user-uuid", resp.UserID)
		assert.Equal(t, "access-key", resp.AccessKeyID)
		assert.Equal(t, "secret-key", resp.SecretAccessKey)
	})

	// At the 5-key cap CreateUser has to free a slot before it can mint a
	// replacement, so it evicts one existing key and then creates the new
	// one.
	t.Run("TestCreateUser_ExistingUserAtKeyCapEvictsKey", func(t *testing.T) {
		var sawCreateUser, sawListKeys, sawGetKey, sawCreateKey bool
		var deleteKeyIfMatch string
		var deletedKeys []string
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersWithTestUserBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == createUserPath:
					sawCreateUser = true
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createUserRespBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("existing-user-uuid"):
					sawListKeys = true
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysAtCapBody)),
					}, nil
				// Matched on any key id rather than just the evicted one so
				// that an over-eager delete shows up as a failed assertion
				// below instead of an opaque "unexpected request" error.
				case req.Method == "GET" && strings.HasPrefix(req.URL.Path, keysPathFor("existing-user-uuid")+"/"):
					sawGetKey = true
					return etagResponse(`{"metadata":{},"data":{"extId":"old-key-ext-id"}}`), nil
				case req.Method == "DELETE" && strings.HasPrefix(req.URL.Path, keysPathFor("existing-user-uuid")+"/"):
					deletedKeys = append(deletedKeys, strings.TrimPrefix(req.URL.Path, keysPathFor("existing-user-uuid")+"/"))
					deleteKeyIfMatch = req.Header.Get("If-Match")
					return &http.Response{
						StatusCode: 204,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("existing-user-uuid"):
					sawCreateKey = true
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createKeyRespBodyExistingUser)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		resp, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.False(t, sawCreateUser, "create user endpoint should NOT have been called when user already exists")
		assert.True(t, sawListKeys, "list keys endpoint should have been called for existing user")
		assert.True(t, sawGetKey, "get key endpoint should have been called to fetch the ETag")
		assert.Equal(t, []string{"old-key-ext-id"}, deletedKeys,
			"exactly one key must be evicted to free a slot at the cap; the rest must be left alone")
		assert.Equal(t, mockEtag, deleteKeyIfMatch, "delete key must echo the fetched ETag back as If-Match")
		assert.True(t, sawCreateKey, "create key endpoint should have been called")
		assert.Equal(t, "existing-user-uuid", resp.UserID)
		assert.Equal(t, "new-access-key", resp.AccessKeyID)
		assert.Equal(t, "new-secret-key", resp.SecretAccessKey)
	})

	t.Run("TestCreateUser_ExistingUserNoKeysSkipsDelete", func(t *testing.T) {
		var sawDeleteKey, sawCreateKey bool
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersWithTestUserBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("existing-user-uuid"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysEmptyBody)),
					}, nil
				case req.Method == "DELETE" && strings.HasPrefix(req.URL.Path, keysPathFor("existing-user-uuid")+"/"):
					sawDeleteKey = true
					return &http.Response{
						StatusCode: 204,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("existing-user-uuid"):
					sawCreateKey = true
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createKeyRespBodyExistingUser)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		resp, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.False(t, sawDeleteKey, "delete key endpoint should NOT be called when the user has no keys")
		assert.True(t, sawCreateKey, "create key endpoint should have been called")
		assert.Equal(t, "existing-user-uuid", resp.UserID)
		assert.Equal(t, "new-access-key", resp.AccessKeyID)
	})

	// Below the cap there is room for another key, so no existing key is
	// touched. This is the common case for a user that has been granted
	// access a handful of times.
	t.Run("TestCreateUser_ExistingUserBelowKeyCapSkipsDelete", func(t *testing.T) {
		var sawDeleteKey, sawCreateKey bool
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersWithTestUserBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("existing-user-uuid"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysWithOneBody)),
					}, nil
				case req.Method == "DELETE" && strings.HasPrefix(req.URL.Path, keysPathFor("existing-user-uuid")+"/"):
					sawDeleteKey = true
					return &http.Response{
						StatusCode: 204,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("existing-user-uuid"):
					sawCreateKey = true
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createKeyRespBodyExistingUser)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		resp, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.False(t, sawDeleteKey, "no key should be evicted while the user is below the key cap")
		assert.True(t, sawCreateKey, "create key endpoint should have been called")
		assert.Equal(t, "existing-user-uuid", resp.UserID)
		assert.Equal(t, "new-access-key", resp.AccessKeyID)
	})

	t.Run("TestCreateUser_MissingUsername", func(t *testing.T) {
		api := &admin.API{}
		_, err := api.CreateUser(ctx, "" /* username */, mockDisplayName)
		assert.Contains(t, err.Error(), "username not set")
	})

	t.Run("TestCreateUser_CreateRequestError", func(t *testing.T) {
		api := baseApi
		api.PCEndpoint = "://"

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create http request")
	})

	t.Run("TestCreateUser_SendRequestError", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("failed to send http request")
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send http request")
	})

	t.Run("TestCreateUser_ListUsersNon200", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(bytes.NewBufferString(`"error":"internal server error"`)),
				}, nil
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-200 response")
	})

	t.Run("TestCreateUser_ListUsersFails", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == "GET" && req.URL.Path == listUsersPath {
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(bytes.NewBufferString(`{"error":"boom"}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list users")
	})

	t.Run("TestCreateUser_CreateUserNon201", func(t *testing.T) {
		errEnvelope := `{
			"metadata": {
				"messages": [
					{"message": "username is invalid", "severity": "ERROR", "code": "BAD_REQUEST"}
				]
			}
		}`
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersEmptyBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == createUserPath:
					return &http.Response{
						StatusCode: 400,
						Body:       io.NopCloser(bytes.NewBufferString(errEnvelope)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-201 response")
		assert.Contains(t, err.Error(), "username is invalid")
	})

	t.Run("TestCreateUser_CreateKeyFails", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersWithTestUserBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("existing-user-uuid"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysEmptyBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("existing-user-uuid"):
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(bytes.NewBufferString(`{"error":"server error"}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-201 response")
	})

	t.Run("TestCreateUser_ListKeysFails", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersWithTestUserBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("existing-user-uuid"):
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(bytes.NewBufferString(`{"metadata":{"messages":[{"message":"keys lookup failed"}]}}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-200 response")
		assert.Contains(t, err.Error(), "keys lookup failed")
	})

	// A failed eviction at the cap must abort CreateUser: there is no slot
	// for the new key, so carrying on would only fail more confusingly at
	// the create-key call.
	t.Run("TestCreateUser_DeleteKeyFails", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersWithTestUserBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("existing-user-uuid"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysAtCapBody)),
					}, nil
				case req.Method == "GET" && req.URL.Path == keyDeletePathFor("existing-user-uuid", "old-key-ext-id"):
					return etagResponse(`{"metadata":{},"data":{"extId":"old-key-ext-id"}}`), nil
				case req.Method == "DELETE" && req.URL.Path == keyDeletePathFor("existing-user-uuid", "old-key-ext-id"):
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(bytes.NewBufferString(`{"metadata":{"messages":[{"message":"key delete failed"}]}}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-204 response")
		assert.Contains(t, err.Error(), "key delete failed")
	})

	t.Run("TestCreateUser_UnmarshalError", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString("{bad json")),
				}, nil
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal")
	})

	t.Run("TestCreateUser_SendsAPIKeyHeader", func(t *testing.T) {
		// Service Account auth path: when the API has PCAPIKey set,
		// every request to the IAM proxy must carry the
		// X-ntnx-api-key header and must NOT fall back to Basic Auth.
		var capturedHeader, capturedAuth string
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				if capturedHeader == "" {
					capturedHeader = req.Header.Get("X-ntnx-api-key")
					capturedAuth = req.Header.Get("Authorization")
				}
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersEmptyBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == createUserPath:
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createUserRespBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("user-uuid"):
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createKeyRespBody)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := admin.API{
			PCEndpoint: "https://pc.example.com",
			PCAPIKey:   "test-key",
			HTTPClient: mockClient,
		}

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.Equal(t, "test-key", capturedHeader)
		assert.Empty(t, capturedAuth)
	})

	t.Run("TestCreateUser_SendsBasicAuthWhenNoAPIKey", func(t *testing.T) {
		// Regression: with username/password only the legacy Basic
		// Auth header must still be set, and the API key header must
		// be absent.
		var capturedHeader, capturedAuth string
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				if capturedAuth == "" {
					capturedHeader = req.Header.Get("X-ntnx-api-key")
					capturedAuth = req.Header.Get("Authorization")
				}
				switch {
				case req.Method == "GET" && req.URL.Path == listUsersPath:
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listUsersEmptyBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == createUserPath:
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createUserRespBody)),
					}, nil
				case req.Method == "POST" && req.URL.Path == keysPathFor("user-uuid"):
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(bytes.NewBufferString(createKeyRespBody)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.Empty(t, capturedHeader)
		assert.True(t, strings.HasPrefix(capturedAuth, "Basic "), "expected Basic Auth header, got %q", capturedAuth)
	})
}

func TestRemoveUser(t *testing.T) {
	ctx := context.Background()

	// listKeysTwoBody covers the multi-key branch: the v4 delete-user
	// contract implicitly wants the user cleaned out first, and the
	// caller-facing docs state every key must be removed when more than
	// one is on file.
	const listKeysTwoBody = `{
		"metadata": {},
		"data": [
			{"extId": "key-a", "name": "u", "keyType": "OBJECT_KEY", "createdTime": "2024-12-01T00:00:00Z"},
			{"extId": "key-b", "name": "u", "keyType": "OBJECT_KEY", "createdTime": "2024-12-02T00:00:00Z"}
		]
	}`

	t.Run("TestRemoveUser_MissingUUID", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{}

		err := api.RemoveUser(ctx, "" /* uuid */)
		assert.Contains(t, err.Error(), "user UUID not set")
	})

	t.Run("TestRemoveUser_CreateRequestError", func(t *testing.T) {
		api := baseApi
		api.PCEndpoint = "://"

		err := api.RemoveUser(ctx, "some-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create http request")
	})

	t.Run("TestRemoveUser_SendRequestError", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("http client error")
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.Contains(t, err.Error(), "failed to send http request")
	})

	// A 404 on the initial GET-by-id short-circuits RemoveUser and
	// returns nil so callers can call it unconditionally.
	t.Run("TestRemoveUser_UserAlreadyGone", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == "GET" && req.URL.Path == userPathFor("some-id") {
					return &http.Response{
						StatusCode: 404,
						Body:       io.NopCloser(bytes.NewBufferString(`{"metadata":{"messages":[{"message":"Requested user does not exist."}]}}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.NoError(t, err)
	})

	// 404 on the DELETE (raced with an out-of-band delete after the GET
	// succeeded) is also tolerated.
	t.Run("TestRemoveUser_DeleteReturns404", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == userPathFor("some-id"):
					return etagResponse(`{"metadata":{},"data":{"extId":"some-id"}}`), nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("some-id"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysEmptyBody)),
					}, nil
				case req.Method == "DELETE" && req.URL.Path == userPathFor("some-id"):
					return &http.Response{
						StatusCode: 404,
						Body:       io.NopCloser(bytes.NewBufferString(`{"metadata":{"messages":[{"message":"Requested user does not exist."}]}}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.NoError(t, err)
	})

	t.Run("TestRemoveUser_Non204Response", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == userPathFor("some-id"):
					return etagResponse(`{"metadata":{},"data":{"extId":"some-id"}}`), nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("some-id"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysEmptyBody)),
					}, nil
				case req.Method == "DELETE" && req.URL.Path == userPathFor("some-id"):
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(bytes.NewBufferString(`{"metadata":{"messages":[{"message":"internal server error"}]}}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-204 response")
		assert.Contains(t, err.Error(), "internal server error")
	})

	// Happy path: user has zero keys. Verify RemoveUser echoes the ETag
	// captured from the GET back as If-Match on the DELETE.
	t.Run("TestRemoveUser_Success", func(t *testing.T) {
		var sawGetUser, sawListKeys, sawDeleteUser bool
		var deleteIfMatch string
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == userPathFor("some-id"):
					sawGetUser = true
					return etagResponse(`{"metadata":{},"data":{"extId":"some-id"}}`), nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("some-id"):
					sawListKeys = true
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysEmptyBody)),
					}, nil
				case req.Method == "DELETE" && req.URL.Path == userPathFor("some-id"):
					sawDeleteUser = true
					deleteIfMatch = req.Header.Get("If-Match")
					return &http.Response{
						StatusCode: 204,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.NoError(t, err)
		assert.True(t, sawGetUser, "GET user should be issued to fetch the ETag")
		assert.True(t, sawListKeys, "list keys should be issued to enumerate keys before deleting the user")
		assert.True(t, sawDeleteUser, "DELETE user should be issued")
		assert.Equal(t, mockEtag, deleteIfMatch, "DELETE user must echo the fetched ETag back as If-Match")
	})

	// User has two OBJECT_KEYs; both must be deleted with their own
	// If-Match hop before the user itself is removed.
	t.Run("TestRemoveUser_DeletesAllKeys", func(t *testing.T) {
		deletedKeys := map[string]string{}
		var sawDeleteUser bool
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == "GET" && req.URL.Path == userPathFor("multi-key-user"):
					return etagResponse(`{"metadata":{},"data":{"extId":"multi-key-user"}}`), nil
				case req.Method == "GET" && req.URL.Path == keysPathFor("multi-key-user"):
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(listKeysTwoBody)),
					}, nil
				case req.Method == "GET" && strings.HasPrefix(req.URL.Path, keysPathFor("multi-key-user")+"/"):
					return etagResponse(`{"metadata":{},"data":{}}`), nil
				case req.Method == "DELETE" && strings.HasPrefix(req.URL.Path, keysPathFor("multi-key-user")+"/"):
					keyID := strings.TrimPrefix(req.URL.Path, keysPathFor("multi-key-user")+"/")
					deletedKeys[keyID] = req.Header.Get("If-Match")
					return &http.Response{
						StatusCode: 204,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				case req.Method == "DELETE" && req.URL.Path == userPathFor("multi-key-user"):
					sawDeleteUser = true
					return &http.Response{
						StatusCode: 204,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}
		err := api.RemoveUser(ctx, "multi-key-user")
		assert.NoError(t, err)
		assert.Equal(t, mockEtag, deletedKeys["key-a"], "key-a must be deleted with a fetched If-Match")
		assert.Equal(t, mockEtag, deletedKeys["key-b"], "key-b must be deleted with a fetched If-Match")
		assert.Len(t, deletedKeys, 2, "every OBJECT_KEY on the user must be deleted")
		assert.True(t, sawDeleteUser, "DELETE user should still run after all keys are deleted")
	})

	// If the ETag response is missing the header AND the $reserved
	// fallback, RemoveUser surfaces a clear error rather than sending an
	// empty If-Match (which the server would reject with 428).
	t.Run("TestRemoveUser_MissingEtag", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == "GET" && req.URL.Path == userPathFor("some-id") {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(`{"metadata":{},"data":{"extId":"some-id"}}`)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing Etag")
	})
}
