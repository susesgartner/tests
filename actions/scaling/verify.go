package scaling

import (
	"testing"

	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	steveV1 "github.com/rancher/shepherd/clients/rancher/v1"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/extensions/defaults/stevetypes"
	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
)

const (
	autoscalerDeployment       = "cluster-autoscaler-clusterapi-kubernetes-cluster-autoscaler"
	autoscalerPausedAnnotation = "provisioning.cattle.io/cluster-autoscaler-paused"
	kubeSystemNamespace        = "kube-system"
)

func VerifyAutoscaler(t *testing.T, client *rancher.Client, cluster *v1.SteveAPIObject) {
	status := &provv1.ClusterStatus{}
	err := steveV1.ConvertToK8sType(cluster.Status, status)
	assert.NoError(t, err)

	downstreamClient, err := client.Steve.ProxyDownstream(status.ClusterName)
	assert.NoError(t, err)
	assert.NotNil(t, downstreamClient)

	deploymentClient := downstreamClient.SteveType(stevetypes.Deployment)

	autoscalerDeployment, err := deploymentClient.ByID(kubeSystemNamespace + "/" + autoscalerDeployment)
	assert.NoError(t, err)

	deployment := &appv1.Deployment{}
	err = steveV1.ConvertToK8sType(autoscalerDeployment.JSONResp, deployment)
	assert.NoError(t, err)
	assert.Equal(t, *deployment.Spec.Replicas, deployment.Status.AvailableReplicas)

	if cluster.Annotations[autoscalerPausedAnnotation] == "true" {
		assert.Zero(t, *deployment.Spec.Replicas)
	} else {
		assert.NotZero(t, *deployment.Spec.Replicas)
	}
}
