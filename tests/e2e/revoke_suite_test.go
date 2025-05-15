//go:build e2e_test

package e2e_test

import (
	"context"

	helpers "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/e2e/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/service/s3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
)

var _ = Describe("Revoke Bucket Access", func() {
	var (
		revokeBucketClass       *v1alpha1.BucketClass
		failBucketClass         *v1alpha1.BucketClass
		revokeBucketClaim       *v1alpha1.BucketClaim
		failBucketClaim         *v1alpha1.BucketClaim
		revokeBucketAccessClass *v1alpha1.BucketAccessClass
		failBucketAccessClass   *v1alpha1.BucketAccessClass
		revokeBucketAccess      *v1alpha1.BucketAccess
		failBucketAccess        *v1alpha1.BucketAccess
		revokeBucket            *v1alpha1.Bucket
		failBucket              *v1alpha1.Bucket
	)

	BeforeEach(func(ctx context.Context) {
		revokeBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "revoke-bucketclass",
			},
			DriverName:     "ntnx.objectstorage.k8s.io",
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
		}

		failBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fail-revoke-bucketclass",
			},
			DriverName:     "ntnx.objectstorage.k8s.io",
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
		}

		revokeBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "revoke-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "revoke-bucketclass",
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
				Name:      "fail-revoke-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "fail-revoke-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}

		revokeBucketAccessClass = &v1alpha1.BucketAccessClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccessClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "revoke-bucketaccessclass",
			},
			DriverName:         "ntnx.objectstorage.k8s.io",
			AuthenticationType: v1alpha1.AuthenticationTypeKey,
		}

		failBucketAccessClass = &v1alpha1.BucketAccessClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccessClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fail-revoke-bucketaccessclass",
			},
			DriverName:         "ntnx.objectstorage.k8s.io",
			AuthenticationType: v1alpha1.AuthenticationTypeKey,
		}

		revokeBucketAccess = &v1alpha1.BucketAccess{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccess",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "revoke-bucketaccess",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketAccessSpec{
				BucketAccessClassName: "revoke-bucketaccessclass",
				BucketClaimName:       "revoke-bucketclaim",
				CredentialsSecretName: "revoke-bucketcredentials",
			},
		}

		failBucketAccess = &v1alpha1.BucketAccess{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccess",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fail-revoke-bucketaccess",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketAccessSpec{
				BucketAccessClassName: "fail-revoke-bucketaccessclass",
				BucketClaimName:       "fail-revoke-bucketclaim",
				CredentialsSecretName: "fail-revoke-bucketcredentials",
			},
		}
	})

	When("User exists", func() {
		It("Successfully revokes user access to bucket", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'revoke-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, revokeBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, revokeBucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'revoke-bucketclaim'")
			revokeBucketClaim, err = bucketClient.ObjectstorageV1alpha1().BucketClaims(revokeBucketClaim.Namespace).Create(ctx, revokeBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClaims(revokeBucketClaim.Namespace).Delete(ctx, revokeBucketClaim.Name, metav1.DeleteOptions{})
				_ = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, revokeBucket.Name)
			})

			// By("Checking if Bucket CR is created")
			revokeBucket, err = helpers.GetBucket(ctx, bucketClient, revokeBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(revokeBucket).ToNot(BeNil())

			// By("Checking if Bucket references the 'delete-bucketclass' and 'delete-bucketclaim'")
			Expect(revokeBucket.Spec.BucketClassName).To(Equal(revokeBucketClass.Name))
			Expect(revokeBucket.Spec.BucketClaim.Name).To(Equal(revokeBucketClaim.Name))

			// By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, revokeBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket is created in the Objectstore backend")
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, revokeBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Creating a BucketAccessClass resource from 'revoke-bucketaccessclass'")
			revokeBucketAccessClass, err = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Create(ctx, revokeBucketAccessClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Delete(ctx, revokeBucketAccessClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketAccess resource from 'revoke-bucketaccess'")
			revokeBucketAccess, err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Create(ctx, revokeBucketAccess, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if BucketAccess status 'accessGranted' is 'true'")
			revokeBucketAccess, err = helpers.GetBucketAccess(ctx, bucketClient, revokeBucketAccess.Name, revokeBucketAccess.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(revokeBucketAccess).ToNot(BeNil())

			// By("Checking if BucketAccess status 'accountID' is not empty")
			Expect(revokeBucketAccess.Status.AccountID).ToNot(Or(BeEmpty(), BeNil()))

			// By("Checking if a new user is created in Objectstore")
			exists, err := helpers.CheckUserExists(ctx, iamClient, revokeBucketAccess.Status.AccountID)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			By("Deleting BucketAccess resource 'revoke-bucketaccess")
			err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Delete(ctx, revokeBucketAccess.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking if user created by resource 'revoke-bucketaccess' is deleted")
			err = helpers.CheckUserDeletion(ctx, iamClient, revokeBucketAccess.Status.AccountID)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("User does not exist", func() {
		It("Completes execution without throwing error", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'revoke-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, failBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Delete(ctx, failBucketAccessClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'fail-revoke-bucketclaim'")
			failBucketClaim, err = bucketClient.ObjectstorageV1alpha1().BucketClaims(failBucketClaim.Namespace).Create(ctx, failBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClaims(failBucketClaim.Namespace).Delete(ctx, failBucketClaim.Name, metav1.DeleteOptions{})
				_ = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, failBucket.Name)
			})

			// By("Checking if Bucket CR is created")
			failBucket, err = helpers.GetBucket(ctx, bucketClient, failBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(failBucket).ToNot(BeNil())

			// By("Checking if Bucket references the 'delete-bucketclass' and 'delete-bucketclaim'")
			Expect(failBucket.Spec.BucketClassName).To(Equal(failBucketClass.Name))
			Expect(failBucket.Spec.BucketClaim.Name).To(Equal(failBucketClaim.Name))

			// By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket is created in the Objectstore backend")
			err = s3Client.WaitUntilBucketExists(&s3.HeadBucketInput{Bucket: &failBucket.Name})
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Creating a BucketAccessClass resource from 'revoke-bucketaccessclass'")
			failBucketAccessClass, err = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Create(ctx, failBucketAccessClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Delete(ctx, failBucketAccessClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketAccess resource from 'fail-revoke-bucketaccess'")
			failBucketAccess, err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Create(ctx, failBucketAccess, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if BucketAccess status 'accessGranted' is 'true'")
			failBucketAccess, err = helpers.GetBucketAccess(ctx, bucketClient, failBucketAccess.Name, failBucketAccess.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(failBucketAccess).ToNot(BeNil())

			// By("Checking if BucketAccess status 'accountID' is not empty")
			Expect(failBucketAccess.Status.AccountID).ToNot(Or(BeEmpty(), BeNil()))

			// By("Checking if a new user is created in Objectstore")
			exists, err := helpers.CheckUserExists(ctx, iamClient, failBucketAccess.Status.AccountID)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			By("Deleting user created by BucketAccess resource 'fail-revoke-bucketaccess'")
			err = iamClient.RemoveUser(ctx, failBucketAccess.Status.AccountID)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting BucketAccess resource 'fail-revoke-bucketaccess")
			err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Delete(ctx, failBucketAccess.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
