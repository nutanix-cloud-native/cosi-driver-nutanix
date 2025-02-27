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

	ntnxIam "github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/util/s3client"
	"k8s.io/klog/v2"
)

func NewDriver(ctx context.Context, provisioner, ntnxEndpoint, accessKey, secretKey,
	pcEndpoint, pcUsername, pcPassword, accountName, s3CaCert, pcCaCert string, s3Insecure, pcInsecure bool) (*IdentityServer, *ProvisionerServer, error) {

	s3Client, err := s3client.NewS3Agent(accessKey, secretKey, ntnxEndpoint, s3CaCert, s3Insecure, true)
	if err != nil {
		errMsg := fmt.Errorf("failed to create S3 client: %w", err)
		klog.Fatalln(errMsg)
	}

	ntnxIamClient, err := ntnxIam.New(ntnxEndpoint, accessKey, secretKey, pcEndpoint, pcUsername, pcPassword, accountName, pcCaCert, pcInsecure, nil)
	if err != nil {
		errMsg := fmt.Errorf("failed to create IAM client: %w", err)
		klog.Fatalln(errMsg)
	}
	return &IdentityServer{
			provisioner: provisioner,
		}, &ProvisionerServer{
			provisioner:   provisioner,
			s3Client:      s3Client,
			ntnxIamClient: ntnxIamClient,
		}, nil
}
