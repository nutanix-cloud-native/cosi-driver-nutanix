package s3client_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/s3client"
	mocks "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/fakes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPutBucketPolicy(t *testing.T) {
	t.Run("TestPutBucketPolicy_Success", func(t *testing.T) {
		ps := s3client.PolicyStatement{
			Sid:       "test-sid",
			Effect:    "Allow",
			Principal: map[string][]string{"AWS": {"user1", "user2"}},
			Action:    []s3client.Action{s3client.ListBucket, s3client.GetObject},
			Resource:  []string{"arn:aws:s3:::test-bucket"},
		}
		policy := s3client.NewBucketPolicy(ps)

		var capturedInput *s3.PutBucketPolicyInput
		mockClient := &mocks.MockS3Client{
			PutBucketPolicyFunc: func(input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
				capturedInput = input
				return &s3.PutBucketPolicyOutput{}, nil
			},
		}
		agent := &s3client.S3Agent{Client: mockClient}

		out, err := agent.PutBucketPolicy("test-bucket", *policy)
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, capturedInput)
		assert.Equal(t, "test-bucket", *capturedInput.Bucket)

		var unmarshalledPolicy s3client.BucketPolicy
		err = json.Unmarshal([]byte(*capturedInput.Policy), &unmarshalledPolicy)
		require.NoError(t, err)
		require.Len(t, unmarshalledPolicy.Statement, 1)
		assert.Equal(t, "test-sid", unmarshalledPolicy.Statement[0].Sid)
	})

	t.Run("TestPutBucketPolicy_Error", func(t *testing.T) {
		expectedErr := errors.New("failed to put policy")
		mockClient := &mocks.MockS3Client{
			PutBucketPolicyFunc: func(input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
				return nil, expectedErr
			},
		}
		agent := &s3client.S3Agent{Client: mockClient}
		policy := s3client.NewBucketPolicy()
		out, err := agent.PutBucketPolicy("test-bucket", *policy)
		assert.Error(t, err)
		assert.Nil(t, out)
		assert.Equal(t, expectedErr, err)
	})
}

func TestGetBucketPolicy(t *testing.T) {
	t.Run("TestGetBucketPolicy_Success", func(t *testing.T) {
		ps := s3client.PolicyStatement{
			Sid:       "test-sid",
			Effect:    "Allow",
			Principal: map[string][]string{"AWS": {"userA"}},
			Action:    []s3client.Action{s3client.GetBucketLocation},
			Resource:  []string{"arn:aws:s3:::my-bucket"},
		}
		policy := s3client.NewBucketPolicy(ps)
		serialized, err := json.Marshal(policy)
		require.NoError(t, err)

		mockClient := &mocks.MockS3Client{
			GetBucketPolicyFunc: func(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
				assert.Equal(t, "my-bucket", *input.Bucket)
				return &s3.GetBucketPolicyOutput{
					Policy: aws.String(string(serialized)),
				}, nil
			},
		}
		agent := &s3client.S3Agent{Client: mockClient}
		retPolicy, err := agent.GetBucketPolicy("my-bucket")
		require.NoError(t, err)
		require.NotNil(t, retPolicy)
		assert.Equal(t, policy.Version, retPolicy.Version)
		require.Len(t, retPolicy.Statement, 1)
		assert.Equal(t, "test-sid", retPolicy.Statement[0].Sid)
	})

	t.Run("TestGetBucketPolicy_S3Error", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			GetBucketPolicyFunc: func(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
				return nil, errors.New("get policy error")
			},
		}
		agent := &s3client.S3Agent{Client: mockClient}
		policy, err := agent.GetBucketPolicy("nonexistent-bucket")
		assert.Error(t, err)
		assert.Nil(t, policy)
	})

	t.Run("TestGetBucketPolicy_UnmarshalError", func(t *testing.T) {
		mockClient := &mocks.MockS3Client{
			GetBucketPolicyFunc: func(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
				return &s3.GetBucketPolicyOutput{Policy: aws.String("invalid-json")}, nil
			},
		}
		agent := &s3client.S3Agent{Client: mockClient}
		policy, err := agent.GetBucketPolicy("bucket")
		assert.Error(t, err)
		assert.Nil(t, policy)
	})
}

func TestModifyBucketPolicy(t *testing.T) {
	t.Run("TestModifyBucketPolicy_UpdateAndAddStatements", func(t *testing.T) {
		ps1 := s3client.PolicyStatement{Sid: "sid1", Effect: "Allow"}
		ps2 := s3client.PolicyStatement{Sid: "sid2", Effect: "Allow"}
		bp := s3client.NewBucketPolicy(ps1, ps2)

		newPS1 := s3client.PolicyStatement{Sid: "sid1", Effect: "Deny"}
		newPS3 := s3client.PolicyStatement{Sid: "sid3", Effect: "Allow"}

		modified := bp.ModifyBucketPolicy(newPS1, newPS3)
		require.Len(t, modified.Statement, 3)

		var foundSID1, foundSID2, foundSID3 bool
		for _, ps := range modified.Statement {
			switch ps.Sid {
			case "sid1":
				foundSID1 = true
				assert.Equal(t, "Deny", string(ps.Effect))
			case "sid2":
				foundSID2 = true
			case "sid3":
				foundSID3 = true
			}
		}
		assert.True(t, foundSID1 && foundSID2 && foundSID3)
	})
}

func TestDropPolicyStatements(t *testing.T) {
	t.Run("TestDropPolicyStatements_RemovesCorrectSID", func(t *testing.T) {
		ps1 := s3client.PolicyStatement{Sid: "sid1"}
		ps2 := s3client.PolicyStatement{Sid: "sid2"}
		ps3 := s3client.PolicyStatement{Sid: "sid3"}
		bp := s3client.NewBucketPolicy(ps1, ps2, ps3)

		modified := bp.DropPolicyStatements("sid2")
		require.Len(t, modified.Statement, 2)
		for _, ps := range modified.Statement {
			assert.NotEqual(t, "sid2", ps.Sid)
		}
	})
}

func TestEjectPrincipals(t *testing.T) {
	t.Run("TestEjectPrincipals_RemovesPrincipal", func(t *testing.T) {
		ps := s3client.PolicyStatement{
			Sid:    "sid1",
			Effect: "Allow",
			Principal: map[string][]string{
				"AWS": {"user1", "user2", "user3"},
			},
		}
		bp := s3client.NewBucketPolicy(ps)

		modified := bp.EjectPrincipals("user2")
		require.Len(t, modified.Statement, 1)
		principals := modified.Statement[0].Principal["AWS"]
		assert.NotContains(t, principals, "user2")
		assert.Contains(t, principals, "user1")
		assert.Contains(t, principals, "user3")
	})
}

func TestPolicyStatementMethods(t *testing.T) {
	t.Run("TestPolicyStatementMethods_ForPrincipals", func(t *testing.T) {
		ps := s3client.NewPolicyStatement()
		ps.ForPrincipals("alice", "bob")
		principals := ps.Principal["AWS"]
		assert.Contains(t, principals, "alice")
		assert.Contains(t, principals, "bob")
	})

	t.Run("TestPolicyStatementMethods_ForResources", func(t *testing.T) {
		ps := s3client.NewPolicyStatement()
		ps.ForResources("mybucket")
		require.Len(t, ps.Resource, 1)
		assert.Equal(t, "arn:aws:s3:::mybucket", ps.Resource[0])
	})

	t.Run("TestPolicyStatementMethods_ForSubResources", func(t *testing.T) {
		ps := s3client.NewPolicyStatement()
		ps.ForSubResources("mybucket")
		require.Len(t, ps.Resource, 1)
		assert.Equal(t, "arn:aws:s3:::mybucket/*", ps.Resource[0])
	})

	t.Run("TestPolicyStatementMethods_AllowsEffect", func(t *testing.T) {
		ps := s3client.NewPolicyStatement()
		ps.Allows()
		assert.Equal(t, s3client.Effect("Allow"), ps.Effect)
		ps.Effect = "Other"
		ps.Allows()
		assert.Equal(t, s3client.Effect("Other"), ps.Effect)
	})

	t.Run("TestPolicyStatementMethods_SetActions", func(t *testing.T) {
		ps := s3client.NewPolicyStatement()
		ps.Actions(s3client.GetBucketAcl, s3client.PutObject)
		assert.ElementsMatch(t, []s3client.Action{s3client.GetBucketAcl, s3client.PutObject}, ps.Action)
	})
}
