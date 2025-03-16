package defaults

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/pkg/config/operations"
)

const (
	ClusterConfigKey = "clusterConfig"
	k8sVersionKey    = "kubernetesVersion"
)

func SetK8sDefault(client *rancher.Client, k8sType string, cattleConfig map[string]any) (map[string]any, error) {
	k8sKeyPath := []string{ClusterConfigKey, k8sVersionKey}
	k8sKeyValue, err := operations.GetValue(k8sKeyPath, cattleConfig)
	if err != nil {
		return nil, err
	}

	if k8sKeyValue == nil || len(k8sKeyValue.([]any)) == 0 {
		versions, err := kubernetesversions.Default(client, k8sType, nil)
		if err != nil {
			return nil, err
		}

		cattleConfig, err = operations.ReplaceValue(k8sKeyPath, versions[0], cattleConfig)
		if err != nil {
			return nil, err
		}

	} else if k8sKeyValue.([]any)[0] == "all" {
		if k8sType == clusters.RKE2ClusterType.String() {
			rke2Versions, err := kubernetesversions.ListRKE2AllVersions(client)
			if err != nil {
				return nil, err
			}

			cattleConfig, err = operations.ReplaceValue(k8sKeyPath, rke2Versions, cattleConfig)
			if err != nil {
				return nil, err
			}

		} else if k8sType == clusters.K3SClusterType.String() {
			k3sVersions, err := kubernetesversions.ListK3SAllVersions(client)
			if err != nil {
				return nil, err
			}

			cattleConfig, err = operations.ReplaceValue(k8sKeyPath, k3sVersions, cattleConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	return cattleConfig, nil
}
