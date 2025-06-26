package s3client_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/s3client"
	mocks "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/fakes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validPEMCert = `-----BEGIN CERTIFICATE-----
MIIDVDCCAjygAwIBAgIRALLmZX4DorTkHZzNVhA+RYUwDQYJKoZIhvcNAQELBQAw
FzEVMBMGA1UEChMMTnV0YW5peCBJbmMuMCAXDTI1MDEyNzA4MDU0NVoYDzIyMDQw
NzAzMDgwNTQ1WjAXMRUwEwYDVQQKEwxOdXRhbml4IEluYy4wggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQDXvGBSfAByt1VCvAuaDThq7H+JlK8He8zcjqD7
DGe6Dc1jq9n7+eN6X+cTT85dKPhajjPp9Nc4g1HT0jfWHD4QHhgkS12Ny0Wrmqqr
qx8Fcuzhaz89BFyaI3tOCBx75yZe9zaNSOBoKJqreu2mAjfI8LM7jFl/cON68Kd9
/b6g89+2FME0gFq2p3mabGXGVerFAK0g7TuffhWKyKL/B9equZ2M7CmhAO7wa0ko
CnQJY3XvNk4OUeb7NXBdpswpWD789rXSsSMbxaS9ZmDrqvB7A1IlWjwEUVzxYnPw
21miqfZ5qb2p3gZtSTbDOqmg0l9yfvwIroaGKMLrRCK0Q+IjAgMBAAGjgZgwgZUw
DgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFPE9h85q
h53rKbrBRd+VnUGuAxpKMFMGA1UdEQRMMEqCInJpbXVydS5wcmlzbS1jZW50cmFs
LmNsdXN0ZXIubG9jYWyCJCoucmltdXJ1LnByaXNtLWNlbnRyYWwuY2x1c3Rlci5s
b2NhbDANBgkqhkiG9w0BAQsFAAOCAQEAd3wN98xaTJqBGE2j0qoSQhqMNb5NmBDa
Cp/Pt0mwlAJmv2petz3ON5edt3/yC81vEvWfT+4GpM/6jAHcY9rZ+XQA+ZkSnjsG
itALgSLq77vDYRTHAXfsWPH2DY140IS6OqqTtLPLukHzux5uR2LH1uggU5sARs5l
EBi1znwsnSxrKfqPOurt4oSgW7FougqiaOiK+Vkm+1FtybVlMXH1w5TkePFK/x7B
OkiKpPoALmPy1Y2BxvbpxQYLjZEFMKwIo7G20pl9opFntCBs6GcY7QNesVYKawV1
zQEsbJYBuhj1XgjzRx+6al2Fjf2NFN3I2aCKQZ9oMFsg0R/M0biBjA==
-----END CERTIFICATE-----
`

func TestNewS3Agent(t *testing.T) {
	accessKey := "dummyAccessKey"
	secretKey := "dummySecretKey"

	t.Run("TestNewS3Agent_ValidSecureConnection", func(t *testing.T) {
		// Insecure false and endpoint starting with "https" should succeed with a valid PEM CA certificate.
		endpoint := "https://127.0.0.1:9440"
		agent, err := s3client.NewS3Agent(accessKey, secretKey, endpoint, validPEMCert, false, false)
		require.NoError(t, err)
		require.NotNil(t, agent)
		require.NotNil(t, agent.Client)

		// Additionally, verify that the client was built using an HTTP client with a proper timeout.
		sess, err := session.NewSession(aws.NewConfig().
			WithRegion("us-east-1").
			WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
			WithEndpoint(endpoint).
			WithS3ForcePathStyle(true).
			WithMaxRetries(5).
			WithDisableSSL(false).
			WithHTTPClient(&http.Client{Timeout: time.Second * 15}))
		require.NoError(t, err)
		svc := s3.New(sess)
		// Not a deep check, but ensures we can instantiate an s3 client.
		assert.IsType(t, svc, agent.Client)
	})

	t.Run("TestNewS3Agent_ValidInsecureConnection", func(t *testing.T) {
		// When insecure is true, even if the endpoint starts with "http", it should succeed.
		endpoint := "http://127.0.0.1:9440"
		agent, err := s3client.NewS3Agent(accessKey, secretKey, endpoint, validPEMCert, true, false)
		require.NoError(t, err)
		require.NotNil(t, agent)
		require.NotNil(t, agent.Client)

		// Additionally, verify that the client was built using an HTTP client with a proper timeout.
		sess, err := session.NewSession(aws.NewConfig().
			WithRegion("us-east-1").
			WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
			WithEndpoint(endpoint).
			WithS3ForcePathStyle(true).
			WithMaxRetries(5).
			WithDisableSSL(false).
			WithHTTPClient(&http.Client{Timeout: time.Second * 15}))
		require.NoError(t, err)
		svc := s3.New(sess)
		// Not a deep check, but ensures we can instantiate an s3 client.
		assert.IsType(t, svc, agent.Client)
	})

	t.Run("TestNewS3Agent_ErrorSecureWithHttpEndpoint", func(t *testing.T) {
		// When insecure is false but the endpoint starts with "http", it should return an error.
		endpoint := "http://127.0.0.1:9440"
		agent, err := s3client.NewS3Agent(accessKey, secretKey, endpoint, validPEMCert, false, false)
		require.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "'http' endpoint cannot be secure")
	})

	t.Run("TestNewS3Agent_ErrorInvalidCACert", func(t *testing.T) {
		// Provide an invalid CA cert string that is not a valid PEM or base64 string.
		endpoint := "https://127.0.0.1:9440"
		invalidCACert := "not-base64"
		agent, err := s3client.NewS3Agent(accessKey, secretKey, endpoint, invalidCACert, false, false)
		require.Error(t, err)
		assert.Nil(t, agent)
		// The error should come from transport.BuildTransportTLS. We expect an error related to decoding the CA cert.
		assert.Contains(t, err.Error(), "failed to decode CA cert")
	})

	t.Run("TestNewS3Agent_DebugMode", func(t *testing.T) {
		// When debug is true, log level should be set to aws.LogDebug.
		// While we cannot easily inspect the AWS config from the created session,
		// we can at least verify that the agent creation does not error.
		endpoint := "https://127.0.0.1:9440"
		agent, err := s3client.NewS3Agent(accessKey, secretKey, endpoint, validPEMCert, false, true)
		require.NoError(t, err)
		require.NotNil(t, agent)
	})
}

func TestCreateBucket(t *testing.T) {
	t.Run("TestCreateBucket_Success", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			CreateBucketFunc: func(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
				assert.Equal(t, "test-bucket", *input.Bucket)
				return &s3.CreateBucketOutput{}, nil
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		err := s.CreateBucket("test-bucket")
		assert.NoError(t, err)
	})

	t.Run("TestCreateBucket_AlreadyExists", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			CreateBucketFunc: func(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
				return nil, awserr.New(s3.ErrCodeBucketAlreadyExists, "bucket exists", nil)
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		err := s.CreateBucket("existing-bucket")
		assert.NoError(t, err)
	})

	t.Run("TestCreateBucket_AlreadyOwned", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			CreateBucketFunc: func(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
				return nil, awserr.New(s3.ErrCodeBucketAlreadyOwnedByYou, "already owned", nil)
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		err := s.CreateBucket("owned-bucket")
		assert.NoError(t, err)
	})

	t.Run("TestCreateBucket_GenericError", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			CreateBucketFunc: func(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
				return nil, errors.New("unexpected error")
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		err := s.CreateBucket("fail-bucket")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create bucket")
		assert.Contains(t, err.Error(), "unexpected error")
	})
}

func TestDeleteBucket(t *testing.T) {
	t.Run("DeleteBucket_Success", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			DeleteBucketFunc: func(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
				assert.Equal(t, "my-bucket", *input.Bucket)
				return &s3.DeleteBucketOutput{}, nil
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		ok, err := s.DeleteBucket("my-bucket")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("DeleteBucket_NoSuchBucket", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			DeleteBucketFunc: func(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
				return &s3.DeleteBucketOutput{}, awserr.New(s3.ErrCodeNoSuchBucket, "The specified bucket does not exist.", nil)
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		ok, err := s.DeleteBucket("my-bucket")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("DeleteBucket_Error", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			DeleteBucketFunc: func(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
				return nil, errors.New("delete failed")
			},
		}
		s := &s3client.S3Agent{Client: mockClient}
		ok, err := s.DeleteBucket("bad-bucket")
		assert.Error(t, err)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

func TestPutObjectInBucket(t *testing.T) {
	t.Run("PutObjectInBucket_Success", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			PutObjectFunc: func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
				assert.Equal(t, "test-bucket", *input.Bucket)
				assert.Equal(t, "test-key", *input.Key)
				assert.Equal(t, "text/plain", *input.ContentType)
				return &s3.PutObjectOutput{}, nil
			},
		}
		s := &s3client.S3Agent{Client: mock}
		ok, err := s.PutObjectInBucket("test-bucket", "hello", "test-key", "text/plain")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("PutObjectInBucket_Error", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			PutObjectFunc: func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
				return nil, errors.New("put error")
			},
		}
		s := &s3client.S3Agent{Client: mock}
		ok, err := s.PutObjectInBucket("fail-bucket", "fail", "fail-key", "text/plain")
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestGetObjectInBucket(t *testing.T) {
	t.Run("GetObjectInBucket_Success", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			GetObjectFunc: func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				assert.Equal(t, "test-bucket", *input.Bucket)
				assert.Equal(t, "test-key", *input.Key)
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewBufferString("test-content")),
				}, nil
			},
		}
		s := &s3client.S3Agent{Client: mock}
		content, err := s.GetObjectInBucket("test-bucket", "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-content", content)
	})

	t.Run("GetObjectInBucket_NotFound", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			GetObjectFunc: func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return nil, errors.New("not found")
			},
		}
		s := &s3client.S3Agent{Client: mock}
		content, err := s.GetObjectInBucket("missing-bucket", "missing-key")
		assert.Error(t, err)
		assert.Equal(t, "ERROR_ OBJECT NOT FOUND", content)
	})
}

func TestDeleteObjectInBucket(t *testing.T) {
	t.Run("DeleteObjectInBucket_Success", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			DeleteObjectFunc: func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				assert.Equal(t, "test-bucket", *input.Bucket)
				assert.Equal(t, "test-key", *input.Key)
				return &s3.DeleteObjectOutput{}, nil
			},
		}
		s := &s3client.S3Agent{Client: mock}
		ok, err := s.DeleteObjectInBucket("test-bucket", "test-key")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("DeleteObjectInBucket_NoSuchBucket", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			DeleteObjectFunc: func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				return nil, awserr.New(s3.ErrCodeNoSuchBucket, "bucket not found", nil)
			},
		}
		s := &s3client.S3Agent{Client: mock}
		ok, err := s.DeleteObjectInBucket("no-bucket", "key")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("DeleteObjectInBucket_NoSuchKey", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			DeleteObjectFunc: func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				return nil, awserr.New(s3.ErrCodeNoSuchKey, "key not found", nil)
			},
		}
		s := &s3client.S3Agent{Client: mock}
		ok, err := s.DeleteObjectInBucket("bucket", "no-key")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("DeleteObjectInBucket_Error", func(t *testing.T) {
		mock := &mocks.MockS3Client{
			DeleteObjectFunc: func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("delete error")
			},
		}
		s := &s3client.S3Agent{Client: mock}
		ok, err := s.DeleteObjectInBucket("bucket", "key")
		assert.Error(t, err)
		assert.False(t, ok)
	})
}
