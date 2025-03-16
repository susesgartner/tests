package permutationdata

import (
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

func CreateK8sPermutation(config map[string]any) (*permutations.Permutation, error) {
	k8sKeyPath := []string{ClusterConfigKey, k8sVersionKey}
	k8sKeyValue, err := operations.GetValue(k8sKeyPath, config)
	if err != nil {
		logrus.Warning("kubernetesVersion not set in config file")
		k8sKeyValue = []any{}
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

	providerPermutation := permutations.CreatePermutation(providerKeyPath, providerKeyValue.([]any), nil)

	return &providerPermutation, nil
}

func CreateCNIPermutation(config map[string]any) (*permutations.Permutation, error) {
	cniKeyPath := []string{ClusterConfigKey, cniKey}
	cniKeyValue, err := operations.GetValue(cniKeyPath, config)
	if err != nil {
		return nil, err
	}

	cniPermutation := permutations.CreatePermutation(cniKeyPath, cniKeyValue.([]any), nil)

	return &cniPermutation, nil
}
