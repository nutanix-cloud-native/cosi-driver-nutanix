package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	marshalError   = "failed to marshal ntnx http request body"
	unmarshalError = "failed to unmarshal ntnx http response"
	createEndpoint = "/oss/iam_proxy/buckets_access_keys"
	deleteEndpoint = "/oss/iam_proxy/users/"
)

var (
	errMissingUsername = errors.New("username not set")
	errMissingUserID   = errors.New("user UUID not set")
)

type NtnxUserReq struct {
	Users []NtnxUserInfo `json:"users"`
}

type NtnxUserInfo struct {
	Type        string `json:"type"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type NutanixUserResp struct {
	Users []struct {
		BucketsAccessKeys []struct {
			AccessKeyID     string    `json:"access_key_id"`
			CreatedTime     time.Time `json:"created_time"`
			SecretAccessKey string    `json:"secret_access_key"`
		} `json:"buckets_access_keys"`
		CreatedTime     time.Time `json:"created_time"`
		DisplayName     string    `json:"display_name"`
		LastUpdatedTime time.Time `json:"last_updated_time"`
		TenantID        string    `json:"tenant_id"`
		Type            string    `json:"type"`
		Username        string    `json:"username"`
		UUID            string    `json:"uuid"`
	} `json:"users"`
}

type NutanixUserErrorResp struct {
	Users []struct {
		BucketsAccessKeys interface{} `json:"buckets_access_keys"`
		Code              int         `json:"code"`
		Message           string      `json:"message"`
		Type              string      `json:"type"`
		Username          string      `json:"username"`
	} `json:"users"`
}

// Nutanix IAM User
func (api *API) CreateUser(ctx context.Context, username, display_name string) (NutanixUserResp, error) {
	result := NutanixUserResp{}
	if username == "" {
		return result, errMissingUsername
	}

	// Create API
	url := api.PCEndpoint + createEndpoint

	// Request Body
	info := &NtnxUserReq{
		Users: []NtnxUserInfo{
			{
				Type:        "external",
				Username:    username,
				DisplayName: display_name,
			},
		},
	}

	// Converts data struct into json
	data, err := json.Marshal(info)
	if err != nil {
		return result, fmt.Errorf("%s. %w", marshalError, err)
	}

	// Send Request
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return result, fmt.Errorf("%w", err)
	}

	request.SetBasicAuth(api.PCUsername, api.PCPassword)
	request.Header.Add("Content-Type", "application/json")
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("%w", err)
	}
	defer resp.Body.Close()

	// Check respsonse status
	if resp.StatusCode != 200 {
		return result, fmt.Errorf("%s", resp.Status)
	}

	decodedResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("%w", err)
	}

	// Unmarshal response into Go type
	err = json.Unmarshal(decodedResponse, &result)
	if err != nil {
		return result, fmt.Errorf("%s. %s. %w", unmarshalError, string(decodedResponse), err)
	}

	// Unmarshal function doesn't return an error if attributes are different from the defined struct
	// len(result.Users[0].BucketsAccessKeys) equal to 0, implies that new user wasn't created
	if len(result.Users[0].BucketsAccessKeys) == 0 {
		// Using NutanixUserErrorResp struct to capture error code and error message
		errorResp := NutanixUserErrorResp{}
		err = json.Unmarshal(decodedResponse, &errorResp)
		if err != nil {
			return result, fmt.Errorf("%s. %s. %w", unmarshalError, string(decodedResponse), err)
		}

		return NutanixUserResp{}, fmt.Errorf("errorCode : %d, errorMessage : %s", errorResp.Users[0].Code, errorResp.Users[0].Message)
	}

	return result, nil
}

// RemoveUser removes an user from the object store
func (api *API) RemoveUser(ctx context.Context, uuid string) error {

	if uuid == "" {
		return errMissingUserID
	}

	// Delete API
	delete_url := api.PCEndpoint + deleteEndpoint + string(uuid)
	delete_request, err := http.NewRequest("DELETE", delete_url, nil)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	delete_request.SetBasicAuth(api.PCUsername, api.PCPassword)
	delete_resp, err := api.HTTPClient.Do(delete_request)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer delete_resp.Body.Close()

	// Check response status
	if delete_resp.StatusCode != 204 {
		return fmt.Errorf("%s", delete_resp.Status)
	}
	return nil
}
