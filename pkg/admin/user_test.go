package admin_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
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

func TestCreateUser(t *testing.T) {
	mockUsername := "testuser"
	mockDisplayName := "Test User"
	mockRespBody := `{
		"users": [{
			"username": "testuser",
			"display_name": "Test User",
			"type": "external",
			"created_time": "2025-01-01T00:00:00Z",
			"last_updated_time": "2025-01-01T00:00:00Z",
			"tenant_id": "tenant-id",
			"uuid": "user-uuid",
			"buckets_access_keys": [{
				"access_key_id": "access-key",
				"secret_access_key": "secret-key",
				"created_time": "2025-01-01T00:00:00Z"
			}]
		}]
	}`

	t.Run("TestCreateUser_Success", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(mockRespBody)),
				}, nil
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		resp, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.NoError(t, err)
		assert.Equal(t, mockUsername, resp.Users[0].Username)
		assert.Equal(t, "access-key", resp.Users[0].BucketsAccessKeys[0].AccessKeyID)
		assert.NotNil(t, len(resp.Users[0].BucketsAccessKeys))
	})

	t.Run("TestCreateUser_UserNotCreated", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				body := `{
					"users": [{
						"buckets_access_keys": []
					}]
				}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}, nil
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		resp, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Equal(t, admin.NutanixUserResp{}, resp)
		assert.Contains(t, err.Error(), "user not created")
	})

	t.Run("TestCreateUser_MissingUsername", func(t *testing.T) {
		api := &admin.API{}
		_, err := api.CreateUser(ctx, "", mockDisplayName)
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

	t.Run("TestCreateUser_Non200Response", func(t *testing.T) {
		mockClient := mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				errMsg := `"error":"internal server error"`
				return &http.Response{
					StatusCode: 500,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(bytes.NewBufferString(errMsg)),
				}, nil
			},
		}

		api := baseApi
		api.HTTPClient = mockClient

		_, err := api.CreateUser(ctx, mockUsername, mockDisplayName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-200 response")
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

}

func TestRemoveUser(t *testing.T) {
	ctx := context.Background()

	t.Run("TestRemoveUser_MissingUUID", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{}

		err := api.RemoveUser(ctx, "")
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

	t.Run("TestRemoveUser_404Response", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				body := `{
					"message": "Requested user does not exist.",
					"code": 404
				}`
				resp := &http.Response{
					StatusCode: 404,
					Status:     "404 Not Found",
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}
				return resp, nil
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.NoError(t, err)
	})

	t.Run("TestRemoveUser_Non204Response", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				body := `{
					"message": "Requested user does not exist.",
					"code": 500
				}`
				resp := &http.Response{
					StatusCode: 500,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}
				return resp, nil
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.Contains(t, err.Error(), "non-204 response")
	})

	t.Run("TestRemoveUser_Success", func(t *testing.T) {
		api := baseApi
		api.HTTPClient = mocks.MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 204,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}
				return resp, nil
			},
		}
		err := api.RemoveUser(ctx, "some-id")
		assert.NoError(t, err)
	})
}
