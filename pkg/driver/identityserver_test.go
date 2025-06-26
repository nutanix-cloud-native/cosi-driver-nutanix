package driver_test

import (
	"context"
	"testing"

	"github.com/nutanix-core/k8s-ntnx-object-cosi/pkg/driver"
	"github.com/stretchr/testify/assert"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

func TestDriverGetInfo(t *testing.T) {
	t.Run("TestDriverGetInfo_ValidServer", func(t *testing.T) {
		srv := &driver.IdentityServer{
			Provisioner: "dummy",
		}

		res, err := srv.DriverGetInfo(context.Background(), &cosi.DriverGetInfoRequest{})
		assert.NoError(t, err)
		assert.Equal(t, "dummy", res.Name)
	})

	t.Run("TestDriverGetInfo_EmptyProvisionerName", func(t *testing.T) {
		srv := &driver.IdentityServer{}
		res, err := srv.DriverGetInfo(context.Background(), &cosi.DriverGetInfoRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
