package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/registry"
)

// NewHelmProviderClient returns a new Helm provider client.
func NewHelmProviderClient(restClient *RESTClientGetter, registryClient *registry.Client, envSettings *cli.EnvSettings) *HelmProviderClient {
	return &HelmProviderClient{
		restClient:     *restClient,
		registryClient: *registryClient,
		envSettings:    *envSettings,
	}
}

// HelmProviderClient holds the provider scoped Helm client configuration.
type HelmProviderClient struct {
	restClient     RESTClientGetter
	registryClient registry.Client
	envSettings    cli.EnvSettings
}

// GetActionConfig returns a Helm action configuration.
func (c *HelmProviderClient) GetActionConfig(namespace string) (*action.Configuration, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	actionConfig, err := c.GetActionConfigE(namespace)
	if err != nil {
		diagnostics.AddError("Failed to get Helm action config.", err.Error())
		return nil, diagnostics
	}

	return actionConfig, diagnostics
}

// GetActionConfigE returns a Helm action configuration or an error.
func (c *HelmProviderClient) GetActionConfigE(namespace string) (*action.Configuration, error) {
	actionConfig := &action.Configuration{}

	if err := actionConfig.Init(c.restClient, namespace, ""); err != nil {
		return nil, fmt.Errorf("init helm action config: %w", err)
	}

	actionConfig.RegistryClient = &c.registryClient

	return actionConfig, nil
}

// RegistryClient returns the provider-scoped Helm registry client.
func (c *HelmProviderClient) RegistryClient() *registry.Client {
	return &c.registryClient
}

// EnvSettings returns the provider-scoped Helm environment settings.
func (c *HelmProviderClient) EnvSettings() *cli.EnvSettings {
	return &c.envSettings
}

// RESTClientGetter is a Kubernetes REST client getter implementing the genericclioptions.RESTClientGetter interface.
type RESTClientGetter struct {
	ClientConfig clientcmd.ClientConfig
}

// ToRESTConfig returns a Kubernetes REST configuration.
func (r RESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.ClientConfig.ClientConfig()
}

// ToDiscoveryClient returns a Kubernetes discovery client.
func (r RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	c, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	d, err := discovery.NewDiscoveryClientForConfig(c)
	if err != nil {
		return nil, err
	}

	return memory.NewMemCacheClient(d), nil
}

// ToRESTMapper returns a Kubernetes REST mapper.
func (r RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	d, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	return restmapper.NewDeferredDiscoveryRESTMapper(d), nil
}

// ToRawKubeConfigLoader returns the Kubernetes configuration loader as-is.
func (r RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return r.ClientConfig
}
