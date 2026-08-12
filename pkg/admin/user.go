package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	marshalError   = "failed to marshal ntnx http request body"
	unmarshalError = "failed to unmarshal ntnx http response"
	userV4Api      = "/api/iam/v4.0/authn/users"
	userKeyV4Api   = "/api/iam/v4.0/authn/users/%s/keys"

	ifMatchHeader = "If-Match"
	etagHeader    = "Etag"

	maxObjectKeysPerUser = 5
)

var (
	errMissingUsername = errors.New("username not set")
	errMissingUserID   = errors.New("user UUID not set")
)

// UserCredentials is the result returned by CreateUser. It carries
// only the fields the provisioner actually needs: the IAM user's extId
// and the freshly minted OBJECT_KEY access/secret pair.
type UserCredentials struct {
	UserID          string
	AccessKeyID     string
	SecretAccessKey string
	CreatedTime     time.Time
}

// v4Metadata is the envelope used by v4 IAM responses to surface
// server-side messages. Only the fields we actually read are listed.
type Metadata struct {
	Messages []struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Code     string `json:"code"`
	} `json:"messages"`
}

type CreateUserReq struct {
	Username    string `json:"username"`
	UserType    string `json:"userType"`
	DisplayName string `json:"displayName"`
}

type UserData struct {
	ExtID       string `json:"extId"`
	Username    string `json:"username"`
	UserType    string `json:"userType"`
	DisplayName string `json:"displayName"`
}

type UserResp struct {
	Metadata Metadata `json:"metadata"`
	Data     UserData `json:"data"`
}

type UserListResp struct {
	Metadata Metadata   `json:"metadata"`
	Data     []UserData `json:"data"`
}

type KeyDetails struct {
	ObjectType string `json:"$objectType"`
	AccessKey  string `json:"accessKey,omitempty"`
	SecretKey  string `json:"secretKey,omitempty"`
}

// CreateKeyReq is the v4 IAM POST /users/{extId}/keys request body.
// For OBJECT_KEY only name and keyType are sent; keyDetails is a
// response-only field populated by the server with the newly minted
// access/secret pair.
type CreateKeyReq struct {
	Name    string `json:"name"`
	KeyType string `json:"keyType"`
}

type KeyData struct {
	ExtID       string     `json:"extId"`
	Name        string     `json:"name"`
	KeyType     string     `json:"keyType"`
	CreatedTime time.Time  `json:"createdTime"`
	KeyDetails  KeyDetails `json:"keyDetails"`
}

type KeyResp struct {
	Metadata Metadata `json:"metadata"`
	Data     KeyData  `json:"data"`
}

type KeyListResp struct {
	Metadata Metadata  `json:"metadata"`
	Data     []KeyData `json:"data"`
}

// fetchETag issues a GET against the given URL and returns the value of
// the response's ETag header. v4 IAM DELETE endpoints (user and key)
// reject the request with 428 IAM-20007 unless the caller echoes the
// current ETag back as an If-Match header, so every DELETE flow has to
// pair with a preceding GET.
func (api *API) fetchETag(ctx context.Context, url string) (string, bool, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return "", false, fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("%w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("non-200 response: %d - %s", resp.StatusCode, decodeError(body))
	}

	if etag := resp.Header.Get(etagHeader); etag != "" {
		return etag, true, nil
	}
	envelope := struct {
		Reserved struct {
			Etag string `json:"Etag"`
		} `json:"$reserved"`
	}{}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Reserved.Etag != "" {
		return envelope.Reserved.Etag, true, nil
	}
	return "", false, fmt.Errorf("response missing Etag header and $reserved.Etag: %s", string(body))
}

// decodeError joins the messages from a v4 IAM error envelope so they
// can be surfaced in the wrapped error returned to callers. It is
// best-effort: if the body isn't a parseable v4 envelope we fall back to
// the raw response text.
func decodeError(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	envelope := struct {
		Metadata Metadata `json:"metadata"`
	}{}
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Metadata.Messages) == 0 {
		return string(body)
	}
	msgs := make([]string, 0, len(envelope.Metadata.Messages))
	for _, m := range envelope.Metadata.Messages {
		if m.Message != "" {
			msgs = append(msgs, m.Message)
		}
	}
	if len(msgs) == 0 {
		return string(body)
	}
	return strings.Join(msgs, "; ")
}

// findUserByUsername looks up an existing IAM user via the list
// endpoint. It returns the user's extId and whether a match was found.
func (api *API) findUserByUsername(ctx context.Context, username string) (string, bool, error) {
	filter := fmt.Sprintf("username eq '%s'", username)
	params := url.Values{}
	params.Add("$filter", filter)
	userURL := api.PCEndpoint + userV4Api + "?" + params.Encode()

	request, err := http.NewRequestWithContext(ctx, "GET", userURL, nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return "", false, fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("%w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("non-200 response: %d - %s", resp.StatusCode, decodeError(body))
	}

	list := UserListResp{}
	if err := json.Unmarshal(body, &list); err != nil {
		return "", false, fmt.Errorf("%s. %s. %w", unmarshalError, string(body), err)
	}

	for _, u := range list.Data {
		if u.Username == username {
			return u.ExtID, true, nil
		}
	}
	return "", false, nil
}

// CreateUser provisions an IAM user for an external (S3) consumer and
// mints a fresh OBJECT_KEY for them. The list endpoint is consulted
// first so that an existing user is reused rather than re-created.
//
// A new key is always minted, because the v4 API only returns the
// access/secret pair at creation time and there is no way to recover it
// afterwards. Existing keys are left in place unless the user has reached
// maxObjectKeysPerUser, in which case one is evicted first to free a slot
// for the new key.
func (api *API) CreateUser(ctx context.Context, username, displayName string) (UserCredentials, error) {
	if username == "" {
		return UserCredentials{}, errMissingUsername
	}

	extID, found, err := api.findUserByUsername(ctx, username)
	if err != nil {
		return UserCredentials{}, fmt.Errorf("failed to list users. %w", err)
	}

	if !found {
		extID, err = api.createIAMUser(ctx, username, displayName)
		if err != nil {
			return UserCredentials{}, err
		}
	} else {
		existingKeys, err := api.listAccessKeys(ctx, extID)
		if err != nil {
			return UserCredentials{}, err
		}
		if len(existingKeys) == maxObjectKeysPerUser {
			if err := api.deleteAccessKey(ctx, extID, existingKeys[0].ExtID); err != nil {
				return UserCredentials{}, err
			}
		}
	}

	key, err := api.createAccessKey(ctx, extID, username)
	if err != nil {
		return UserCredentials{}, err
	}

	if key.Data.KeyDetails.AccessKey == "" || key.Data.KeyDetails.SecretKey == "" {
		return UserCredentials{}, fmt.Errorf("create key response for user %s carried no OBJECT_KEY credentials (keyType=%q)", extID, key.Data.KeyType)
	}

	return UserCredentials{
		UserID:          extID,
		AccessKeyID:     key.Data.KeyDetails.AccessKey,
		SecretAccessKey: key.Data.KeyDetails.SecretKey,
		CreatedTime:     key.Data.CreatedTime,
	}, nil
}

func (api *API) createIAMUser(ctx context.Context, username, displayName string) (string, error) {
	body := CreateUserReq{
		Username:    username,
		UserType:    "EXTERNAL",
		DisplayName: displayName,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("%s. %w", marshalError, err)
	}

	url := api.PCEndpoint + userV4Api
	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	request.Header.Add("Content-Type", "application/json")
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("non-201 response: %d - %s", resp.StatusCode, decodeError(respBody))
	}

	created := UserResp{}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("%s. %s. %w", unmarshalError, string(respBody), err)
	}
	if created.Data.ExtID == "" {
		return "", fmt.Errorf("create user response missing extId: %s", string(respBody))
	}
	return created.Data.ExtID, nil
}

func (api *API) createAccessKey(ctx context.Context, extID, username string) (KeyResp, error) {
	result := KeyResp{}

	body := CreateKeyReq{
		Name:    username,
		KeyType: "OBJECT_KEY",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return result, fmt.Errorf("%s. %w", marshalError, err)
	}

	url := api.PCEndpoint + fmt.Sprintf(userKeyV4Api, extID)
	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return result, fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	request.Header.Add("Content-Type", "application/json")
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("%w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return result, fmt.Errorf("non-201 response: %d - %s", resp.StatusCode, decodeError(respBody))
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return result, fmt.Errorf("%s. %s. %w", unmarshalError, string(respBody), err)
	}
	return result, nil
}

// listAccessKeys returns all OBJECT_KEYs currently bound to the given
// user. We use it to discover the extId of the existing key that needs
// to be rotated before issuing a fresh access/secret pair.
func (api *API) listAccessKeys(ctx context.Context, extID string) ([]KeyData, error) {
	url := api.PCEndpoint + fmt.Sprintf(userKeyV4Api, extID)
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response: %d - %s", resp.StatusCode, decodeError(body))
	}

	list := KeyListResp{}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("%s. %s. %w", unmarshalError, string(body), err)
	}
	return list.Data, nil
}

// deleteAccessKey removes a single OBJECT_KEY from a user via
// /api/iam/v4.0/authn/users/{extID}/keys/{keyExtID}.
func (api *API) deleteAccessKey(ctx context.Context, extID, keyExtID string) error {
	keyURL := api.PCEndpoint + fmt.Sprintf(userKeyV4Api, extID) + "/" + keyExtID

	etag, found, err := api.fetchETag(ctx, keyURL)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	request, err := http.NewRequestWithContext(ctx, "DELETE", keyURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	request.Header.Set(ifMatchHeader, etag)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("non-204 response: %d - %s", resp.StatusCode, decodeError(body))
	}
	return nil
}

// RemoveUser deletes an IAM user by extId via the v4.0 API.
//
// Every OBJECT_KEY still bound to the user is deleted up front. The v4
// key-delete API has the same If-Match requirement, so we rely on
// deleteAccessKey to fetch each key's ETag internally. All keys are
// deleted, not just the first one, so users that somehow accumulated
// more than one access key still get fully torn down.
func (api *API) RemoveUser(ctx context.Context, uuid string) error {
	if uuid == "" {
		return errMissingUserID
	}

	userURL := api.PCEndpoint + userV4Api + "/" + uuid

	etag, found, err := api.fetchETag(ctx, userURL)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	keys, err := api.listAccessKeys(ctx, uuid)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := api.deleteAccessKey(ctx, uuid, k.ExtID); err != nil {
			return err
		}
	}

	request, err := http.NewRequestWithContext(ctx, "DELETE", userURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request. %w", err)
	}

	api.Authenticate(request)
	request.Header.Set(ifMatchHeader, etag)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("non-204 response: %d - %s", resp.StatusCode, decodeError(body))
	}
	return nil
}
