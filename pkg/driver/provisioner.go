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
	"fmt"

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
	provisioner   string
	s3Client      *s3cli.S3Agent
	ntnxIamClient *ntnxIam.API
}

// ProvisionerCreateBucket is a method for creating buckets
// It is expected to create the same bucket given a bucketName and protocol
// If the bucket already exists, then it MUST return codes.AlreadyExists
// Return values
//    nil -                   Bucket successfully created
//    codes.AlreadyExists -   Bucket already exists. No more retries
//    non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *ProvisionerServer) ProvisionerCreateBucket(ctx context.Context,
	req *cosi.ProvisionerCreateBucketRequest) (*cosi.ProvisionerCreateBucketResponse, error) {
	klog.InfoS("Using Nutanix Object store to create Backend Bucket")
	protocol := req.GetProtocol()
	if protocol == nil {
		klog.ErrorS(errNilProtocol, "Protocol is nil")
		return nil, status.Error(codes.InvalidArgument, "Protocol is nil")
	}
	s3 := protocol.GetS3()
	if s3 == nil {
		klog.ErrorS(errs3ProtocolMissing, "S3 protocol is missing, only S3 is supported")
		return nil, status.Error(codes.InvalidArgument, "only S3 protocol supported")
	}
	bucketName := req.GetName()
	klog.V(3).InfoS("Creating Bucket", "name", bucketName)

	err := s.s3Client.CreateBucket(bucketName)
	if err != nil {
		// Check to see if the bucket already exists by above API
		klog.ErrorS(err, "failed to create bucket", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to create bucket")
	}
	klog.InfoS("Successfully created Backend Bucket on Nutanix Objects", "bucketName", bucketName)

	return &cosi.ProvisionerCreateBucketResponse{
		BucketId: bucketName,
	}, nil
}

func (s *ProvisionerServer) ProvisionerDeleteBucket(ctx context.Context,
	req *cosi.ProvisionerDeleteBucketRequest) (*cosi.ProvisionerDeleteBucketResponse, error) {
	klog.InfoS("Deleting bucket", "id", req.GetBucketId())
	if _, err := s.s3Client.DeleteBucket(req.GetBucketId()); err != nil {
		klog.ErrorS(err, "failed to delete bucket %q", req.GetBucketId())
		return nil, status.Error(codes.Internal, "failed to delete bucket")
	}
	klog.InfoS("Successfully deleted Bucket", "id", req.GetBucketId())

	return &cosi.ProvisionerDeleteBucketResponse{}, nil
}

func (s *ProvisionerServer) ProvisionerGrantBucketAccess(ctx context.Context,
	req *cosi.ProvisionerGrantBucketAccessRequest) (*cosi.ProvisionerGrantBucketAccessResponse, error) {
	userName := req.GetAccountName() + "@nutanix.com"
	displayName := s.ntnxIamClient.AccountName + "_" + req.GetAccountName()
	bucketName := req.GetBucketId()
	accessPolicy := req.GetAccessPolicy()
	klog.InfoS("Granting user accessPolicy to bucket", "userName", userName, "displayName",
		displayName, "bucketName", bucketName, "accessPolicy", accessPolicy)

	// Format : {type: "external", email: <userName>@nutanix.com, displayname: <accountName>_<userName> (optional)}
	user, err := s.ntnxIamClient.CreateUser(ctx, userName, displayName)
	if err != nil {
		klog.ErrorS(err, "failed to create an IAM user for Nutanix Objects")
		return nil, err
	}

	// Fetch Bucket Policy
	policy, err := s.s3Client.GetBucketPolicy(bucketName)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != "NoSuchBucketPolicy" {
			return nil, status.Error(codes.Internal, "fetching policy failed")
		}
	}

	// Share bucket with the newly created IAM user
	statement := s3cli.NewPolicyStatement().
		WithSID(userName).
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
	_, err = s.s3Client.PutBucketPolicy(bucketName, *policy)
	if err != nil {
		klog.ErrorS(err, "failed to set policy")
		return nil, status.Error(codes.Internal, "failed to set policy")
	}

	return &cosi.ProvisionerGrantBucketAccessResponse{
		AccountId:   user.Users[0].UUID,
		Credentials: fetchUserCredentials(user, s.ntnxIamClient.Endpoint, bucketName),
	}, nil
}

func (s *ProvisionerServer) ProvisionerRevokeBucketAccess(ctx context.Context,
	req *cosi.ProvisionerRevokeBucketAccessRequest) (*cosi.ProvisionerRevokeBucketAccessResponse, error) {
	// TODO : instead of deleting user, revoke its permission and delete only if no more bucket attached to it
	klog.InfoS("Deleting user", "id", req.GetAccountId())

	err := s.ntnxIamClient.RemoveUser(ctx, req.GetAccountId())
	if err != nil {
		klog.ErrorS(err, "failed to delete user")
	}
	return &cosi.ProvisionerRevokeBucketAccessResponse{}, nil
}

func fetchUserCredentials(user ntnxIam.NutanixUserResp, endpoint, bucketName string) string {

	return fmt.Sprintf("AWS_ACCESS_KEY=%s;AWS_SECRET_KEY=%s;ENDPOINT=%s;BUCKET_ID=%s",
		user.Users[0].BucketsAccessKeys[0].AccessKeyID,
		user.Users[0].BucketsAccessKeys[0].SecretAccessKey, endpoint, bucketName)
}
