//go:build validation || (recurring && airgap) || airgap

package airgap

import (
	"os"
	"testing"

	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	steveV1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/qase"
	"github.com/rancher/tests/actions/registries"
	"github.com/rancher/tests/actions/workloads/deployment"
	"github.com/rancher/tests/actions/workloads/pods"
	tfpConfig "github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/framework/cleanup"
	tfpCustom "github.com/rancher/tfp-automation/tests/infrastructure/downstream/custom"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestCustomRKE2Airgap(t *testing.T) {
	r := airgapSetup(t, defaults.RKE2)
	nodeRolesStandard := []tfpConfig.Nodepool{{Quantity: 1, Etcd: true}, {Quantity: 1, Controlplane: true}, {Quantity: 1, Worker: true}}
	nodeRolesDedicatedWindows := []tfpConfig.Nodepool{{Quantity: 1, Etcd: true}, {Quantity: 1, Controlplane: true}, {Quantity: 1, Worker: true}, {Quantity: 1, Windows: true}}

	tests := []struct {
		name        string
		client      *rancher.Client
		clusterType string
		nodePools   []tfpConfig.Nodepool
	}{
		{
			"RKE2_Airgap_Custom",
			r.standardUserClient,
			defaults.RKE2,
			nodeRolesStandard,
		},
		{
			"RKE2_Airgap_Custom_Windows",
			r.standardUserClient,
			"rke2_windows_2022",
			nodeRolesDedicatedWindows,
		},
	}

	for _, tt := range tests {
		t.Cleanup(func() {
			logrus.Infof("Running cleanup (%s)", tt.name)
			r.session.Cleanup()

			if r.tunnel != nil {
				r.tunnel.StopBastionSSHTunnel()
			}
		})

		rancherConfig, terraformConfig, terratestConfig, _ := tfpConfig.LoadTFPConfigs(r.cattleConfig)
		terratestConfig.Nodepools = tt.nodePools

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logrus.Info("Provisioning cluster")
			nestedRancherModuleDir, perTestTerraformOptions, keyPath, cluster := tfpCustom.CreateCustomCluster(t, tt.client, rancherConfig, terraformConfig, terratestConfig, tt.clusterType, "validation/provisioning/airgap")
			defer os.RemoveAll(nestedRancherModuleDir)
			defer cleanup.Cleanup(t, perTestTerraformOptions, keyPath)

			logrus.Infof("Verifying the cluster is ready (%s)", cluster.Name)
			var err error
			err = provisioning.VerifyClusterReady(r.client, cluster)
			require.NoError(t, err)

			logrus.Infof("Verifying cluster deployments (%s)", cluster.Name)
			err = deployment.VerifyClusterDeployments(r.client, cluster)
			require.NoError(t, err)

			logrus.Infof("Verifying cluster pods (%s)", cluster.Name)
			err = pods.VerifyClusterPods(r.client, cluster)
			require.NoError(t, err)

			clusterStatus := &provv1.ClusterStatus{}
			err = steveV1.ConvertToK8sType(cluster.Status, clusterStatus)
			require.NoError(t, err)

			logrus.Infof("Verifying cluster pods use private registry")
			_, err = registries.CheckAllClusterPodsForRegistryPrefix(tt.client, clusterStatus.ClusterName, r.terraformConfig.PrivateRegistries.SystemDefaultRegistry)
			require.NoError(t, err)
		})

		params := provisioning.GetCustomSchemaParams(tt.client, r.cattleConfig)
		err := qase.UpdateSchemaParameters(tt.name, params)
		if err != nil {
			logrus.Warningf("Failed to upload schema parameters %s", err)
		}
	}
}
