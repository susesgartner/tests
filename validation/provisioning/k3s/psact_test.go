//go:build validation

package k3s

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/extensions/users"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type K3SPSACTTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	standardUserClient *rancher.Client
	provisioningConfig *provisioninginput.Config
}

func (k *K3SPSACTTestSuite) TearDownSuite() {
	k.session.Cleanup()
}

func (k *K3SPSACTTestSuite) SetupSuite() {
	testSession := session.NewSession()
	k.session = testSession

	k.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, k.provisioningConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(k.T(), err)

	k.client = client

	if k.provisioningConfig.K3SKubernetesVersions == nil {
		k3sVersions, err := kubernetesversions.Default(k.client, clusters.K3SClusterType.String(), nil)
		require.NoError(k.T(), err)

		k.provisioningConfig.K3SKubernetesVersions = k3sVersions
	} else if k.provisioningConfig.K3SKubernetesVersions[0] == "all" {
		k3sVersions, err := kubernetesversions.ListK3SAllVersions(k.client)
		require.NoError(k.T(), err)

		k.provisioningConfig.K3SKubernetesVersions = k3sVersions
	}

	enabled := true
	var testuser = namegen.AppendRandomString("testuser-")
	var testpassword = password.GenerateUserPassword("testpass-")
	user := &management.User{
		Username: testuser,
		Password: testpassword,
		Name:     testuser,
		Enabled:  &enabled,
	}

	newUser, err := users.CreateUserWithRole(client, user, "user")
	require.NoError(k.T(), err)

	newUser.Password = user.Password

	standardUserClient, err := client.AsUser(newUser)
	require.NoError(k.T(), err)

	k.standardUserClient = standardUserClient
}

func (k *K3SPSACTTestSuite) TestK3SPSACTNodeDriverCluster() {
	nodeRolesStandard := []provisioninginput.MachinePools{provisioninginput.EtcdMachinePool, provisioninginput.ControlPlaneMachinePool, provisioninginput.WorkerMachinePool}

	nodeRolesStandard[0].MachinePoolConfig.Quantity = 3
	nodeRolesStandard[1].MachinePoolConfig.Quantity = 2
	nodeRolesStandard[2].MachinePoolConfig.Quantity = 3

	tests := []struct {
		name         string
		machinePools []provisioninginput.MachinePools
		psact        provisioninginput.PSACT
		client       *rancher.Client
	}{
		{"Rancher Privileged " + provisioninginput.AdminClientName.String(), nodeRolesStandard, "rancher-privileged", k.client},
		{"Rancher Restricted " + provisioninginput.AdminClientName.String(), nodeRolesStandard, "rancher-restricted", k.client},
		{"Rancher Baseline " + provisioninginput.AdminClientName.String(), nodeRolesStandard, "rancher-baseline", k.client},
	}

	for _, tt := range tests {
		provisioningConfig := *k.provisioningConfig
		provisioningConfig.MachinePools = tt.machinePools
		provisioningConfig.PSACT = string(tt.psact)

		permutations.RunTestPermutations(&k.Suite, tt.name, tt.client, &provisioningConfig, permutations.K3SProvisionCluster, nil, nil)
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestK3SPSACTTestSuite(t *testing.T) {
	suite.Run(t, new(K3SPSACTTestSuite))
}
