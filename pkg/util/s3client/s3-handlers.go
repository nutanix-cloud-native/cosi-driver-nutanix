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
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/transport"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"k8s.io/klog/v2"
)

const (
	ErrNoSuchBucket = "NoSuchBucket"
)

type S3iface interface {
	CreateBucket(name string) error
	DeleteBucket(name string) (bool, error)
	GetBucketPolicy(bucket string) (*BucketPolicy, error)
	PutBucketPolicy(bucket string, policy BucketPolicy) (*s3.PutBucketPolicyOutput, error)
}

type S3client interface {
	CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
	DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error)
	DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	GetBucketPolicy(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error)
	PutBucketPolicy(input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error)
}

// S3Agent wraps the s3.S3 structure to allow for wrapper methods
type S3Agent struct {
	Client S3client
}

func NewS3Agent(accessKey, secretKey, endpoint, caCert string, insecure, debug bool) (*S3Agent, error) {
	const nutanixRegion = "us-east-1"

	tlsConfig := transport.TlsConfig{
		CACert:   caCert,
		Insecure: insecure,
		Endpoint: endpoint,
	}

	if !tlsConfig.Insecure && strings.HasPrefix(endpoint, "http://") {
		return nil, fmt.Errorf("'http' endpoint cannot be secure. Use an `https` endpoint or use insecure connection")
	}

	logLevel := aws.LogOff
	if debug {
		logLevel = aws.LogDebug
	}

	transport, err := transport.BuildTransportTLS(tlsConfig)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: transport,
	}

	sess, err := session.NewSession(
		aws.NewConfig().
			WithRegion(nutanixRegion).
			WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
			WithEndpoint(endpoint).
			WithS3ForcePathStyle(true).
			WithMaxRetries(5).
			WithDisableSSL(tlsConfig.Insecure).
			WithHTTPClient(client).
			WithLogLevel(logLevel),
	)
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess)
	return &S3Agent{
		Client: svc,
	}, nil
}

// CreateBucket creates a bucket with the given name
func (s *S3Agent) CreateBucket(name string) error {

	klog.InfoS("Creating bucket", "name", name)
	bucketInput := &s3.CreateBucketInput{
		Bucket: &name,
	}
	_, err := s.Client.CreateBucket(bucketInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			klog.InfoS("DEBUG: after s3 call", "ok", ok, "aerr", aerr)
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists:
				klog.InfoS("Bucket already exists", "name", name)
				return nil
			case s3.ErrCodeBucketAlreadyOwnedByYou:
				klog.InfoS("Bucket already owned by you", "name", name)
				return nil
			}
		}
		return fmt.Errorf("failed to create bucket %q error %w", name, err)
	}
	klog.InfoS("Successfully created bucket", "name", name)

	return nil
}

// DeleteBucket function deletes given bucket using s3 client
func (s *S3Agent) DeleteBucket(name string) (bool, error) {
	_, err := s.Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			klog.InfoS("DEBUG: after s3 call", "ok", ok, "aerr", aerr)
			if aerr.Code() == s3.ErrCodeNoSuchBucket {
				klog.InfoS("Bucket does not exist", "name", name)
				return true, nil
			}
		}
		klog.ErrorS(err, "failed to delete bucket")
		return false, err
	}
	return true, nil
}

// PutObjectInBucket function puts an object in a bucket using s3 client
func (s *S3Agent) PutObjectInBucket(bucketname string, body string, key string,
	contentType string) (bool, error) {
	_, err := s.Client.PutObject(&s3.PutObjectInput{
		Body:        strings.NewReader(body),
		Bucket:      &bucketname,
		Key:         &key,
		ContentType: &contentType,
	})
	if err != nil {
		klog.ErrorS(err, "failed to put object in bucket")
		return false, err

	}
	return true, nil
}

// GetObjectInBucket function retrieves an object from a bucket using s3 client
func (s *S3Agent) GetObjectInBucket(bucketname string, key string) (string, error) {
	result, err := s.Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketname),
		Key:    aws.String(key),
	})

	if err != nil {
		klog.ErrorS(err, "failed to retrieve object from bucket")
		return "ERROR_ OBJECT NOT FOUND", err

	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DeleteObjectInBucket function deletes given bucket using s3 client
func (s *S3Agent) DeleteObjectInBucket(bucketname string, key string) (bool, error) {
	_, err := s.Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketname),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return true, nil
			case s3.ErrCodeNoSuchKey:
				return true, nil
			}
		}
		klog.ErrorS(err, "failed to delete object from bucket")
		return false, err

	}
	return true, nil
}
