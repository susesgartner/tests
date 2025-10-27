package featureflags

import (
	"context"
	"strings"
	"time"

	"github.com/rancher/norman/types"
	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	steveV1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/extensions/defaults/namespaces"
	"github.com/rancher/shepherd/extensions/defaults/stevetypes"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	"github.com/sirupsen/logrus"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

const (
	rancherPodProfix = "rancher"
)

func UpdateFeatureFlag(client *rancher.Client, name string, enabled bool) error {
	featureOpts := &types.ListOpts{Filters: map[string]interface{}{
		"name": name,
	}}

	features, err := client.Management.Feature.List(featureOpts)
	if err != nil {
		return err
	}

	for _, feature := range features.Data {
		if feature.Value != &enabled {
			feature.Value = &enabled
		}

		logrus.Debugf("Updating: %s state to %v", feature.Name, *feature.Value)
		client.Management.Feature.Update(&feature, &feature)
	}

	cluster, err := client.Steve.SteveType(stevetypes.Provisioning).ByID(namespaces.FleetLocal + "/local")
	if err != nil {
		return err
	}

	status := &provv1.ClusterStatus{}
	err = steveV1.ConvertToK8sType(cluster.Status, status)
	if err != nil {
		return err
	}

	downstreamClient, err := client.Steve.ProxyDownstream(status.ClusterName)
	if err != nil {
		return err
	}

	logrus.Debug("Waiting for rancher deployment to restart")
	restarted := false
	steveClient := downstreamClient.SteveType(stevetypes.Pod)
	err = kwait.PollUntilContextTimeout(context.TODO(), 5*time.Second, defaults.FiveMinuteTimeout, true, func(ctx context.Context) (done bool, err error) {
		clusterPods, err := steveClient.List(nil)
		if err != nil {
			restarted = true
			return false, nil
		}

		for _, pod := range clusterPods.Data {
			if strings.Contains(pod.Name, rancherPodProfix) {
				isReady, err := pods.IsPodReady(&pod)
				if !isReady {
					restarted = true
					return false, err
				}
			}
		}

		if !restarted {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		logrus.Warning("Rancher restart was not observed")
	}

	return nil
}
