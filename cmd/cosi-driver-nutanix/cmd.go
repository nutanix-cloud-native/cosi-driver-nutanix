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

package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	ntnxIam "github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/admin"
	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/driver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"

	"sigs.k8s.io/container-object-storage-interface-provisioner-sidecar/pkg/provisioner"
)

const provisionerName = "ntnx.objectstorage.k8s.io"

var (
	driverAddress = "unix:///var/lib/cosi/cosi.sock"
	AccessKey     = ""
	SecretKey     = ""
	Endpoint      = ""
	PCEndpoint    = ""
	PCUsername    = ""
	PCPassword    = ""
	PCSecret      = ""
	AccountName   = ""
	S3CACert      = ""
	PCCACert      = ""
	S3Insecure    = false
	PCInsecure    = false
)

var cmd = &cobra.Command{
	Use:           "cosi-driver-nutanix",
	Short:         "Kubernetes COSI driver for Nutanix Object Store",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return run(cmd.Context())
	},
	DisableFlagsInUseLine: true,
}

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	flag.Set("alsologtostderr", "true")
	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	persistentFlags := cmd.PersistentFlags()
	persistentFlags.AddGoFlagSet(kflags)

	stringFlag := persistentFlags.StringVarP
	boolFlag := persistentFlags.BoolVarP

	stringFlag(&driverAddress,
		"driver_address",
		"d",
		driverAddress,
		"Path to unix domain socket where driver should listen")

	stringFlag(&Endpoint,
		"endpoint",
		"e",
		Endpoint,
		"Nutanix Object Store instance endpoint")

	stringFlag(&AccessKey,
		"access_key",
		"a",
		AccessKey,
		"Admin IAM Access key to be used for Nutanix Objects")

	stringFlag(&SecretKey,
		"secret_key",
		"s",
		SecretKey,
		"Admin IAM Secret key to be used for Nutanix Objects")

	stringFlag(&PCEndpoint,
		"pc_endpoint",
		"t",
		PCEndpoint,
		"Prism Central Endpoint, eg: https://10.56.192.122:9440")
		
	stringFlag(&PCSecret,
		"pc_secret",
		"k",
		PCSecret,
		"Prism Central Credentials in the format <pc_user>:<pc_password>")

	stringFlag(&AccountName,
		"account_name",
		"u",
		AccountName,
		"User IAM Account Name is an identifier for Nutanix Objects")

	stringFlag(&S3CACert,
		"s3_ca_cert",
		"c",
		S3CACert,
		"S3 CA Certificate in base64 format")

	stringFlag(&PCCACert,
		"pc_ca_cert",
		"p",
		PCCACert,
		"PC CA Certificate in base64 format")

	boolFlag(&S3Insecure,
		"s3_insecure",
		"i",
		S3Insecure,
		"Controls whether certificate chain will be validated for objectstore endpoint (true/false)")

	boolFlag(&PCInsecure,
		"pc_insecure",
		"r",
		PCInsecure,
		"Controls whether certificate chain will be validated for Prism Central endpoint (true/false)")

	viper.BindPFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.PersistentFlags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}

func run(ctx context.Context) error {
	PCUsername, PCPassword, err := ntnxIam.GetCredsFromPCSecret(PCSecret)
	if err != nil {
		errMsg := fmt.Errorf("failed to extract PC credential information from secret: %w", err)
		klog.Error(errMsg)
		return err
	}

	err = ntnxIam.ValidateEndpoint(PCEndpoint)
	if err != nil {
		klog.Error(fmt.Errorf("failed to validate PC endpoint: %w", err))
		return err
	}

	identityServer, bucketProvisioner, err := driver.NewDriver(ctx,
		provisionerName,
		Endpoint,
		AccessKey,
		SecretKey,
		PCEndpoint,
		PCUsername,
		PCPassword,
		AccountName,
		S3CACert,
		PCCACert,
		S3Insecure,
		PCInsecure)
	if err != nil {
		return err
	}

	server, err := provisioner.NewDefaultCOSIProvisionerServer(driverAddress,
		identityServer,
		bucketProvisioner)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}
