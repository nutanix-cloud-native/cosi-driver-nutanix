//go:build e2e_test

package e2e_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	helpers "github.com/nutanix-core/k8s-ntnx-object-cosi/tests/e2e/helpers"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	bucketclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned"
)

var (
	k8sClient    *kubernetes.Clientset
	bucketClient *bucketclientset.Clientset
	s3Client     *s3.S3
	iamClient    *admin.API

	namespace     string
	ossEndpoint   string
	prismEndpoint string
	prismUsername string
	prismPassword string
	accessKey     string
	secretKey     string
	nodeIP        string
)

const (
	deploymentName = "objectstorage-provisioner"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func(ctx context.Context) {

	By("Extracting environment variables")

	kubeConfig, exists := os.LookupEnv("KUBECONFIG")
	Expect(exists).To(BeTrue())

	namespace, exists = os.LookupEnv("DRIVER_NAMESPACE")
	Expect(exists).To(BeTrue())

	ossEndpoint, exists = os.LookupEnv("OSS_ENDPOINT")
	Expect(exists).To(BeTrue())

	prismEndpoint, exists = os.LookupEnv("PC_ENDPOINT")
	Expect(exists).To(BeTrue())

	prismUsername, exists = os.LookupEnv("PC_USERNAME")
	Expect(exists).To(BeTrue())

	prismPassword, exists = os.LookupEnv("PC_PASSWORD")
	Expect(exists).To(BeTrue())

	accessKey, exists = os.LookupEnv("ACCESS_KEY")
	Expect(exists).To(BeTrue())

	secretKey, exists = os.LookupEnv("SECRET_KEY")
	Expect(exists).To(BeTrue())

	useTriton, _ := os.LookupEnv("USE_TRITON")
	if useTriton != "" {
		nodeIP, exists = os.LookupEnv("NODE_IP")
		Expect(exists).To(BeTrue())

		ossEndpoint = "http://" + nodeIP + ":30720"
		prismEndpoint = "http://" + nodeIP + ":30556"
	}

	By("Building kubernetes client")
	testConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	Expect(err).ToNot(HaveOccurred())
	k8sClient, err = kubernetes.NewForConfig(testConfig)
	Expect(err).ToNot(HaveOccurred())

	By("Building COSI buckets client")
	bucketClient, err = bucketclientset.NewForConfig(testConfig)
	Expect(err).ToNot(HaveOccurred())

	By("Building S3 client")
	sess, err := session.NewSession(
		aws.NewConfig().
			WithRegion("us-east-1").
			WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
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
	s3Client = s3.New(sess)
	Expect(s3Client).ToNot(BeNil())

	By("Building IAM client")
	iamClient = &admin.API{
		Endpoint:    ossEndpoint,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		PCEndpoint:  prismEndpoint,
		PCUsername:  prismUsername,
		PCPassword:  prismPassword,
		AccountName: "cosi-test",
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
	Expect(iamClient).ToNot(BeNil())

	By("Checking cluster availability")
	value, err := k8sClient.ServerVersion()
	Expect(err).ToNot(HaveOccurred())
	Expect(value).ToNot(BeNil())

	By("Checking COSI namespace")
	_, err = k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Checking COSI installation")
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(deployment.Status.Conditions).To(ContainElement(HaveField("Type", Equal(appsv1.DeploymentAvailable))))

	By("Checking objectstore existence")
	err = helpers.VerifyObjectstore(ctx, ossEndpoint, s3Client)
	Expect(err).ToNot(HaveOccurred())

})
