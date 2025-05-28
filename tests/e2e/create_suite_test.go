//go:build e2e_test

package e2e_test

import (
	"context"

	"github.com/aws/aws-sdk-go/service/s3"
	helpers "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/e2e/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
)

var _ = Describe("Create Bucket", func() {
	var (
		bucketClass        *v1alpha1.BucketClass
		bucketClaim        *v1alpha1.BucketClaim
		invalidBucketClaim *v1alpha1.BucketClaim
		bucket             *v1alpha1.Bucket
	)

	BeforeEach(func(ctx context.Context) {
		bucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "create-bucketclass",
			},
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
			DriverName:     "ntnx.objectstorage.k8s.io",
		}

		bucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "create-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "create-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}

		invalidBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-create-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "invalid-create-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}
	})

	When("Valid BucketClaim is used", func() {
		It("Successfully creates a bucket", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'create-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, bucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, bucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'create-bucketclaim'")
			bucketClaim, err := bucketClient.ObjectstorageV1alpha1().BucketClaims(bucketClaim.Namespace).Create(ctx, bucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClaims(bucketClaim.Namespace).Delete(ctx, bucketClaim.Name, metav1.DeleteOptions{})
				err = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, bucket.Name)
			})

			By("Checking if Bucket CR is created")
			bucket, err = helpers.GetBucket(ctx, bucketClient, bucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(bucket).ToNot(BeNil())

			By("Checking if Bucket references the 'create-bucketclass' and 'create-bucketclaim'")
			Expect(bucket.Spec.BucketClassName).To(Equal(bucketClass.Name))
			Expect(bucket.Spec.BucketClaim.Name).To(Equal(bucketClaim.Name))

			By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, bucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Checking if Bucket is created in the Objectstore backend")
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, bucket.Name)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("Invalid BucketClaim is used", func() {
		It("Fails to create a bucket", func(ctx context.Context) {
			By("Counting number of Buckets in the Objectstore backend before running the test")
			out, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
			Expect(err).ToNot(HaveOccurred())
			oldBucketsCount := len(out.Buckets)

			By("Creating a BucketClaim resource from 'invalid-create-bucketclaim'")
			invalidBucketClaim, err := bucketClient.ObjectstorageV1alpha1().BucketClaims(invalidBucketClaim.Namespace).Create(ctx, invalidBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClaims(invalidBucketClaim.Namespace).Delete(ctx, invalidBucketClaim.Name, metav1.DeleteOptions{})
			})

			By("Checking if status of BucketClaim is empty")
			invalidBucketClaim, err = helpers.GetBucketClaim(ctx, bucketClient, invalidBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(invalidBucketClaim.Status.BucketName).To(BeEmpty())

			By("Checking if Bucket is not created in the Objectstore backend")
			out, err = s3Client.ListBuckets(&s3.ListBucketsInput{})
			Expect(err).ToNot(HaveOccurred())
			bucketsCount := len(out.Buckets)
			Expect(bucketsCount).To(BeNumerically("<=", oldBucketsCount))
		})
	})
})
