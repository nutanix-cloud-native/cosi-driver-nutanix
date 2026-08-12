package driver_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/driver"
	s3cli "github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/s3client"
	mocks "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/fakes"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

func TestDriverCreateBucket(t *testing.T) {
	t.Run("TestDriverCreateBucket_Success", func(t *testing.T) {
		mockS3 := mocks.MockProvisionerS3{
			CreateBucketFunc: func(name string) error {
				assert.Equal(t, "test-bucket", name)
				return nil
			},
		}
		server := &driver.ProvisionerServer{S3Client: mockS3}
		req := &cosi.DriverCreateBucketRequest{
			Name:       "test-bucket",
			Parameters: map[string]string{},
		}
		resp, err := server.DriverCreateBucket(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test-bucket", resp.BucketId)
	})

	t.Run("TestDriverCreateBucket_CreateBucketError", func(t *testing.T) {
		mockS3 := mocks.MockProvisionerS3{
			CreateBucketFunc: func(name string) error {
				return errors.New("s3 error")
			},
		}
		server := &driver.ProvisionerServer{S3Client: mockS3}
		req := &cosi.DriverCreateBucketRequest{
			Name:       "fail-bucket",
			Parameters: map[string]string{},
		}
		resp, err := server.DriverCreateBucket(context.Background(), req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to create bucket")
	})
}

func TestDriverDeleteBucket(t *testing.T) {
	t.Run("TestDriverDeleteBucket_Success", func(t *testing.T) {
		mockS3 := mocks.MockProvisionerS3{
			DeleteBucketFunc: func(name string) (bool, error) {
				assert.Equal(t, "test-bucket", name)
				return true, nil
			},
		}
		server := &driver.ProvisionerServer{S3Client: mockS3}
		req := &cosi.DriverDeleteBucketRequest{
			BucketId: "test-bucket",
		}
		resp, err := server.DriverDeleteBucket(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("TestDriverDeleteBucket_DeleteBucketError", func(t *testing.T) {
		mockS3 := &mocks.MockProvisionerS3{
			DeleteBucketFunc: func(name string) (bool, error) {
				return false, errors.New("backend delete error")
			},
		}
		server := &driver.ProvisionerServer{S3Client: mockS3}

		req := &cosi.DriverDeleteBucketRequest{BucketId: "bucket-fail"}
		resp, err := server.DriverDeleteBucket(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestDriverGrantBucketAccess(t *testing.T) {
	mockIAM := mocks.MockIAM{
		CreateUserFunc: func(ctx context.Context, username, displayName string) (admin.UserCredentials, error) {
			return admin.UserCredentials{
				UserID:          "test-uuid",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-key",
			}, nil
		},
		GetAccountNameFunc: func() string { return "cosi" },
		GetEndpointFunc:    func() string { return "https://pc-endpoint" },
	}

	t.Run("TestDriverGrantBucketAccess_NewPolicySuccess", func(t *testing.T) {
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return nil, awserr.New("NoSuchBucketPolicy", "no existing policy", nil /* origErr */)
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				return &s3.PutBucketPolicyOutput{}, nil
			},
		}

		server := &driver.ProvisionerServer{
			Provisioner:   "test",
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}

		req := &cosi.DriverGrantBucketAccessRequest{
			Name:               "ba-test",
			BucketId:           "bucket-test",
			AuthenticationType: cosi.AuthenticationType_Key,
		}

		resp, err := server.DriverGrantBucketAccess(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "test-uuid", resp.AccountId)
		assert.Equal(t, "test-key", resp.Credentials["s3"].Secrets["accessKeyID"])
		assert.Equal(t, "test-key", resp.Credentials["s3"].Secrets["accessSecretKey"])
	})

	t.Run("TestDriverGrantBucketAccess_ModifyPolicySuccess", func(t *testing.T) {
		existingPolicy := s3cli.NewBucketPolicy(*s3cli.NewPolicyStatement().
			WithSID("other-user").
			ForPrincipals("other-user").
			ForResources("bucket-test").
			ForSubResources("bucket-test").
			Allows().
			Actions("s3:GetObject"))

		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return existingPolicy, nil
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				return &s3.PutBucketPolicyOutput{}, nil
			},
		}

		server := &driver.ProvisionerServer{
			Provisioner:   "test",
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}

		req := &cosi.DriverGrantBucketAccessRequest{
			Name:     "ba-test2",
			BucketId: "bucket-test",
		}

		resp, err := server.DriverGrantBucketAccess(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "test-uuid", resp.AccountId)
	})

	t.Run("TestDriverGrantBucketAccess_UserCreateFail", func(t *testing.T) {
		mockIAMFail := mockIAM
		mockIAMFail.CreateUserFunc = func(ctx context.Context, username, displayName string) (admin.UserCredentials, error) {
			return admin.UserCredentials{}, fmt.Errorf("iam error")
		}

		server := &driver.ProvisionerServer{
			S3Client: mocks.MockProvisionerS3{
				GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
					return nil, nil
				},
			},
			NtnxIamClient: mockIAMFail,
		}

		req := &cosi.DriverGrantBucketAccessRequest{
			Name:     "ba-test",
			BucketId: "bucket-test",
		}

		_, err := server.DriverGrantBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create an IAM user")
	})

	t.Run("TestDriverGrantBucketAccess_GetPolicyError", func(t *testing.T) {
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return nil, awserr.New("UnknownError", "some s3 error", nil /* origErr */)
			},
		}

		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}

		req := &cosi.DriverGrantBucketAccessRequest{
			Name:     "ba-test",
			BucketId: "bucket-test",
		}

		_, err := server.DriverGrantBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetching policy failed")
	})

	t.Run("TestDriverGrantBucketAccess_PutPolicyError", func(t *testing.T) {
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return nil, awserr.New("NoSuchBucketPolicy", "no policy", nil /* origErr */)
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				return nil, fmt.Errorf("put policy failed")
			},
		}

		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}

		req := &cosi.DriverGrantBucketAccessRequest{
			Name:     "ba-test",
			BucketId: "bucket-test",
		}

		_, err := server.DriverGrantBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set policy")
	})
}

func TestDriverRevokeBucketAccess(t *testing.T) {
	t.Run("TestDriverRevokeBucketAccess_Success_LastStatementDeletesPolicy", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				assert.Equal(t, "user-uuid", uuid)
				return nil
			},
		}
		var deleteCalled bool
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				policy := s3cli.NewBucketPolicy(*s3cli.NewPolicyStatement().
					WithSID("user-uuid").
					ForPrincipals("user-uuid").
					ForResources("bucket-test").
					ForSubResources("bucket-test").
					Allows().
					Actions("s3:GetObject"))
				return policy, nil
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				t.Fatalf("PutBucketPolicy must not be called when the last statement is removed")
				return nil, nil
			},
			DeleteBucketPolicyFunc: func(bucket string) (*s3.DeleteBucketPolicyOutput, error) {
				deleteCalled = true
				assert.Equal(t, "", bucket)
				return &s3.DeleteBucketPolicyOutput{}, nil
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, deleteCalled, "DeleteBucketPolicy must be invoked when the last statement is removed")
	})

	t.Run("TestDriverRevokeBucketAccess_Success_KeepRemainingStatements", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return nil
			},
		}
		var putCalled bool
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return s3cli.NewBucketPolicy(
					*s3cli.NewPolicyStatement().WithSID("user-uuid-1").
						ForPrincipals("user-uuid-1").ForResources("bucket-test").
						ForSubResources("bucket-test").Allows().Actions("s3:GetObject"),
					*s3cli.NewPolicyStatement().WithSID("user-uuid-2").
						ForPrincipals("user-uuid-2").ForResources("bucket-test").
						ForSubResources("bucket-test").Allows().Actions("s3:GetObject"),
				), nil
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				putCalled = true
				assert.Len(t, policy.Statement, 1)
				assert.Equal(t, "user-uuid-2", policy.Statement[0].Sid)
				return &s3.PutBucketPolicyOutput{}, nil
			},
			DeleteBucketPolicyFunc: func(bucket string) (*s3.DeleteBucketPolicyOutput, error) {
				t.Fatalf("DeleteBucketPolicy must not be called when statements remain")
				return nil, nil
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid-1"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, putCalled, "PutBucketPolicy must be invoked with the surviving statement")
	})

	t.Run("TestDriverRevokeBucketAccess_NoSuchBucketPolicyTolerated", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return nil
			},
		}
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return nil, awserr.New("NoSuchBucketPolicy", "no policy", nil)
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("TestDriverRevokeBucketAccess_DeletePolicyError", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return nil
			},
		}
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return s3cli.NewBucketPolicy(*s3cli.NewPolicyStatement().
					WithSID("user-uuid").ForPrincipals("user-uuid").
					ForResources("bucket-test").ForSubResources("bucket-test").
					Allows().Actions("s3:GetObject")), nil
			},
			DeleteBucketPolicyFunc: func(bucket string) (*s3.DeleteBucketPolicyOutput, error) {
				return nil, fmt.Errorf("delete policy failed")
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete policy")
		assert.Nil(t, resp)
	})

	t.Run("TestDriverRevokeBucketAccess_RemoveUserError", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return errors.New("delete user failed")
			},
		}
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				policy := s3cli.NewBucketPolicy(*s3cli.NewPolicyStatement().
					WithSID("user-uuid").
					ForPrincipals("user-uuid").
					ForResources("bucket-test").
					ForSubResources("bucket-test").
					Allows().
					Actions("s3:GetObject"))
				return policy, nil
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				return &s3.PutBucketPolicyOutput{}, nil
			},
			DeleteBucketPolicyFunc: func(bucket string) (*s3.DeleteBucketPolicyOutput, error) {
				return &s3.DeleteBucketPolicyOutput{}, nil
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete user")
		assert.Nil(t, resp)
	})

	t.Run("TestDriverRevokeBucketAccess_GetPolicyError", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return nil
			},
		}
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				return nil, awserr.New("UnknownError", "some s3 error", nil /* origErr */)
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get policy")
		assert.Nil(t, resp)
	})

	t.Run("TestDriverRevokeBucketAccess_PutPolicyError", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return nil
			},
		}
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				statement1 := *s3cli.NewPolicyStatement().
					WithSID("user-uuid-1").
					ForPrincipals("user-uuid").
					ForResources("bucket-test").
					ForSubResources("bucket-test").
					Allows().
					Actions("s3:GetObject")

				statement2 := *s3cli.NewPolicyStatement().
					WithSID("user-uuid-2").
					ForPrincipals("user-uuid").
					ForResources("bucket-test").
					ForSubResources("bucket-test").
					Allows().
					Actions("s3:GetObject")

				policy := s3cli.NewBucketPolicy(statement1, statement2)

				return policy, nil
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				return nil, fmt.Errorf("put policy failed")
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "user-uuid-1"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set policy")
		assert.Nil(t, resp)
	})

	t.Run("TestDriverRevokeBucketAccess_UserNotFound", func(t *testing.T) {
		mockIAM := mocks.MockIAM{
			RemoveUserFunc: func(ctx context.Context, uuid string) error {
				return nil
			},
		}
		mockS3 := mocks.MockProvisionerS3{
			GetBucketPolicyFunc: func(bucket string) (*s3cli.BucketPolicy, error) {
				policy := s3cli.NewBucketPolicy(*s3cli.NewPolicyStatement().
					WithSID("user-uuid").
					ForPrincipals("user-uuid").
					ForResources("bucket-test").
					ForSubResources("bucket-test").
					Allows().
					Actions("s3:GetObject"))
				return policy, nil
			},
			PutBucketPolicyFunc: func(bucket string, policy s3cli.BucketPolicy) (*s3.PutBucketPolicyOutput, error) {
				return &s3.PutBucketPolicyOutput{}, nil
			},
		}
		server := &driver.ProvisionerServer{
			S3Client:      mockS3,
			NtnxIamClient: mockIAM,
		}
		req := &cosi.DriverRevokeBucketAccessRequest{AccountId: "fake-user-uuid"}
		resp, err := server.DriverRevokeBucketAccess(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
}
