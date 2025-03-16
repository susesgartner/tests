package permutationdata

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/config/operations/permutations"
	"github.com/sirupsen/logrus"
)

const (
	ClusterConfigKey = "clusterConfig"
	nodeProvidersKey = "nodeProvider"
	providerKey      = "provider"
	k8sVersionKey    = "kubernetesVersion"
	cniKey           = "cni"
)

func CreateK8sPermutation(client *rancher.Client, k8sType string, config map[string]any) (*permutations.Permutation, error) {
	k8sKeyPath := []string{ClusterConfigKey, k8sVersionKey}
	k8sKeyValue, err := operations.GetValue(k8sKeyPath, config)
	if err != nil {
		return nil, err
	}

	if _, ok := k8sKeyValue.(string); ok {
		k8sPermutation := permutations.CreatePermutation(k8sKeyPath, []any{k8sKeyValue}, nil)
		return &k8sPermutation, nil
	}

	if len(k8sKeyValue.([]any)) < 1 || k8sKeyValue.([]any)[0] == "" {
		logrus.Warning("kubernetesVersion not set in config file")
		k8sKeyValue, err = kubernetesversions.Default(client, k8sType, nil)
		if err != nil {
			return nil, err
		}
	} else if k8sKeyValue.([]any)[0] == "all" {
		if k8sType == clusters.RKE2ClusterType.String() {
			k8sKeyValue, err = kubernetesversions.ListRKE2AllVersions(client)
			if err != nil {
				return nil, err
			}
		} else if k8sType == clusters.K3SClusterType.String() {
			k8sKeyValue, err = kubernetesversions.ListK3SAllVersions(client)
			if err != nil {
				return nil, err
			}
		}
	}

	k8sPermutation := permutations.CreatePermutation(k8sKeyPath, k8sKeyValue.([]any), nil)

	return &k8sPermutation, nil
}

func CreateProviderPermutation(config map[string]any) (*permutations.Permutation, error) {
	providerKeyPath := []string{ClusterConfigKey, providerKey}
	providerKeyValue, err := operations.GetValue(providerKeyPath, config)
	if err != nil {
		return nil, err
	}

	if _, ok := providerKeyValue.(string); ok {
		providerKeyValue = []any{providerKeyValue}
	}

	providerPermutation := permutations.CreatePermutation(providerKeyPath, providerKeyValue.([]any), nil)

	return &providerPermutation, nil
}

func CreateCNIPermutation(config map[string]any) (*permutations.Permutation, error) {
	cniKeyPath := []string{ClusterConfigKey, cniKey}
	cniKeyValue, err := operations.GetValue(cniKeyPath, config)
	if err != nil {
		return nil, err
	}

	if _, ok := cniKeyValue.(string); ok {
		cniKeyValue = []any{cniKeyValue}
	}

	cniPermutation := permutations.CreatePermutation(cniKeyPath, cniKeyValue.([]any), nil)

	return &cniPermutation, nil
}
