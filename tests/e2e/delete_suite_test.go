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

var _ = Describe("Delete Bucket", func() {
	var (
		deleteBucketClass *v1alpha1.BucketClass
		failBucketClass   *v1alpha1.BucketClass
		deleteBucketClaim *v1alpha1.BucketClaim
		retainBucketClass *v1alpha1.BucketClass
		retainBucketClaim *v1alpha1.BucketClaim
		failBucketClaim   *v1alpha1.BucketClaim
	)

	BeforeEach(func(ctx context.Context) {
		deleteBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "delete-bucketclass",
			},
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
			DriverName:     "ntnx.objectstorage.k8s.io",
		}

		failBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fail-delete-bucketclass",
			},
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
			DriverName:     "ntnx.objectstorage.k8s.io",
		}

		deleteBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "delete-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "delete-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}

		failBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fail-delete-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "fail-delete-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}

		retainBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "retain-bucketclass",
			},
			DeletionPolicy: v1alpha1.DeletionPolicyRetain,
			DriverName:     "ntnx.objectstorage.k8s.io",
		}

		retainBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "retain-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "retain-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}
	})

	When("DeletionPolicy is set to 'Delete'", func() {
		It("Successfully deletes the bucket from Objectstore backend", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'delete-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, deleteBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, deleteBucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'delete-bucketclaim'")
			deleteBucketClaim, err := bucketClient.ObjectstorageV1alpha1().BucketClaims(deleteBucketClaim.Namespace).Create(ctx, deleteBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket CR is created")
			deleteBucket, err := helpers.GetBucket(ctx, bucketClient, deleteBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleteBucket).ToNot(BeNil())

			// By("Checking if Bucket references the 'delete-bucketclass' and 'delete-bucketclaim'")
			Expect(deleteBucket.Spec.BucketClassName).To(Equal(deleteBucketClass.Name))
			Expect(deleteBucket.Spec.BucketClaim.Name).To(Equal(deleteBucketClaim.Name))

			// By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, deleteBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket is created in the Objectstore backend")
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, deleteBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting BucketClaim resource 'delete-bucketClaim")
			err = bucketClient.ObjectstorageV1alpha1().BucketClaims(deleteBucketClaim.Namespace).Delete(ctx, deleteBucketClaim.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking if Bucket is available in the Objectstore backend")
			err = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, deleteBucket.Name)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("DeletionPolicy is set to 'Retain'", func() {
		It("Does not delete the bucket from Objectstore backend", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'retain-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, retainBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, retainBucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'retain-bucketclaim'")
			retainBucketClaim, err := bucketClient.ObjectstorageV1alpha1().BucketClaims(retainBucketClaim.Namespace).Create(ctx, retainBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket CR is created")
			retainBucket, err := helpers.GetBucket(ctx, bucketClient, retainBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(retainBucket).ToNot(BeNil())

			// By("Checking if Bucket references the 'retain-bucketclass' and 'retain-bucketclaim'")
			Expect(retainBucket.Spec.BucketClassName).To(Equal(retainBucketClass.Name))
			Expect(retainBucket.Spec.BucketClaim.Name).To(Equal(retainBucketClaim.Name))

			// By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, retainBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket is created in the Objectstore backend")
			// err = s3Client.WaitUntilBucketExists(&s3.HeadBucketInput{Bucket: &retainBucket.Name})
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, retainBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting BucketClaim resource 'retain-bucketClaim")
			err = bucketClient.ObjectstorageV1alpha1().BucketClaims(retainBucketClaim.Namespace).Delete(ctx, retainBucketClaim.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking if Bucket is available in the Objectstore backend")
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, retainBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_, _ = s3Client.DeleteBucket(&s3.DeleteBucketInput{Bucket: &retainBucket.Name})
				_ = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, retainBucket.Name)
			})
		})
	})

	When("Bucket does not exist", func() {
		It("Completes execution without throwing error", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'delete-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, failBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, failBucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'delete-bucketclaim'")
			failBucketClaim, err := bucketClient.ObjectstorageV1alpha1().BucketClaims(failBucketClaim.Namespace).Create(ctx, failBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket CR is created")
			failBucket, err := helpers.GetBucket(ctx, bucketClient, failBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(failBucket).ToNot(BeNil())

			// By("Checking if Bucket references the 'delete-bucketclass' and 'delete-bucketclaim'")
			Expect(failBucket.Spec.BucketClassName).To(Equal(failBucketClass.Name))
			Expect(failBucket.Spec.BucketClaim.Name).To(Equal(failBucketClaim.Name))

			// By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket is created in the Objectstore backend")
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting the Bucket from Objectstore")
			_, err = s3Client.DeleteBucket(&s3.DeleteBucketInput{Bucket: &failBucket.Name})
			Expect(err).ToNot(HaveOccurred())

			By("Checking if Bucket is available in the Objectstore backend")
			err = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting BucketClaim resource 'delete-bucketclaim'")
			err = bucketClient.ObjectstorageV1alpha1().BucketClaims(failBucketClaim.Namespace).Delete(ctx, failBucketClaim.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
