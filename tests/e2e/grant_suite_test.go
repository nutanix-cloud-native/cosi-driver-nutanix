//go:build e2e_test

package e2e_test

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	helpers "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/e2e/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
)

var _ = Describe("Grant Bucket Access", func() {
	var (
		grantBucketClass       *v1alpha1.BucketClass
		grantBucketClaim       *v1alpha1.BucketClaim
		grantBucketAccessClass *v1alpha1.BucketAccessClass
		failBucketClass        *v1alpha1.BucketClass
		failBucketClaim        *v1alpha1.BucketClaim
		failBucketAccessClass  *v1alpha1.BucketAccessClass
		grantBucketAccess      *v1alpha1.BucketAccess
		failBucketAccess       *v1alpha1.BucketAccess
		grantBucket            *v1alpha1.Bucket
		failBucket             *v1alpha1.Bucket
	)

	BeforeEach(func(ctx context.Context) {
		grantBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "grant-bucketclass",
			},
			DriverName:     "ntnx.objectstorage.k8s.io",
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
		}

		grantBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "grant-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "grant-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}

		grantBucketAccessClass = &v1alpha1.BucketAccessClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccessClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "grant-bucketaccessclass",
			},
			DriverName:         "ntnx.objectstorage.k8s.io",
			AuthenticationType: v1alpha1.AuthenticationTypeKey,
		}

		failBucketClass = &v1alpha1.BucketClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fail-grant-bucketclass",
			},
			DriverName:     "ntnx.objectstorage.k8s.io",
			DeletionPolicy: v1alpha1.DeletionPolicyDelete,
		}

		failBucketClaim = &v1alpha1.BucketClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketClaim",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fail0-grant-bucketclaim",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketClaimSpec{
				BucketClassName: "fail-grant-bucketclass",
				Protocols: []v1alpha1.Protocol{
					v1alpha1.ProtocolS3,
				},
			},
		}

		failBucketAccessClass = &v1alpha1.BucketAccessClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccessClass",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "fail-grant-bucketaccessclass",
			},
			DriverName:         "ntnx.objectstorage.k8s.io",
			AuthenticationType: v1alpha1.AuthenticationTypeKey,
		}

		grantBucketAccess = &v1alpha1.BucketAccess{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccess",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "grant-bucketaccess",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketAccessSpec{
				BucketAccessClassName: "grant-bucketaccessclass",
				BucketClaimName:       "grant-bucketclaim",
				CredentialsSecretName: "grant-bucketcredentials",
			},
		}

		failBucketAccess = &v1alpha1.BucketAccess{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BucketAccess",
				APIVersion: "objectstorage.k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fail-grant-bucketaccess",
				Namespace: namespace,
			},
			Spec: v1alpha1.BucketAccessSpec{
				BucketAccessClassName: "fail-grant-bucketaccessclass",
				BucketClaimName:       "fail-grant-bucketclaim",
				CredentialsSecretName: "fail-grant-bucketcredentials",
			},
		}
	})

	When("Bucket exists", func() {
		It("Successfully creates user and grants bucket access", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'grant-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, grantBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, grantBucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'grant-bucketclaim'")
			grantBucketClaim, err = bucketClient.ObjectstorageV1alpha1().BucketClaims(grantBucketClaim.Namespace).Create(ctx, grantBucketClaim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClaims(grantBucketClaim.Namespace).Delete(ctx, grantBucketClaim.Name, metav1.DeleteOptions{})
				_ = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, grantBucket.Name)
			})

			// By("Checking if Bucket CR is created")
			grantBucket, err = helpers.GetBucket(ctx, bucketClient, grantBucketClaim)
			Expect(err).ToNot(HaveOccurred())
			Expect(grantBucket).ToNot(BeNil())

			// By("Checking if Bucket references the 'delete-bucketclass' and 'delete-bucketclaim'")
			Expect(grantBucket.Spec.BucketClassName).To(Equal(grantBucketClass.Name))
			Expect(grantBucket.Spec.BucketClaim.Name).To(Equal(grantBucketClaim.Name))

			// By("Checking if Bucket status is 'bucketReady'")
			err = helpers.CheckBucketStatusReady(ctx, bucketClient, grantBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			// By("Checking if Bucket is created in the Objectstore backend")
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, grantBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Creating a BucketAccessClass resource from 'grant-bucketaccessclass'")
			grantBucketAccessClass, err = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Create(ctx, grantBucketAccessClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Delete(ctx, grantBucketAccessClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketAccess resource from 'grant-bucketaccess'")
			grantBucketAccess, err := bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Create(ctx, grantBucketAccess, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Delete(ctx, grantBucketAccess.Name, metav1.DeleteOptions{})
				_ = helpers.CheckUserDeletion(ctx, iamClient, grantBucketAccess.Status.AccountID)
			})

			By("Checking if BucketAccess status 'accessGranted' is 'true'")
			grantBucketAccess, err = helpers.GetBucketAccess(ctx, bucketClient, grantBucketAccess.Name, grantBucketAccess.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(grantBucketAccess).ToNot(BeNil())

			By("Checking if BucketAccess status 'accountID' is not empty")
			Expect(grantBucketAccess.Status.AccountID).ToNot(Or(BeEmpty(), BeNil()))

			By("Checking if a new user is created in Objectstore")
			exists, err := helpers.CheckUserExists(ctx, iamClient, grantBucketAccess.Status.AccountID)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			By("Checking if secret 'grant-bucketcredentials' is created")
			secret, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, grantBucketAccess.Spec.CredentialsSecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(secret).ToNot(BeNil())
			Expect(secret.Data).ToNot(Or(BeNil(), BeEmpty()))

			By("Checking S3 ops using secret 'grant-bucketcredentials'")
			newS3Client, err := helpers.CreateNewS3ClientFromSecret(ctx, k8sClient, secret.Name, secret.Namespace, ossEndpoint)
			Expect(err).ToNot(HaveOccurred())
			Expect(newS3Client).ToNot(BeNil())

			_, err = newS3Client.PutObject(&s3.PutObjectInput{
				Body:   strings.NewReader("Test Object"),
				Bucket: aws.String(grantBucket.Name),
				Key:    aws.String("test.txt"),
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = newS3Client.DeleteObject(&s3.DeleteObjectInput{
				Bucket: aws.String(grantBucket.Name),
				Key:    aws.String("test.txt"),
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("Bucket does not exist", func() {
		It("Fails to get bucket details and user is not created", func(ctx context.Context) {
			By("Creating a BucketClass resource from 'fail-grant-bucketclass")
			_, err := bucketClient.ObjectstorageV1alpha1().BucketClasses().Create(ctx, failBucketClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketClasses().Delete(ctx, failBucketClass.Name, metav1.DeleteOptions{})
			})

			By("Creating a BucketClaim resource from 'fail-grant-bucketclaim'")
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
			err = helpers.CheckBucketExistenceInObjectstore(ctx, s3Client, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Creating a BucketAccessClass resource from 'fail-grant-bucketaccessclass'")
			failBucketAccessClass, err = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Create(ctx, failBucketAccessClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccessClasses().Delete(ctx, failBucketAccessClass.Name, metav1.DeleteOptions{})
			})

			initNumOfUsers, err := helpers.GetNumOfUsersInObjectstore(ctx, iamClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(initNumOfUsers).ToNot(Equal(-1))

			By("Deleting bucket from objectstore")
			_, err = s3Client.DeleteBucket(&s3.DeleteBucketInput{Bucket: &failBucket.Name})
			Expect(err).ToNot(HaveOccurred())

			By("Checking if bucket is deleted in the objectstore backend")
			err = helpers.CheckBucketDeletionInObjectstore(ctx, s3Client, failBucket.Name)
			Expect(err).ToNot(HaveOccurred())

			By("Creating a BucketAccess resource from 'fail-grant-bucketaccess'")
			failBucketAccess, err = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Create(ctx, failBucketAccess, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func(ctx context.Context) {
				_ = bucketClient.ObjectstorageV1alpha1().BucketAccesses(namespace).Delete(ctx, failBucketAccess.Name, metav1.DeleteOptions{})
				_ = helpers.CheckUserDeletion(ctx, iamClient, failBucketAccess.Status.AccountID)
			})

			By("Checking status of bucketAccess resource 'fail-grant-bucketaccess")
			err = helpers.CheckBucketAccessNotGranted(ctx, bucketClient, failBucketAccess.Name, failBucketAccess.Namespace)
			Expect(err).ToNot(HaveOccurred())

			By("Checking if user is created in Objectstore")
			newNumOfUsers, err := helpers.GetNumOfUsersInObjectstore(ctx, iamClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(newNumOfUsers).To(Equal(initNumOfUsers))

			By("Checking if secret 'fail-grant-bucketcredentials' is created")
			_, err = k8sClient.CoreV1().Secrets(namespace).Get(ctx, failBucketAccess.Spec.CredentialsSecretName, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})
	})

})
