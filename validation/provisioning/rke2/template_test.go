//go:build validation || recurring

package rke2

import (
	"testing"
	"time"

	"github.com/rancher/rancher/tests/v2/actions/charts"
	"github.com/rancher/rancher/tests/v2/actions/provisioning"
	"github.com/rancher/rancher/tests/v2/actions/provisioninginput"
	"github.com/rancher/rancher/tests/v2/actions/reports"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/extensions/cloudcredentials"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/extensions/defaults/namespaces"
	"github.com/rancher/shepherd/extensions/defaults/stevestates"
	"github.com/rancher/shepherd/extensions/defaults/stevetypes"
	"github.com/rancher/shepherd/extensions/steve"
	"github.com/rancher/shepherd/extensions/users"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/namegenerator"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/stretchr/testify/assert"
)

const (
	localCluster          = "local"
	providerName          = "rke2"
	templateTestConfigKey = "templateTest"
)

type templateTest struct {
	client             *rancher.Client
	standardUserClient *rancher.Client
	session            *session.Session
	templateConfig     *provisioninginput.TemplateConfig
	cloudCredentials   *v1.SteveAPIObject
}

func templateSetup(t *testing.T) templateTest {
	var r templateTest
	testSession := session.NewSession()
	r.session = testSession

	r.templateConfig = new(provisioninginput.TemplateConfig)
	config.LoadConfig(templateTestConfigKey, r.templateConfig)

	client, err := rancher.NewClient("", testSession)
	assert.NoError(t, err)
	r.client = client

	provider := provisioning.CreateProvider(r.templateConfig.TemplateProvider)
	cloudCredentialConfig := cloudcredentials.LoadCloudCredential(r.templateConfig.TemplateProvider)
	r.cloudCredentials, err = provider.CloudCredFunc(client, cloudCredentialConfig)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	newUser.Password = user.Password

	standardUserClient, err := client.AsUser(newUser)
	assert.NoError(t, err)

	r.standardUserClient = standardUserClient

	return r
}

func TestTemplate(t *testing.T) {
	t.Parallel()
	r := templateSetup(t)

	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{"RKE2_Template|etcd|cp|worker", r.standardUserClient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := steve.CreateAndWaitForResource(r.client, namespaces.FleetLocal+"/"+localCluster, stevetypes.ClusterRepo, r.templateConfig.Repo, stevestates.Active, 5*time.Second, defaults.FiveMinuteTimeout)
			assert.NoError(t, err)

			k8sversions, err := kubernetesversions.Default(r.client, providerName, nil)
			assert.NoError(t, err)

			clusterName := namegenerator.AppendRandomString(providerName + "-template")
			err = charts.InstallTemplateChart(r.client, r.templateConfig.Repo.ObjectMeta.Name, r.templateConfig.TemplateName, clusterName, k8sversions[0], r.cloudCredentials)
			assert.NoError(t, err)

			_, cluster, err := clusters.GetProvisioningClusterByName(r.client, clusterName, namespaces.FleetDefault)
			reports.TimeoutClusterReport(cluster, err)
			assert.NoError(t, err)

			provisioning.VerifyCluster(t, r.client, nil, cluster)
		})
	}
}
