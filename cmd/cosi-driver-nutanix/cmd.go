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
)

var cmd = &cobra.Command{
	Use:           "cosi-driver-nutanix",
	Short:         "Kubernetes COSI driver for Nutanix Object Store",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd.Context(), args)
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

	stringFlag(&PCSecret,
		"pc_secret",
		"k",
		PCSecret,
		"Base64 encoded format of <prism-ip>:<prism-port>:<pc_user>:<pc_password>")

	stringFlag(&AccountName,
		"account_name",
		"u",
		AccountName,
		"User IAM Account Name is an identifier for Nutanix Objects")

	viper.BindPFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.PersistentFlags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}

func run(ctx context.Context, args []string) error {
	PCEndpoint, PCUsername, PCPassword, err := ntnxIam.GetCredsFromPCSecret(PCSecret)
	klog.InfoS(PCEndpoint, PCUsername, PCPassword, err)
	if err != nil {
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
		AccountName)
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
