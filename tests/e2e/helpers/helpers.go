package e2e_test_helper

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cosiapi "sigs.k8s.io/container-object-storage-interface-api/apis"
	"sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
	bucketclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned"
)

const (
	_attempts = 30
	sleep     = 2 * time.Second
)

func retry(ctx context.Context, f func() error) (err error) {
	for i := 0; i < _attempts; i++ {
		if i > 0 {
			time.Sleep(sleep)
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return err
}

func VerifyObjectstore(ctx context.Context, ossEndpoint string, s3Client *s3.S3) error {
	err := retry(ctx, func() error {
		var err error

		_, err = http.Get(ossEndpoint)
		if err != nil {
			return err
		}
		_, err = s3Client.ListBuckets(&s3.ListBucketsInput{})
		if err != nil {
			return nil
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func GetBucketClaim(ctx context.Context, bucketClient *bucketclientset.Clientset, bucketClaim *v1alpha1.BucketClaim) (*v1alpha1.BucketClaim, error) {
	err := retry(ctx, func() error {
		var err error

		bucketClaim, err = bucketClient.ObjectstorageV1alpha1().BucketClaims(bucketClaim.Namespace).Get(ctx, bucketClaim.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return bucketClaim, nil
}

func GetBucket(ctx context.Context, bucketClient *bucketclientset.Clientset, bucketClaim *v1alpha1.BucketClaim) (*v1alpha1.Bucket, error) {
	var bucket *v1alpha1.Bucket

	err := retry(ctx, func() error {
		var err error

		bucketClaim, err = bucketClient.ObjectstorageV1alpha1().BucketClaims(bucketClaim.Namespace).Get(ctx, bucketClaim.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if bucketClaim.Status.BucketName == "" {
			return fmt.Errorf("BucketName is empty")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = retry(ctx, func() error {
		var err error

		name := bucketClaim.Status.BucketName

		bucket, err = bucketClient.ObjectstorageV1alpha1().Buckets().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func CheckBucketStatusReady(ctx context.Context, bucketClient *bucketclientset.Clientset, bucketName string) error {
	err := retry(ctx, func() error {
		var err error

		bucket, err := bucketClient.ObjectstorageV1alpha1().Buckets().Get(ctx, bucketName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if bucket.Status.BucketReady == false {
			return fmt.Errorf("bucket is not ready.")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func CheckBucketStatusNotReady(ctx context.Context, bucketClient *bucketclientset.Clientset, bucketName string) error {
	err := retry(ctx, func() error {
		var err error

		bucket, err := bucketClient.ObjectstorageV1alpha1().Buckets().Get(ctx, bucketName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if bucket.Status.BucketReady == true {
			return fmt.Errorf("bucket is ready.")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func CheckBucketExistenceInObjectstore(ctx context.Context, s3Client *s3.S3, bucketName string) error {
	err := retry(ctx, func() error {
		bucketList, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
		if err != nil {
			return err
		}

		for _, bucket := range bucketList.Buckets {
			if *bucket.Name == bucketName {
				return nil
			}
		}

		return fmt.Errorf("bucket does not exist in objectstore")
	})
	if err != nil {
		return err
	}

	return nil
}

func CheckBucketDeletionInObjectstore(ctx context.Context, s3Client *s3.S3, bucketName string) error {
	err := retry(ctx, func() error {
		bucketList, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
		if err != nil {
			return err
		}

		for _, bucket := range bucketList.Buckets {
			if *bucket.Name == bucketName {
				return fmt.Errorf("bucket exists in objectstore")
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func GetBucketAccess(ctx context.Context, bucketClient *bucketclientset.Clientset, bucketAccessName, bucketAccessNamespace string) (*v1alpha1.BucketAccess, error) {
	var bucketAccess *v1alpha1.BucketAccess
	err := retry(ctx, func() error {
		var err error

		bucketAccess, err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(bucketAccessNamespace).Get(ctx, bucketAccessName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if !bucketAccess.Status.AccessGranted {
			return fmt.Errorf("BucketAccess 'accessGranted' is false")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return bucketAccess, nil
}

func CheckBucketAccessNotGranted(ctx context.Context, bucketClient *bucketclientset.Clientset, bucketAccessName, bucketAccessNamespace string) error {
	var bucketAccess *v1alpha1.BucketAccess
	err := retry(ctx, func() error {
		var err error

		bucketAccess, err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(bucketAccessNamespace).Get(ctx, bucketAccessName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if bucketAccess.Status.AccessGranted {
			return fmt.Errorf("BucketAccess 'accessGranted' is true")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func CreateNewS3ClientFromSecret(ctx context.Context, k8sClient *kubernetes.Clientset, secretName, secretNamespace, ossEndpoint string) (*s3.S3, error) {
	secret, err := k8sClient.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(secret).ToNot(BeNil())

	var bucketInfo cosiapi.BucketInfo

	err = json.Unmarshal(secret.Data["BucketInfo"], &bucketInfo)
	Expect(err).ToNot(HaveOccurred())

	sess, err := session.NewSession(
		aws.NewConfig().
			WithRegion("us-east-1").
			WithCredentials(credentials.NewStaticCredentials(bucketInfo.Spec.S3.AccessKeyID, bucketInfo.Spec.S3.AccessSecretKey, "")).
			WithEndpoint(ossEndpoint).
			WithS3ForcePathStyle(true).
			WithMaxRetries(5).
			WithDisableSSL(true).
			WithHTTPClient(
				&http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				},
			).
			WithLogLevel(aws.LogOff),
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(sess).ToNot(BeNil())

	newS3Client := s3.New(sess)
	Expect(newS3Client).ToNot(BeNil())

	return newS3Client, err
}

func checkUserExistsUtil(ctx context.Context, api *admin.API, uuid string) (bool, error) {
	url := api.PCEndpoint + "/oss/iam_proxy/users/" + string(uuid)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create http request. %w", err)
	}

	request.SetBasicAuth(api.PCUsername, api.PCPassword)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return false, nil
	} else if resp.StatusCode == 200 {
		return true, nil
	} else {
		return false, fmt.Errorf("non-200 response: %d", resp.StatusCode)
	}
}

func CheckUserExists(ctx context.Context, api *admin.API, uuid string) (bool, error) {
	err := retry(ctx, func() error {
		var err error

		exists, err := checkUserExistsUtil(ctx, api, uuid)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		return fmt.Errorf("user does not exist")
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func CheckUserDeletion(ctx context.Context, api *admin.API, uuid string) error {
	err := retry(ctx, func() error {
		var err error

		exists, err := checkUserExistsUtil(ctx, api, uuid)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("user exists")
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func GetNumOfUsersInObjectstore(ctx context.Context, api *admin.API) (int, error) {
	var userResp struct {
		Length int `json:"length"`
	}
	url := api.PCEndpoint + "/oss/iam_proxy/users"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to create http request. %w", err)
	}

	request.SetBasicAuth(api.PCUsername, api.PCPassword)
	resp, err := api.HTTPClient.Do(request)
	if err != nil {
		return -1, fmt.Errorf("failed to send http request. %w", err)
	}
	defer resp.Body.Close()
	decodedResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	err = json.Unmarshal(decodedResponse, &userResp)
	if err != nil {
		return -1, err
	}

	return userResp.Length, nil
}
