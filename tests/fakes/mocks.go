package mocks

import (
	"context"
	"net/http"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/s3client"

	"github.com/aws/aws-sdk-go/service/s3"
)

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m MockHTTPClient) Do(req *http.Request) (*http.Response, error) { return m.DoFunc(req) }

type MockIAM struct {
	CreateUserFunc     func(ctx context.Context, username string, display_name string) (admin.NutanixUserResp, error)
	RemoveUserFunc     func(ctx context.Context, uuid string) error
	GetAccountNameFunc func() string
	GetEndpointFunc    func() string
}

func (m MockIAM) CreateUser(ctx context.Context, username string, display_name string) (admin.NutanixUserResp, error) {
	return m.CreateUserFunc(ctx, username, display_name)
}
func (m MockIAM) RemoveUser(ctx context.Context, uuid string) error {
	return m.RemoveUserFunc(ctx, uuid)
}
func (m MockIAM) GetAccountName() string { return m.GetAccountNameFunc() }
func (m MockIAM) GetEndpoint() string    { return m.GetEndpointFunc() }

type MockS3Client struct {
	CreateBucketFunc    func(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
	DeleteBucketFunc    func(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error)
	DeleteObjectFunc    func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	PutObjectFunc       func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObjectFunc       func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	GetBucketPolicyFunc func(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error)
	PutBucketPolicyFunc func(input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error)
}

func (m *MockS3Client) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return m.CreateBucketFunc(input)
}
func (m *MockS3Client) DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return m.DeleteBucketFunc(input)
}
func (m *MockS3Client) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return m.DeleteObjectFunc(input)
}
func (m *MockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return m.PutObjectFunc(input)
}
func (m *MockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.GetObjectFunc(input)
}
func (m *MockS3Client) GetBucketPolicy(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
	return m.GetBucketPolicyFunc(input)
}
func (m *MockS3Client) PutBucketPolicy(input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
	return m.PutBucketPolicyFunc(input)
}

type MockProvisionerS3 struct {
	CreateBucketFunc    func(name string) error
	DeleteBucketFunc    func(name string) (bool, error)
	GetBucketPolicyFunc func(bucket string) (*s3client.BucketPolicy, error)
	PutBucketPolicyFunc func(bucket string, policy s3client.BucketPolicy) (*s3.PutBucketPolicyOutput, error)
}

func (m MockProvisionerS3) CreateBucket(name string) error         { return m.CreateBucketFunc(name) }
func (m MockProvisionerS3) DeleteBucket(name string) (bool, error) { return m.DeleteBucketFunc(name) }
func (m MockProvisionerS3) GetBucketPolicy(bucket string) (*s3client.BucketPolicy, error) {
	return m.GetBucketPolicyFunc(bucket)
}
func (m MockProvisionerS3) PutBucketPolicy(bucket string, policy s3client.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
	return m.PutBucketPolicyFunc(bucket, policy)
}
