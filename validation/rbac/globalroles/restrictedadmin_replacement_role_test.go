package globalroles

import (
	"testing"

	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	extensionscluster "github.com/rancher/shepherd/extensions/clusters"
	extensionsettings "github.com/rancher/shepherd/extensions/settings"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
	rbac "github.com/rancher/tests/actions/rbac"
	"github.com/rancher/tests/actions/settings"
	"github.com/rancher/tests/actions/users"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RestrictedAdminReplacementTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	cluster            *management.Cluster
	provisioningConfig *provisioninginput.Config
}

func (ra *RestrictedAdminReplacementTestSuite) TearDownSuite() {
	ra.session.Cleanup()
}

func (ra *RestrictedAdminReplacementTestSuite) SetupSuite() {
	ra.session = session.NewSession()

	ra.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, ra.provisioningConfig)

	client, err := rancher.NewClient("", ra.session)
	require.NoError(ra.T(), err)
	ra.client = client

	log.Info("Getting cluster name from the config file and append cluster details in the struct.")
	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(ra.T(), clusterName, "Cluster name to install should be set")
	clusterID, err := extensionscluster.GetClusterIDByName(ra.client, clusterName)
	require.NoError(ra.T(), err, "Error getting cluster ID")
	ra.cluster, err = ra.client.Management.Cluster.ByID(clusterID)
	require.NoError(ra.T(), err)
}

func (ra *RestrictedAdminReplacementTestSuite) createRestrictedAdminReplacementRoleAndUser(addManageUsersRule bool) (*v3.GlobalRole, *management.User, *rancher.Client) {
	restrictedAdminReplacementRoleName := namegen.AppendRandomString("restricted-admin-replacement")
	restrictedAdminReplacementRole := newRestrictedAdminReplacementTemplate(restrictedAdminReplacementRoleName)

	if addManageUsersRule {
		restrictedAdminReplacementRole.Rules = append(restrictedAdminReplacementRole.Rules, manageUsersVerb)
	}

	globalRole, user, err := createCustomGlobalRoleAndUser(ra.client, &restrictedAdminReplacementRole)
	require.NoError(ra.T(), err, "failed to create global role and user")

	userClient, err := ra.client.AsUser(user)
	require.NoError(ra.T(), err, "failed to create user client")

	return globalRole, user, userClient
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementCreateCluster() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role")
	createdRaReplacementRole, createdRaReplacementUser, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(false)

	ra.T().Logf("Verifying user %s with role %s can create a downstream cluster", createdRaReplacementUser.Name, createdRaReplacementRole.Name)
	nodeRolesAll := []provisioninginput.MachinePools{provisioninginput.AllRolesMachinePool}
	provisioningConfig := *ra.provisioningConfig
	provisioningConfig.MachinePools = nodeRolesAll

	log.Info("Setting up cluster config and provider for downstream k3s cluster")
	clusterConfig := clusters.ConvertConfigToClusterConfig(&provisioningConfig)
	clusterConfig.KubernetesVersion = ra.provisioningConfig.K3SKubernetesVersions[0]
	k3sprovider, _, _, _ := permutations.GetClusterProvider(permutations.K3SProvisionCluster, (*clusterConfig.Providers)[0], &provisioningConfig)
	clusterObject, err := provisioning.CreateProvisioningCluster(createdRaReplacementUserClient, *k3sprovider, clusterConfig, nil)
	require.NoError(ra.T(), err)

	provisioning.VerifyCluster(ra.T(), ra.client, clusterConfig, clusterObject)
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementListGlobalSettings() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role")
	createdRaReplacementRole, createdRaReplacementUser, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(false)

	log.Infof("Verifying user %s with role %s can list global settings", createdRaReplacementUser.Name, createdRaReplacementRole.Name)
	raReplacementUserSettingsList, err := settings.GetGlobalSettingNames(createdRaReplacementUserClient, ra.cluster.ID)
	require.NoError(ra.T(), err)
	adminGlobalSettingsList, err := settings.GetGlobalSettingNames(ra.client, ra.cluster.ID)
	require.NoError(ra.T(), err)

	require.Equal(ra.T(), adminGlobalSettingsList, raReplacementUserSettingsList)
	require.Equal(ra.T(), len(adminGlobalSettingsList), len(raReplacementUserSettingsList))
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementCantUpdateGlobalSettings() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role")
	_, createdRaReplacementUser, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(false)

	steveRAReplacementClient := createdRaReplacementUserClient.Steve
	steveAdminClient := ra.client.Steve

	log.Info("Getting kubeconfig-default-token-ttl-minutes setting")
	kubeConfigTokenSetting, err := steveAdminClient.SteveType(extensionsettings.ManagementSetting).ByID(extensionsettings.KubeConfigToken)
	require.NoError(ra.T(), err)

	log.Infof("Verifying user %s with the replacement restricted admin global role cannot update setting", createdRaReplacementUser.Name)
	_, err = extensionsettings.UpdateGlobalSettings(steveRAReplacementClient, kubeConfigTokenSetting, "3")
	require.Error(ra.T(), err)
	require.Contains(ra.T(), err.Error(), "Resource type [management.cattle.io.setting] is not updatable")
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementCantUpdateAdminPassword() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role")
	_, _, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(false)

	adminUser, err := createUserWithBuiltinRole(ra.client, rbac.Admin)
	require.NoError(ra.T(), err)

	log.Info("Verifying user with restricted admin replacement role cannot update admin user password")
	testPass := password.GenerateUserPassword("testpass-")
	err = users.UpdateUserPassword(createdRaReplacementUserClient, adminUser, testPass)
	require.Error(ra.T(), err)
	require.Contains(ra.T(), err.Error(), "denied the request: request is attempting to modify user with more permissions than requester user")
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementCantDeleteAdmin() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role")
	_, _, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(false)

	adminUser, err := createUserWithBuiltinRole(ra.client, rbac.Admin)
	require.NoError(ra.T(), err)

	log.Info("Verifying user with restricted admin replacement role cannot delete admin user")
	err = createdRaReplacementUserClient.Management.User.Delete(adminUser)
	require.Error(ra.T(), err)
	require.Contains(ra.T(), err.Error(), "denied the request: request is attempting to modify user with more permissions than requester user")
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementWithManageUsersCanUpdateAdminPassword() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role with the manage-users verb")
	_, _, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(true)

	adminUser, err := createUserWithBuiltinRole(ra.client, rbac.Admin)
	require.NoError(ra.T(), err)

	log.Info("Verifying user with restricted admin replacement role with manage-users verb can update admin user password")
	testPass := password.GenerateUserPassword("testpass-")
	err = users.UpdateUserPassword(createdRaReplacementUserClient, adminUser, testPass)
	require.NoError(ra.T(), err)
}

func (ra *RestrictedAdminReplacementTestSuite) TestRestrictedAdminReplacementWithManageUsersCanDeleteAdmin() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating the replacement restricted admin global role with the manage-users verb")
	_, _, createdRaReplacementUserClient := ra.createRestrictedAdminReplacementRoleAndUser(true)

	adminUser, err := createUserWithBuiltinRole(ra.client, rbac.Admin)
	require.NoError(ra.T(), err)

	log.Info("Verifying user with restricted admin replacement role with manage users verb can delete admin user")
	err = createdRaReplacementUserClient.Management.User.Delete(adminUser)
	require.NoError(ra.T(), err)
}

func TestRestrictedAdminReplacementTestSuite(t *testing.T) {
	suite.Run(t, new(RestrictedAdminReplacementTestSuite))
}
