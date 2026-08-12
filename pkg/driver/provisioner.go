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

package driver

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/awserr"
	ntnxIam "github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	s3cli "github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/s3client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

// contains two clients
// 1.) for ntnxIamClientOps : mainly for user related operations
// 2.) for S3 operations : mainly for bucket related operations
type ProvisionerServer struct {
	Provisioner   string
	S3Client      s3cli.S3iface
	NtnxIamClient ntnxIam.IAMiface
}

// ProvisionerCreateBucket is a method for creating buckets
// It is expected to create the same bucket given a bucketName and protocol
// If the bucket already exists, then it MUST return codes.AlreadyExists
// Return values
//
//	nil -                   Bucket successfully created
//	codes.AlreadyExists -   Bucket already exists. No more retries
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *ProvisionerServer) DriverCreateBucket(ctx context.Context,
	req *cosi.DriverCreateBucketRequest) (*cosi.DriverCreateBucketResponse, error) {
	klog.InfoS("Using Nutanix Object store to create Backend Bucket")

	// Get the name of the bucket from the request which is formed
	// by getting the name from the bucket object which is created
	// by the cosi-central-controller.
	bucketName := req.GetName()
	klog.V(3).InfoS("Creating Bucket", "name", bucketName)

	err := s.S3Client.CreateBucket(bucketName)
	if err != nil {
		// Check to see if the bucket already exists by above API
		klog.ErrorS(err, "failed to create bucket", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to create bucket")
	}
	klog.InfoS("Successfully created Backend Bucket on Nutanix Objects", "bucketName", bucketName)

	// Not sure what bucketInfo Stores
	return &cosi.DriverCreateBucketResponse{
		BucketId: bucketName,
	}, nil
}

func (s *ProvisionerServer) DriverDeleteBucket(ctx context.Context,
	req *cosi.DriverDeleteBucketRequest) (*cosi.DriverDeleteBucketResponse, error) {
	klog.InfoS("Deleting bucket", "id", req.GetBucketId())
	if _, err := s.S3Client.DeleteBucket(req.GetBucketId()); err != nil {
		klog.ErrorS(err, "failed to delete bucket %q", req.GetBucketId())
		return nil, status.Error(codes.Internal, "failed to delete bucket")
	}
	klog.InfoS("Successfully deleted Bucket", "id", req.GetBucketId())

	return &cosi.DriverDeleteBucketResponse{}, nil
}

func (s *ProvisionerServer) DriverGrantBucketAccess(ctx context.Context,
	req *cosi.DriverGrantBucketAccessRequest) (*cosi.DriverGrantBucketAccessResponse, error) {
	// Form the username for this new user.
	// username is formed by concatenating the account name prefix
	// stored in req which is of the form- "ba-<BucketAccessUUID>"
	// with the suffix "@nutanix.com"
	userName := req.GetName() + "@nutanix.com"
	displayName := s.NtnxIamClient.GetAccountName() + "_" + req.GetName()
	bucketName := req.GetBucketId()

	klog.InfoS("Granting user accessPolicy to bucket", "userName", userName, "displayName",
		displayName, "bucketName", bucketName)

	// Fetch Bucket Policy
	policy, err := s.S3Client.GetBucketPolicy(bucketName)
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NoSuchBucketPolicy" {
			klog.ErrorS(err, "failed to get bucket policy", "bucketName", bucketName)
			return nil, status.Error(codes.Internal, "fetching policy failed")
		}
	}

	// Format : {type: "external", username: <userName>@nutanix.com, displayname: <accountName>_<userName> (optional)}
	user, err := s.NtnxIamClient.CreateUser(ctx, userName, displayName)
	if err != nil {
		klog.ErrorS(err, "failed to create an IAM user for Nutanix Objects")
		return nil, status.Error(codes.Internal, "failed to create an IAM user")
	}

	accountId := user.UserID

	// Share bucket with the newly created IAM user
	statement := s3cli.NewPolicyStatement().
		WithSID(accountId).
		ForPrincipals(userName).
		ForResources(bucketName).
		ForSubResources(bucketName).
		Allows().
		Actions(s3cli.AllowedActions...)
	if policy == nil {
		policy = s3cli.NewBucketPolicy(*statement)
	} else {
		policy = policy.ModifyBucketPolicy(*statement)
	}
	_, err = s.S3Client.PutBucketPolicy(bucketName, *policy)
	if err != nil {
		klog.ErrorS(err, "failed to set policy")
		return nil, status.Error(codes.Internal, "failed to set policy")
	}

	klog.InfoS("Successfully granted access to user.", "userName", userName, "displayName", displayName, "bucketName", bucketName)

	return &cosi.DriverGrantBucketAccessResponse{
		AccountId:   accountId,
		Credentials: fetchUserCredentials(user, s.NtnxIamClient.GetEndpoint()),
	}, nil
}

func (s *ProvisionerServer) DriverRevokeBucketAccess(ctx context.Context,
	req *cosi.DriverRevokeBucketAccessRequest) (*cosi.DriverRevokeBucketAccessResponse, error) {

	klog.InfoS("Revoking bucket access", "accountId", req.GetAccountId(), "bucketId", req.GetBucketId())

	policy, err := s.S3Client.GetBucketPolicy(req.GetBucketId())
	if err != nil {
		// A missing bucket policy is not fatal here: the user we are about
		// to delete simply has no statement to remove. Any other error must
		// be surfaced so the operation is retried.
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NoSuchBucketPolicy" {
			klog.ErrorS(err, "failed to get policy")
			return nil, status.Error(codes.Internal, "failed to get policy")
		}
		policy = nil
	}
	if policy != nil {
		klog.InfoS("Removing bucket policy statement", "accountId", req.GetAccountId())
		policy = policy.DropPolicyStatements(req.GetAccountId())
		if len(policy.Statement) > 0 {
			if _, err = s.S3Client.PutBucketPolicy(req.GetBucketId(), *policy); err != nil {
				klog.ErrorS(err, "failed to set policy")
				return nil, status.Error(codes.Internal, "failed to set policy")
			}
		} else {
			// S3 rejects PutBucketPolicy with an empty statement list, so
			// clear the policy explicitly. If we skipped this call the
			// stale statement (referencing the user we are about to
			// delete) would remain on the bucket and break subsequent
			// GrantBucketAccess with "Invalid principal in policy".
			if _, err = s.S3Client.DeleteBucketPolicy(req.GetBucketId()); err != nil {
				if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NoSuchBucketPolicy" {
					klog.ErrorS(err, "failed to delete policy")
					return nil, status.Error(codes.Internal, "failed to delete policy")
				}
			}
		}
	}

	klog.InfoS("Deleting user", "id", req.GetAccountId())

	err = s.NtnxIamClient.RemoveUser(ctx, req.GetAccountId())
	if err != nil {
		klog.ErrorS(err, "failed to delete user")
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	klog.InfoS("Successfully revoked access of user", "id", req.GetAccountId(), "bucketName", req.GetBucketId())
	return &cosi.DriverRevokeBucketAccessResponse{}, nil
}

func fetchUserCredentials(user ntnxIam.UserCredentials, endpoint string) map[string]*cosi.CredentialDetails {

	secretsMap := make(map[string]string)
	secretsMap["accessKeyID"] = user.AccessKeyID
	secretsMap["accessSecretKey"] = user.SecretAccessKey
	secretsMap["endpoint"] = endpoint
	// region mapping needs to be updated
	secretsMap["region"] = "us-east-1"

	creds := &cosi.CredentialDetails{
		Secrets: secretsMap,
	}

	credDetailsMap := make(map[string]*cosi.CredentialDetails)
	credDetailsMap["s3"] = creds
	return credDetailsMap
}
