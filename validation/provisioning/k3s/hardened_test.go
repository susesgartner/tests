//go:build validation || recurring

package k3s

import (
	"os"
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	extensionscluster "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/charts"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/logging"
	"github.com/rancher/tests/actions/projects"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/qase"
	"github.com/rancher/tests/actions/workloads/deployment"
	"github.com/rancher/tests/actions/workloads/pods"
	compliance "github.com/rancher/tests/validation/provisioning/resources/rancherCompliance"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	tfpConfig "github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/framework/cleanup"
	tfpCustom "github.com/rancher/tfp-automation/tests/infrastructure/downstream/custom"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type hardenedTest struct {
	client              *rancher.Client
	session             *session.Session
	standardUserClient  *rancher.Client
	cattleConfig        map[string]any
	chartInstallOptions *charts.InstallOptions
}

func hardenedSetup(t *testing.T) hardenedTest {
	var k hardenedTest
	testSession := session.NewSession()
	k.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(t, err)

	k.client = client

	k.cattleConfig = config.LoadConfigFromFile(os.Getenv(config.ConfigEnvironmentKey))

	k.cattleConfig, err = defaults.LoadPackageDefaults(k.cattleConfig, "")
	require.NoError(t, err)

	loggingConfig := new(logging.Logging)
	operations.LoadObjectFromMap(logging.LoggingKey, k.cattleConfig, loggingConfig)

	err = logging.SetLogger(loggingConfig)
	require.NoError(t, err)

	k.cattleConfig, err = defaults.SetK8sDefault(client, defaults.K3S, k.cattleConfig)
	require.NoError(t, err)

	k.standardUserClient, _, _, err = standard.CreateStandardUser(k.client)
	require.NoError(t, err)

	return k
}

func TestHardened(t *testing.T) {
	t.Parallel()
	var err error
	k := hardenedSetup(t)
	nodeRolesStandard := []tfpConfig.Nodepool{{Quantity: 3, Etcd: true}, {Quantity: 2, Controlplane: true}, {Quantity: 3, Worker: true}}

	tests := []struct {
		name            string
		client          *rancher.Client
		scanProfileName string
	}{
		{"K3S_Rancher_Compliance", k.standardUserClient, "k3s-cis-1.11-profile"},
	}
	for _, tt := range tests {
		t.Cleanup(func() {
			logrus.Infof("Running cleanup (%s)", tt.name)
			k.session.Cleanup()
		})

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rancherConfig, terraformConfig, terratestConfig, _ := tfpConfig.LoadTFPConfigs(k.cattleConfig)
			terratestConfig.Nodepools = nodeRolesStandard

			logrus.Info("Provisioning custom cluster")
			nestedRancherModuleDir, perTestTerraformOptions, keyPath, cluster := tfpCustom.CreateCustomCluster(t, tt.client, rancherConfig, terraformConfig, terratestConfig, defaults.K3S, "validation/provisioning/k3s")
			defer os.RemoveAll(nestedRancherModuleDir)
			defer cleanup.Cleanup(t, perTestTerraformOptions, keyPath)

			logrus.Infof("Verifying the cluster is ready (%s)", cluster.Name)
			err = provisioning.VerifyClusterReady(k.client, cluster)
			require.NoError(t, err)

			logrus.Infof("Verifying cluster deployments (%s)", cluster.Name)
			err = deployment.VerifyClusterDeployments(k.client, cluster)
			require.NoError(t, err)

			logrus.Infof("Verifying cluster pods (%s)", cluster.Name)
			err = pods.VerifyClusterPods(k.client, cluster)
			require.NoError(t, err)

			logrus.Infof("Verifying service account token secret (%s)", cluster.Name)
			err = clusters.VerifyServiceAccountTokenSecret(k.client, cluster.Name)
			require.NoError(t, err)

			chartName := charts.ComplianceName
			chartNamespace := charts.ComplianceNamespace

			clusterMeta, err := extensionscluster.NewClusterMeta(tt.client, cluster.Name)
			require.NoError(t, err)

			latestRancherComplianceVersion, err := tt.client.Catalog.GetLatestChartVersion(chartName, catalog.RancherChartRepo)
			require.NoError(t, err)

			project, err := projects.GetProjectByName(tt.client, clusterMeta.ID, compliance.System)
			require.NoError(t, err)

			k.chartInstallOptions = &charts.InstallOptions{
				Cluster:   clusterMeta,
				Version:   latestRancherComplianceVersion,
				ProjectID: project.ID,
			}

			logrus.Infof("Project cluster id: %s", project.ClusterID)

			compliance.SetupComplianceChart(tt.client, project.ClusterID, k.chartInstallOptions, chartName, chartNamespace)
			compliance.RunRancherComplianceScan(tt.client, project.ClusterID, tt.scanProfileName)
		})

		params := provisioning.GetCustomSchemaParams(tt.client, k.cattleConfig)
		err := qase.UpdateSchemaParameters(tt.name, params)
		if err != nil {
			logrus.Warningf("Failed to upload schema parameters %s", err)
		}
	}
}
