package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/go-homedir"

	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/helmpath"
	"helm.sh/helm/v4/pkg/registry"
)

// getHelmClient returns the Helm client.
func getHelmClient(ctx context.Context, model HelmProviderModel) (*HelmProviderClient, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	envSettings := &cli.EnvSettings{}

	if !model.RegistryConfig.IsNull() {
		envSettings.RegistryConfig = model.RegistryConfig.ValueString()
	} else if v := os.Getenv("HELM_REGISTRY_CONFIG"); len(v) != 0 {
		envSettings.RegistryConfig = v
	} else {
		envSettings.RegistryConfig = helmpath.ConfigPath("registry/config.json")
	}

	if !model.RepositoryConfig.IsNull() {
		envSettings.RepositoryConfig = model.RepositoryConfig.ValueString()
	} else if v := os.Getenv("HELM_REPOSITORY_CONFIG"); len(v) != 0 {
		envSettings.RepositoryConfig = v
	} else {
		envSettings.RepositoryConfig = helmpath.ConfigPath("repositories.yaml")
	}

	if !model.RepositoryCache.IsNull() {
		envSettings.RepositoryCache = model.RepositoryCache.ValueString()
	} else if v := os.Getenv("HELM_REPOSITORY_CACHE"); len(v) != 0 {
		envSettings.RepositoryCache = v
	} else {
		envSettings.RepositoryCache = helmpath.CachePath("repository")
	}

	k8sClientConfig, diags := getK8sClientConfig(ctx, *model.Kubernetes)
	if diagnostics.Append(diags...); diagnostics.HasError() {
		return nil, diagnostics
	}
	restClient := &RESTClientGetter{
		ClientConfig: k8sClientConfig,
	}

	registryClient, err := newRegistryClient(envSettings.RegistryConfig)
	if err != nil {
		diagnostics.AddError("Failed to create Helm registry client.", err.Error())
		return nil, diagnostics
	}

	return NewHelmProviderClient(restClient, registryClient, envSettings), diagnostics
}

// getK8sClientConfig returns the Kubernetes client config.
func getK8sClientConfig(ctx context.Context, model KubernetesConfigModel) (clientcmd.ClientConfig, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	loader := &clientcmd.ClientConfigLoadingRules{}
	overrides := &clientcmd.ConfigOverrides{}

	var configPaths []string
	if !model.ConfigPaths.IsNull() {
		configPaths = make([]string, 0, len(model.ConfigPaths.Elements()))
		diags := model.ConfigPaths.ElementsAs(ctx, &configPaths, false)
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return nil, diagnostics
		}
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); len(v) != 0 {
		configPaths = filepath.SplitList(v)
	} else if v := os.Getenv("KUBE_CONFIG_PATH"); len(v) != 0 {
		configPaths = []string{v}
	}

	if len(configPaths) > 0 {
		for i, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				diagnostics.AddError("Failed to expand home directory.", err.Error())
				return nil, diagnostics
			}

			configPaths[i] = path
		}

		if len(configPaths) == 1 {
			loader.ExplicitPath = configPaths[0]
		} else {
			loader.Precedence = configPaths
		}

		if !model.ConfigContext.IsNull() || !model.ConfigContextAuthInfo.IsNull() || !model.ConfigContextCluster.IsNull() {
			if !model.ConfigContext.IsNull() {
				overrides.CurrentContext = model.ConfigContext.ValueString()
			} else if v := os.Getenv("KUBE_CTX"); len(v) != 0 {
				overrides.CurrentContext = v
			}

			overrides.Context = clientcmdapi.Context{}

			if !model.ConfigContextAuthInfo.IsNull() {
				overrides.Context.AuthInfo = model.ConfigContextAuthInfo.ValueString()
			} else if v := os.Getenv("KUBE_CTX_AUTH_INFO"); len(v) != 0 {
				overrides.Context.AuthInfo = v
			}

			if !model.ConfigContextCluster.IsNull() {
				overrides.Context.Cluster = model.ConfigContextCluster.ValueString()
			} else if v := os.Getenv("KUBE_CTX_CLUSTER"); len(v) != 0 {
				overrides.Context.Cluster = v
			}
		}
	}

	if !model.Insecure.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = model.Insecure.ValueBool()
	} else if v := os.Getenv("KUBE_INSECURE"); len(v) != 0 {
		insecure, err := strconv.ParseBool(v)
		if err != nil {
			diagnostics.AddError("Failed to parse KUBE_INSECURE environment variable.", err.Error())
			return nil, diagnostics
		}

		overrides.ClusterInfo.InsecureSkipTLSVerify = insecure
	}

	if !model.TLSServerName.IsNull() {
		overrides.ClusterInfo.TLSServerName = model.TLSServerName.ValueString()
	} else if v := os.Getenv("KUBE_TLS_SERVER_NAME"); len(v) != 0 {
		overrides.ClusterInfo.TLSServerName = v
	}

	if !model.ClusterCACertificate.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(model.ClusterCACertificate.ValueString())
	} else if v := os.Getenv("KUBE_CLUSTER_CA_CERT_DATA"); len(v) != 0 {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(v)
	}

	if !model.ClientCertificate.IsNull() {
		overrides.AuthInfo.ClientCertificateData = []byte(model.ClientCertificate.ValueString())
	} else if v := os.Getenv("KUBE_CLIENT_CERT_DATA"); len(v) != 0 {
		overrides.AuthInfo.ClientCertificateData = []byte(v)
	}

	var host string
	if !model.Host.IsNull() {
		host = model.Host.ValueString()
	} else if v := os.Getenv("KUBE_HOST"); len(v) != 0 {
		host = v
	}

	if len(host) > 0 {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := (hasCA || hasCert) && !overrides.ClusterInfo.InsecureSkipTLSVerify
		url, _, err := rest.DefaultServerURL(host, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			diagnostics.AddError("Failed to get host URL.", err.Error())
			return nil, diagnostics
		}

		overrides.ClusterInfo.Server = url.String()
	}

	if !model.Username.IsNull() {
		overrides.AuthInfo.Username = model.Username.ValueString()
	} else if v := os.Getenv("KUBE_USER"); len(v) != 0 {
		overrides.AuthInfo.Username = v
	}

	if !model.Password.IsNull() {
		overrides.AuthInfo.Password = model.Password.ValueString()
	} else if v := os.Getenv("KUBE_PASSWORD"); len(v) != 0 {
		overrides.AuthInfo.Password = v
	}

	if !model.ClientKey.IsNull() {
		overrides.AuthInfo.ClientKeyData = []byte(model.ClientKey.ValueString())
	} else if v := os.Getenv("KUBE_CLIENT_KEY_DATA"); len(v) != 0 {
		overrides.AuthInfo.ClientKeyData = []byte(v)
	}

	if !model.Token.IsNull() {
		overrides.AuthInfo.Token = model.Token.ValueString()
	} else if v := os.Getenv("KUBE_TOKEN"); len(v) != 0 {
		overrides.AuthInfo.Token = v
	}

	if model.Exec != nil {
		execModel := model.Exec
		if !execModel.APIVersion.IsNull() && !execModel.Command.IsNull() {
			args := []string{}
			if !execModel.Args.IsNull() && !execModel.Args.IsUnknown() {
				args = make([]string, 0, len(execModel.Args.Elements()))
				diags := execModel.Args.ElementsAs(ctx, &args, false)
				diagnostics.Append(diags...)
				if diagnostics.HasError() {
					return nil, diagnostics
				}
			}

			env := []clientcmdapi.ExecEnvVar{}
			if !execModel.Env.IsNull() && !execModel.Env.IsUnknown() {
				env = make([]clientcmdapi.ExecEnvVar, 0, len(execModel.Env.Elements()))
				for k, v := range execModel.Env.Elements() {
					val, ok := v.(types.String)
					if !ok {
						diagnostics.AddError("Invalid type in exec env.", fmt.Sprintf("expected string, got %T", v))
						return nil, diagnostics
					}

					env = append(env, clientcmdapi.ExecEnvVar{
						Name:  k,
						Value: val.ValueString(),
					})
				}
			}

			overrides.AuthInfo.Exec = &clientcmdapi.ExecConfig{
				APIVersion:      execModel.APIVersion.ValueString(),
				Command:         execModel.Command.ValueString(),
				Args:            args,
				Env:             env,
				InteractiveMode: clientcmdapi.IfAvailableExecInteractiveMode,
			}
		}
	}

	if !model.ProxyURL.IsNull() {
		overrides.ClusterDefaults.ProxyURL = model.ProxyURL.ValueString()
	} else if v := os.Getenv("KUBE_PROXY_URL"); len(v) != 0 {
		overrides.ClusterDefaults.ProxyURL = v
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides), diagnostics
}

// newRegistryClient creates a new Helm registry client.
func newRegistryClient(registryConfigPath string) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(false),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(registryConfigPath),
	}

	return registry.NewClient(opts...)
}
