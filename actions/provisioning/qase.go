package provisioning

import (
	"strings"

	rancherEc2 "github.com/rancher/shepherd/clients/ec2"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/cloudcredentials"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/machinepools"
	"github.com/sirupsen/logrus"
	upstream "go.qase.io/qase-api-client"
)

// GetProvisioningSchemaParams gets a set of params from the cattle config and returns a qase params object
func GetProvisioningSchemaParams(client *rancher.Client, cattleConfig map[string]any) []upstream.TestCaseParameterCreate {
	var params []upstream.TestCaseParameterCreate
	clusterConfig := new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, cattleConfig, clusterConfig)

	provider := CreateProvider(clusterConfig.Provider)
	credentialSpec := cloudcredentials.LoadCloudCredential(string(provider.Name))
	machineConfigSpec := machinepools.LoadMachineConfigs(string(provider.Name))

	var osParam upstream.TestCaseParameterCreate
	if strings.Contains(clusterConfig.Provider, "aws") {
		osNames, err := provider.GetOSNamesFunc(client, credentialSpec, machineConfigSpec)
		if err != nil {
			logrus.Warningf("Error getting OS Name %s", err)
			return nil
		}

		osParam = upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "OS", Values: osNames}}
	}

	providerParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "Provider", Values: []string{clusterConfig.Provider}}}
	k8sParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "K8sVersion", Values: []string{clusterConfig.KubernetesVersion}}}
	cniParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "CNI", Values: []string{clusterConfig.CNI}}}

	params = append(params, providerParam, osParam, k8sParam, cniParam)

	return params
}

// GetCustomSchemaParams gets a set of params from the cattle config and returns a qase params object
func GetCustomSchemaParams(client *rancher.Client, cattleConfig map[string]any) []upstream.TestCaseParameterCreate {
	var params []upstream.TestCaseParameterCreate
	clusterConfig := new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, cattleConfig, clusterConfig)

	customConfig := new(rancherEc2.AWSEC2Configs)
	operations.LoadObjectFromMap(defaults.AWSEC2Configs, cattleConfig, customConfig)
	externalNodeProvider := ExternalNodeProviderSetup(clusterConfig.NodeProvider)

	osNames, err := externalNodeProvider.GetOSNamesFunc(client, *customConfig)
	if err != nil {
		logrus.Warningf("Error getting OS Name %s", err)
		return nil
	}

	osParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "OS", Values: osNames}}
	providerParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "Provider", Values: []string{clusterConfig.Provider}}}
	k8sParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "K8sVersion", Values: []string{clusterConfig.KubernetesVersion}}}
	cniParam := upstream.TestCaseParameterCreate{ParameterSingle: &upstream.ParameterSingle{Title: "CNI", Values: []string{clusterConfig.CNI}}}

	params = append(params, providerParam, osParam, k8sParam, cniParam)

	return params
}
