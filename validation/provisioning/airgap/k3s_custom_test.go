//go:build validation || (recurring && airgap) || airgap

package airgap

import (
	"os"
	"testing"

	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	steveV1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/provisioninginput"
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

func TestCustomK3SAirgap(t *testing.T) {
	r := airgapSetup(t, defaults.K3S)

	nodeRolesStandard := []provisioninginput.MachinePools{
		provisioninginput.AllRolesMachinePool,
	}

	nodeRolesStandard[0].MachinePoolConfig.Quantity = 1

	clusterConfig := new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, r.cattleConfig, clusterConfig)

	tests := []struct {
		name         string
		client       *rancher.Client
		machinePools []provisioninginput.MachinePools
		nodePools    []tfpConfig.Nodepool
	}{
		{"K3S_Airgap_Custom", r.standardUserClient, nodeRolesStandard, []tfpConfig.Nodepool{{Quantity: 1, Etcd: true, Controlplane: true, Worker: true}}},
	}

	for _, tt := range tests {
		t.Cleanup(func() {
			logrus.Infof("Running cleanup (%s)", tt.name)
			r.session.Cleanup()

			if r.tunnel != nil {
				r.tunnel.StopBastionSSHTunnel()
			}
		})

		clusterConfig := new(clusters.ClusterConfig)
		operations.LoadObjectFromMap(defaults.ClusterConfigKey, r.cattleConfig, clusterConfig)

		clusterConfig.MachinePools = tt.machinePools
		rancherConfig, terraformConfig, terratestConfig, _ := tfpConfig.LoadTFPConfigs(r.cattleConfig)
		terratestConfig.Nodepools = tt.nodePools

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logrus.Info("Provisioning cluster")
			nestedRancherModuleDir, perTestTerraformOptions, keyPath, cluster := tfpCustom.CreateCustomCluster(t, tt.client, rancherConfig, terraformConfig, terratestConfig, defaults.K3S, "validation/provisioning/airgap")
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
