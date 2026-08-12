/*
Copyright 2022 Nutanix Inc.

Licensed under the Apache License, Version 2.0 (the "License");
You may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package s3client

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/s3"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
)

type Action string

// ActionList is a custom type that can unmarshal from both a string and an array
// S3 policies can have Action as either "s3:GetObject" or ["s3:GetObject", "s3:PutObject"]
type ActionList []Action

func (a *ActionList) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as an array first
	var actions []Action
	if err := json.Unmarshal(data, &actions); err == nil {
		*a = actions
		return nil
	}

	// If that fails, try as a single string
	var single Action
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*a = []Action{single}
	return nil
}

func (a ActionList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]Action(a))
}

// StringOrArray handles S3 policy fields that can be string or []string
// Used for Resource field which can be "arn:..." or ["arn:...", "arn:..."]
type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*s = []string{single}
	return nil
}

func (s StringOrArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(s))
}

// Principal handles AWS Principal which can be:
// - "*" (string for public access)
// - {"AWS": "arn:..."} (single principal)
// - {"AWS": ["arn:...", "arn:..."]} (multiple principals)
type Principal map[string]StringOrArray

func (p *Principal) UnmarshalJSON(data []byte) error {
	// Try as a string first (e.g., "*")
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*p = Principal{"AWS": StringOrArray{str}}
		return nil
	}

	// Try as map with string values
	var mapStr map[string]string
	if err := json.Unmarshal(data, &mapStr); err == nil {
		result := make(Principal)
		for k, v := range mapStr {
			result[k] = StringOrArray{v}
		}
		*p = result
		return nil
	}

	// Try as map with array values
	var mapArr map[string][]string
	if err := json.Unmarshal(data, &mapArr); err == nil {
		result := make(Principal)
		for k, v := range mapArr {
			result[k] = StringOrArray(v)
		}
		*p = result
		return nil
	}

	return fmt.Errorf("cannot unmarshal Principal from: %s", string(data))
}

func (p Principal) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]StringOrArray(p))
}

const (
	All                            Action = "s3:*"
	AbortMultipartUpload           Action = "s3:AbortMultipartUpload"
	CreateBucket                   Action = "s3:CreateBucket"
	DeleteBucketPolicy             Action = "s3:DeleteBucketPolicy"
	DeleteBucket                   Action = "s3:DeleteBucket"
	DeleteBucketWebsite            Action = "s3:DeleteBucketWebsite"
	DeleteObject                   Action = "s3:DeleteObject"
	DeleteObjectVersion            Action = "s3:DeleteObjectVersion"
	DeleteReplicationConfiguration Action = "s3:DeleteReplicationConfiguration"
	GetAccelerateConfiguration     Action = "s3:GetAccelerateConfiguration"
	GetBucketAcl                   Action = "s3:GetBucketAcl"
	GetBucketCORS                  Action = "s3:GetBucketCORS"
	GetBucketLocation              Action = "s3:GetBucketLocation"
	GetBucketLogging               Action = "s3:GetBucketLogging"
	GetBucketNotification          Action = "s3:GetBucketNotification"
	GetBucketPolicy                Action = "s3:GetBucketPolicy"
	GetBucketRequestPayment        Action = "s3:GetBucketRequestPayment"
	GetBucketTagging               Action = "s3:GetBucketTagging"
	GetBucketVersioning            Action = "s3:GetBucketVersioning"
	GetBucketWebsite               Action = "s3:GetBucketWebsite"
	GetLifecycleConfiguration      Action = "s3:GetLifecycleConfiguration"
	GetObjectAcl                   Action = "s3:GetObjectAcl"
	GetObject                      Action = "s3:GetObject"
	GetObjectTorrent               Action = "s3:GetObjectTorrent"
	GetObjectVersionAcl            Action = "s3:GetObjectVersionAcl"
	GetObjectVersion               Action = "s3:GetObjectVersion"
	GetObjectVersionTorrent        Action = "s3:GetObjectVersionTorrent"
	GetReplicationConfiguration    Action = "s3:GetReplicationConfiguration"
	ListAllMyBuckets               Action = "s3:ListAllMyBuckets"
	ListBucketMultipartUploads     Action = "s3:ListBucketMultipartUploads"
	ListBucket                     Action = "s3:ListBucket"
	ListBucketVersions             Action = "s3:ListBucketVersions"
	ListMultipartUploadParts       Action = "s3:ListMultipartUploadParts"
	PutAccelerateConfiguration     Action = "s3:PutAccelerateConfiguration"
	PutBucketAcl                   Action = "s3:PutBucketAcl"
	PutBucketCORS                  Action = "s3:PutBucketCORS"
	PutBucketLogging               Action = "s3:PutBucketLogging"
	PutBucketNotification          Action = "s3:PutBucketNotification"
	PutBucketPolicy                Action = "s3:PutBucketPolicy"
	PutBucketRequestPayment        Action = "s3:PutBucketRequestPayment"
	PutBucketTagging               Action = "s3:PutBucketTagging"
	PutBucketVersioning            Action = "s3:PutBucketVersioning"
	PutBucketWebsite               Action = "s3:PutBucketWebsite"
	PutLifecycleConfiguration      Action = "s3:PutLifecycleConfiguration"
	PutObjectAcl                   Action = "s3:PutObjectAcl"
	PutObject                      Action = "s3:PutObject"
	PutObjectVersionAcl            Action = "s3:PutObjectVersionAcl"
	PutReplicationConfiguration    Action = "s3:PutReplicationConfiguration"
	RestoreObject                  Action = "s3:RestoreObject"
)

var AllowedActions = []Action{
	AbortMultipartUpload,
	DeleteObject,
	GetBucketLocation,
	GetObject,
	ListBucket,
	ListBucketMultipartUploads,
	ListMultipartUploadParts,
	PutObject,
	PutLifecycleConfiguration,
}

type Effect string

// effectAllow values are expected by the S3 API to be 'Allow' explicitly
const (
	effectAllow Effect = "Allow"
)

// PolicyStatment is the Go representation of a PolicyStatement json struct
// it defines what Actions that a Principle can or cannot perform on a Resource
type PolicyStatement struct {
	// Sid (optional) is the PolicyStatement's unique identifier
	Sid string `json:"Sid"`
	// Effect determines whether the Action(s) are 'Allow'ed
	Effect Effect `json:"Effect"`
	// Principle is/are the nutanix user names affected by this PolicyStatement
	// Must be in the format of '<username>'
	Principal Principal `json:"Principal"`
	// Action is a list of s3:* actions
	Action ActionList `json:"Action"`
	// Resource is the ARN identifier for the S3 resource (bucket)
	// Must be in the format of 'arn:aws:s3:::<bucket>'
	Resource StringOrArray `json:"Resource"`
}

// BucketPolicy represents set of policy statements for a single bucket.
type BucketPolicy struct {
	// Id (optional) identifies the bucket policy
	Id string `json:"Id"`
	// Version is the version of the BucketPolicy data structure
	// should always be '2012-10-17'
	Version   string            `json:"Version"`
	Statement []PolicyStatement `json:"Statement"`
}

// the version of the BucketPolicy json structure
const version = "2012-10-17"

// NewBucketPolicy obviously returns a new BucketPolicy.  PolicyStatements may be passed in at creation
// or added after the fact.  BucketPolicies should be passed to PutBucketPolicy().
func NewBucketPolicy(ps ...PolicyStatement) *BucketPolicy {
	bp := &BucketPolicy{
		Version:   version,
		Statement: append([]PolicyStatement{}, ps...),
	}
	return bp
}

// PutBucketPolicy applies the policy to the bucket
func (s *S3Agent) PutBucketPolicy(bucket string, policy BucketPolicy) (*s3.PutBucketPolicyOutput, error) {

	confirmRemoveSelfBucketAccess := false
	serializedPolicy, _ := k8sjson.Marshal(policy)
	consumablePolicy := string(serializedPolicy)

	p := &s3.PutBucketPolicyInput{
		Bucket:                        &bucket,
		ConfirmRemoveSelfBucketAccess: &confirmRemoveSelfBucketAccess,
		Policy:                        &consumablePolicy,
	}
	out, err := s.Client.PutBucketPolicy(p)
	if err != nil {
		return out, err
	}
	return out, nil
}

// DeleteBucketPolicy removes the policy from the bucket entirely.
// S3 does not accept a PutBucketPolicy call with zero statements, so this
// must be used when the caller wants to clear the policy (for example,
// after revoking the last principal).
func (s *S3Agent) DeleteBucketPolicy(bucket string) (*s3.DeleteBucketPolicyOutput, error) {
	return s.Client.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
		Bucket: &bucket,
	})
}

func (s *S3Agent) GetBucketPolicy(bucket string) (*BucketPolicy, error) {
	out, err := s.Client.GetBucketPolicy(&s3.GetBucketPolicyInput{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, err
	}

	policy := &BucketPolicy{}
	err = k8sjson.Unmarshal([]byte(*out.Policy), policy)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// ModifyBucketPolicy new and old statement SIDs and overwrites on a match.
// This allows users to Get, modify, and Replace existing statements as well as
// add new ones.
func (bp *BucketPolicy) ModifyBucketPolicy(ps ...PolicyStatement) *BucketPolicy {
	for _, newP := range ps {
		var match bool
		for j, oldP := range bp.Statement {
			if newP.Sid == oldP.Sid {
				bp.Statement[j] = newP
				match = true
			}
		}
		if !match {
			bp.Statement = append(bp.Statement, newP)
		}
	}
	return bp
}

func (bp *BucketPolicy) DropPolicyStatements(sid ...string) *BucketPolicy {
	for _, s := range sid {
		for i, stmt := range bp.Statement {
			if stmt.Sid == s {
				bp.Statement = append(bp.Statement[:i], bp.Statement[i+1:]...)
				break
			}
		}
	}
	return bp
}

func (bp *BucketPolicy) EjectPrincipals(users ...string) *BucketPolicy {
	statements := bp.Statement
	for _, s := range statements {
		s.EjectPrincipals(users...)
	}
	bp.Statement = statements
	return bp
}

// NewPolicyStatement generates a new PolicyStatement. PolicyStatment methods are designed to
// be chain called with dot notation to allow for easy configuration at creation.  This is preferable
// to a long parameter list.
func NewPolicyStatement() *PolicyStatement {
	return &PolicyStatement{
		Sid:       "",
		Effect:    "",
		Principal: Principal{},
		Action:    ActionList{},
		Resource:  StringOrArray{},
	}
}

func (ps *PolicyStatement) WithSID(sid string) *PolicyStatement {
	ps.Sid = sid
	return ps
}

const awsPrinciple = "AWS"
const arnPrefixResource = "arn:aws:s3:::%s"

// ForPrincipals adds users to the PolicyStatement
func (ps *PolicyStatement) ForPrincipals(users ...string) *PolicyStatement {
	principals := []string(ps.Principal[awsPrinciple])
	for _, u := range users {
		principals = append(principals, u)
	}
	ps.Principal[awsPrinciple] = StringOrArray(principals)
	return ps
}

// ForResources adds resources (buckets) to the PolicyStatement with the appropriate ARN prefix
func (ps *PolicyStatement) ForResources(resources ...string) *PolicyStatement {
	for _, v := range resources {
		ps.Resource = append(ps.Resource, fmt.Sprintf(arnPrefixResource, v))
	}
	return ps
}

// ForSubResources add contents inside the bucket to the PolicyStatement with the appropriate ARN prefix
func (ps *PolicyStatement) ForSubResources(resources ...string) *PolicyStatement {
	var subresource string
	for _, v := range resources {
		subresource = fmt.Sprintf("%s/*", v)
		ps.Resource = append(ps.Resource, fmt.Sprintf(arnPrefixResource, subresource))
	}
	return ps
}

// Allows sets the effect of the PolicyStatement to allow PolicyStatement's Actions
func (ps *PolicyStatement) Allows() *PolicyStatement {
	if ps.Effect != "" {
		return ps
	}
	ps.Effect = effectAllow
	return ps
}

// Actions is the set of "s3:*" actions for the PolicyStatement is concerned
func (ps *PolicyStatement) Actions(actions ...Action) *PolicyStatement {
	ps.Action = actions
	return ps
}

func (ps *PolicyStatement) EjectPrincipals(users ...string) {
	principals := []string(ps.Principal[awsPrinciple])
	for _, u := range users {
		for j, v := range principals {
			if u == v {
				principals = append(principals[:j], principals[j+1:]...)
			}
		}
	}
	ps.Principal[awsPrinciple] = StringOrArray(principals)
}
