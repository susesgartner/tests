package charts

import (
	"context"
	"fmt"

	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	rv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	"github.com/rancher/shepherd/pkg/wait"
	"github.com/rancher/tests/actions/charts"
	kubenamespaces "github.com/rancher/tests/actions/kubeapi/namespaces"
	"github.com/rancher/tests/actions/namespaces"
	"github.com/rancher/tests/interoperability/observability"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	// Public constants
	StackstateExtensionNamespace = "cattle-ui-plugin-system"
	StackstateExtensionsName     = "observability"
	UIPluginName                 = "rancher-ui-plugins"
	StackstateK8sAgent           = "stackstate-k8s-agent"
	StackstateNamespace          = "stackstate"
	StackstateCRD                = "observability.rancher.io.configuration"
	RancherPartnerChartRepo      = "rancher-partner-charts"
	rancherChartsName            = "rancher-charts"
	rancherPartnerCharts         = "rancher-partner-charts"
	serverURLSettingID           = "server-url"
	StackStateServerChartRepo    = "suse-observability"
	StackStateServerNamespace    = "suse-observability"
)

var (
	timeoutSeconds = int64(defaults.TwoMinuteTimeout)
)

// InstallStackStateServerChart installs the StackState chart into the specified Kubernetes cluster and namespace.
// It uses Rancher client and configuration details, and waits for the chart deployment to complete successfully.
// Parameters:
// - client: the Rancher client used for connecting to the Kubernetes cluster.
// - installOptions: the installation options including cluster info and chart version.
// - stackstateConfigs: configuration details for StackState such as the service token and URL.
// - systemProjectID: the ID of the system project where the chart is deployed.
// - additionalValues: additional Helm chart values as a map for custom configurations.
// Returns an error if the chart installation fails or its status cannot be confirmed.
func InstallStackStateServerChart(client *rancher.Client, installOptions *charts.InstallOptions, systemProjectID string, additionalValues map[string]interface{}) error {

	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		log.Info("Error getting server setting.")
		return err
	}

	stackstateChartInstallActionPayload := &charts.PayloadOpts{
		InstallOptions: *installOptions,
		Name:           StackStateServerChartRepo,
		Namespace:      StackStateServerNamespace,
		Host:           serverSetting.Value,
	}

	chartInstallAction := newStackStateServerChartInstallAction(stackstateChartInstallActionPayload, systemProjectID, additionalValues)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		log.Info("Error getting catalogClient")
		return err
	}

	err = catalogClient.InstallChart(chartInstallAction, StackStateServerChartRepo)
	if err != nil {
		log.Info("Error installing the StackState chart")
		return err
	}

	watchAppInterface, err := catalogClient.Apps(StackStateServerNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + StackStateServerChartRepo,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	if err != nil {
		log.Info("StackState App failed to install")
		return err
	}

	err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
		app := event.Object.(*catalogv1.App)

		state := app.Status.Summary.State
		if state == string(catalogv1.StatusDeployed) {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		log.Info("Unable to obtain the status of the installed app ")
		return err
	}
	return nil
}

// InstallStackstateAgentChart is a private helper function that returns chart install action with stack state agent and payload options.
func InstallStackstateAgentChart(client *rancher.Client, installOptions *charts.InstallOptions, stackstateConfigs *observability.StackStateConfig, systemProjectID string) error {
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		log.Info("Error getting server setting.")
		return err
	}

	stackstateAgentChartInstallActionPayload := &charts.PayloadOpts{
		InstallOptions: *installOptions,
		Name:           StackstateK8sAgent,
		Namespace:      StackstateNamespace,
		Host:           serverSetting.Value,
	}

	chartInstallAction := newStackstateAgentChartInstallAction(stackstateAgentChartInstallActionPayload, stackstateConfigs, systemProjectID)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		log.Info("Error getting catalogClient")
		return err
	}

	// register uninstall stackstate agent as a cleanup function
	client.Session.RegisterCleanupFunc(func() error {
		defaultChartUninstallAction := charts.NewChartUninstallAction()

		err := catalogClient.UninstallChart(StackstateK8sAgent, StackstateNamespace, defaultChartUninstallAction)
		if err != nil {
			return err
		}

		watchAppInterface, err := catalogClient.Apps(StackstateNamespace).Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + StackstateK8sAgent,
			TimeoutSeconds: &timeoutSeconds,
		})
		if err != nil {
			return err
		}

		err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
			chart := event.Object.(*catalogv1.App)
			if event.Type == watch.Error {
				return false, fmt.Errorf("there was an error uninstalling stackstate agent chart")
			} else if event.Type == watch.Deleted {
				return true, nil
			} else if chart == nil {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return err
		}

		steveclient, err := client.Steve.ProxyDownstream(installOptions.Cluster.ID)
		if err != nil {
			return err
		}
		namespaceClient := steveclient.SteveType(namespaces.NamespaceSteveType)

		namespace, err := namespaceClient.ByID(StackstateNamespace)
		if err != nil {
			return err
		}

		err = namespaceClient.Delete(namespace)
		if err != nil {
			return err
		}

		adminClient, err := rancher.NewClient(client.RancherConfig.AdminToken, client.Session)
		if err != nil {
			return err
		}

		adminDynamicClient, err := adminClient.GetDownStreamClusterClient(installOptions.Cluster.ID)
		if err != nil {
			return err
		}

		adminNamespaceResource := adminDynamicClient.Resource(kubenamespaces.NamespaceGroupVersionResource).Namespace("")

		watchNamespaceInterface, err := adminNamespaceResource.Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + StackstateNamespace,
			TimeoutSeconds: &timeoutSeconds,
		})
		if err != nil {
			return err
		}

		return wait.WatchWait(watchNamespaceInterface, func(event watch.Event) (ready bool, err error) {
			if event.Type == watch.Deleted {
				return true, nil
			}
			return false, nil
		})
	})

	err = catalogClient.InstallChart(chartInstallAction, RancherPartnerChartRepo)
	if err != nil {
		log.Info("Errored installing the chart")
		return err
	}

	// wait for chart to be fully deployed
	watchAppInterface, err := catalogClient.Apps(StackstateNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + StackstateK8sAgent,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	if err != nil {
		log.Info("Unable to obtain the installed app ")
		return err
	}

	err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
		app := event.Object.(*catalogv1.App)

		state := app.Status.Summary.State
		if state == string(catalogv1.StatusDeployed) {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		log.Info("Unable to obtain the status of the installed app ")
		return err
	}
	return nil
}

// newStackstateAgentChartInstallAction is a helper function that returns an array of charts.NewChartInstallActions for installing the stackstate agent charts
func newStackstateAgentChartInstallAction(p *charts.PayloadOpts, stackstateConfigs *observability.StackStateConfig, systemProjectID string) *types.ChartInstallAction {
	stackstateValues := map[string]interface{}{
		"stackstate": map[string]interface{}{
			"cluster": map[string]interface{}{
				"name": p.Cluster.Name,
			},
			"apiKey": stackstateConfigs.ClusterApiKey,
			"url":    stackstateConfigs.Url,
		},
	}

	chartInstall := charts.NewChartInstall(p.Name, p.Version, p.Cluster.ID, p.Cluster.Name, p.Host, rancherPartnerCharts, systemProjectID, p.DefaultRegistry, stackstateValues)

	chartInstalls := []types.ChartInstall{*chartInstall}
	chartInstallAction := charts.NewChartInstallAction(p.Namespace, p.ProjectID, chartInstalls)

	return chartInstallAction
}

func newStackStateServerChartInstallAction(p *charts.PayloadOpts, systemProjectID string, additionalValues map[string]interface{}) *types.ChartInstallAction {

	chartInstall := charts.NewChartInstall(p.Name, p.Version, p.Cluster.ID, p.Cluster.Name, p.Host, StackStateServerChartRepo, systemProjectID, p.DefaultRegistry, additionalValues)

	chartInstalls := []types.ChartInstall{*chartInstall}
	chartInstallAction := charts.NewChartInstallAction(p.Namespace, p.ProjectID, chartInstalls)

	return chartInstallAction
}

// UpgradeStackstateAgentChart is a helper function that upgrades the stackstate agent chart.
func UpgradeStackstateAgentChart(client *rancher.Client, installOptions *charts.InstallOptions, stackstateConfigs *observability.StackStateConfig, systemProjectID string) error {
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}

	stackstateAgentChartUpgradeActionPayload := &charts.PayloadOpts{
		InstallOptions: *installOptions,
		Name:           StackstateK8sAgent,
		Namespace:      StackstateNamespace,
		Host:           serverSetting.Value,
	}

	chartUpgradeAction := newStackstateAgentChartUpgradeAction(stackstateAgentChartUpgradeActionPayload, stackstateConfigs)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	err = catalogClient.UpgradeChart(chartUpgradeAction, RancherPartnerChartRepo)
	if err != nil {
		return err
	}

	adminClient, err := rancher.NewClient(client.RancherConfig.AdminToken, client.Session)
	if err != nil {
		return err
	}

	adminCatalogClient, err := adminClient.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// wait for chart to be in status pending upgrade
	watchAppInterface, err := adminCatalogClient.Apps(StackstateNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + StackstateK8sAgent,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	if err != nil {
		return err
	}

	err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
		app := event.Object.(*catalogv1.App)

		state := app.Status.Summary.State
		if state == string(catalogv1.StatusPendingUpgrade) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	watchAppInterface, err = adminCatalogClient.Apps(StackstateNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + StackstateK8sAgent,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	if err != nil {
		return err
	}

	err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
		app := event.Object.(*catalogv1.App)

		state := app.Status.Summary.State
		if state == string(catalogv1.StatusDeployed) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// newStackstateAgentChartUpgradeAction is a private helper function that returns chart upgrade action.
func newStackstateAgentChartUpgradeAction(p *charts.PayloadOpts, stackstateConfigs *observability.StackStateConfig) *types.ChartUpgradeAction {

	stackstateValues := map[string]interface{}{
		"stackstate": map[string]interface{}{
			"cluster": map[string]interface{}{
				"name": p.Cluster.Name,
			},
			"apiKey": stackstateConfigs.ClusterApiKey,
			"url":    stackstateConfigs.Url,
		},
	}

	chartUpgrade := charts.NewChartUpgrade(p.Name, p.Name, p.Version, p.Cluster.ID, p.Cluster.Name, p.Host, p.DefaultRegistry, stackstateValues)
	chartUpgrades := []types.ChartUpgrade{*chartUpgrade}
	chartUpgradeAction := charts.NewChartUpgradeAction(p.Namespace, chartUpgrades)

	return chartUpgradeAction
}

// CreateClusterRepo creates a new ClusterRepo resource in the Kubernetes cluster using the provided catalog client.
// It takes the client, repository name, and repository URL as arguments and returns an error if the operation fails.
func CreateClusterRepo(client *rancher.Client, catalogClient *catalog.Client, name, url string) error {
	ctx := context.Background()
	repo := &rv1.ClusterRepo{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       rv1.RepoSpec{URL: url},
	}
	_, err := catalogClient.ClusterRepos().Create(ctx, repo, metav1.CreateOptions{})

	client.Session.RegisterCleanupFunc(func() error {

		var propagation = metav1.DeletePropagationForeground
		err := catalogClient.ClusterRepos().Delete(context.Background(), name, metav1.DeleteOptions{PropagationPolicy: &propagation})
		if err != nil {
			return err
		}

		return err
	})
	return err
}

// StructToMap Helper function to convert struct to map[string]interface{}
func StructToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var intermediate map[interface{}]interface{}
	if err = yaml.Unmarshal(data, &intermediate); err != nil {
		return nil, err
	}

	return convertMapInterfaceToMapString(intermediate), nil
}

// Converts map[interface{}]interface{} to map[string]interface{}
func convertMapInterfaceToMapString(obj map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range obj {
		key := fmt.Sprintf("%v", k)
		result[key] = convertToStringKeys(v)
	}
	return result
}

// convertToStringKeys converts any map[interface{}]interface{} to map[string]interface{} recursively
func convertToStringKeys(val interface{}) interface{} {
	switch v := val.(type) {
	case map[interface{}]interface{}:
		return convertMapToStringKeys(v)
	case []interface{}:
		convertedSlice := make([]interface{}, len(v))
		for i, v2 := range v {
			convertedSlice[i] = convertToStringKeys(v2)
		}
		return convertedSlice
	default:
		return v
	}
}

// convertMapToStringKeys converts a map[interface{}]interface{} to a map[string]interface{}
func convertMapToStringKeys(input map[interface{}]interface{}) map[string]interface{} {
	strMap := make(map[string]interface{})
	for k, v := range input {
		strKey := fmt.Sprintf("%v", k)
		strMap[strKey] = convertToStringKeys(v)
	}
	return strMap
}

// MergeValues merges multiple map[string]interface{} values into one, recursively merging nested maps if keys overlap.
func MergeValues(values ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, currentMap := range values {
		for key, currentValue := range currentMap {
			if existingValue, exists := result[key]; exists {
				// Check if both existing and current values are maps,
				// if so, recursively merge them
				mergedMap := mergeMaps(existingValue, currentValue)
				if mergedMap != nil {
					result[key] = mergedMap
					continue
				}
			}
			// Otherwise, overwrite with the current value
			result[key] = currentValue
		}
	}
	return result
}

// mergeMaps recursively merges two maps if both are maps, else returns nil
func mergeMaps(existingValue, currentValue interface{}) map[string]interface{} {
	existingMap, ok1 := existingValue.(map[string]interface{})
	currentMap, ok2 := currentValue.(map[string]interface{})
	if ok1 && ok2 {
		return MergeValues(existingMap, currentMap)
	}
	return nil
}
