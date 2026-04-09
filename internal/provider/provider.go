package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure HelmProvider satisfies various provider interfaces.
var (
	_ provider.Provider                       = &HelmProvider{}
	_ provider.ProviderWithFunctions          = &HelmProvider{}
	_ provider.ProviderWithEphemeralResources = &HelmProvider{}
)

// New returns a new provider implementation.
func New(version, commit string) func() provider.Provider {
	return func() provider.Provider {
		return &HelmProvider{
			version: version,
			commit:  commit,
		}
	}
}

// HelmProviderData is the data available to the resource and data sources.
type HelmProviderData struct {
	provider        *HelmProvider
	Model           *HelmProviderModel
	Client          *HelmProviderClient
	DefaultTimeouts *Timeouts
}

// Timeouts represents a set of timeouts.
type Timeouts struct {
	Create time.Duration
	Read   time.Duration
	Update time.Duration
	Delete time.Duration
}

// HelmProviderModel describes the provider data model.
type HelmProviderModel struct {
	RegistryConfig   types.String           `tfsdk:"registry_config"`
	RepositoryConfig types.String           `tfsdk:"repository_config"`
	RepositoryCache  types.String           `tfsdk:"repository_cache"`
	Kubernetes       *KubernetesConfigModel `tfsdk:"kubernetes"`
	Timeouts         timeouts.Value         `tfsdk:"timeouts"`
}

// KubernetesConfigModel describes the Kubernetes configuration model.
type KubernetesConfigModel struct {
	Host                  types.String     `tfsdk:"host"`
	Username              types.String     `tfsdk:"username"`
	Password              types.String     `tfsdk:"password"`
	Insecure              types.Bool       `tfsdk:"insecure"`
	TLSServerName         types.String     `tfsdk:"tls_server_name"`
	ClientCertificate     types.String     `tfsdk:"client_certificate"`
	ClientKey             types.String     `tfsdk:"client_key"`
	ClusterCACertificate  types.String     `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           types.List       `tfsdk:"config_paths"`
	ConfigContext         types.String     `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String     `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String     `tfsdk:"config_context_cluster"`
	Token                 types.String     `tfsdk:"token"`
	ProxyURL              types.String     `tfsdk:"proxy_url"`
	Exec                  *ExecConfigModel `tfsdk:"exec"`
}

// ExecConfigModel configures an external command to configure the Kubernetes client.
type ExecConfigModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Command    types.String `tfsdk:"command"`
	Env        types.Map    `tfsdk:"env"`
	Args       types.List   `tfsdk:"args"`
}

// HelmProvider defines the provider implementation.
type HelmProvider struct {
	version string
	commit  string
}

func (p *HelmProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "helm"
	resp.Version = p.version
}

func (p *HelmProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Helm provider.",
		Attributes: map[string]schema.Attribute{
			"registry_config": schema.StringAttribute{
				MarkdownDescription: "Path to the _Helm_ registry configuration file. Can be set with the `HELM_REGISTRY_CONFIG` environment variable.",
				Optional:            true,
			},
			"repository_config": schema.StringAttribute{
				MarkdownDescription: "Path to the _Helm_ repository configuration file. Can be set with the `HELM_REPOSITORY_CONFIG` environment variable.",
				Optional:            true,
			},
			"repository_cache": schema.StringAttribute{
				MarkdownDescription: "Path to the _Helm_ repository cache. Can be set with the `HELM_REPOSITORY_CACHE` environment variable.",
				Optional:            true,
			},
			"kubernetes": schema.SingleNestedAttribute{
				MarkdownDescription: "Kubernetes configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						MarkdownDescription: "The hostname (in form of URI) of _Kubernetes_ master. Can be set with the `KUBE_HOST` environment variable.",
						Optional:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "The username to use for HTTP basic authentication when accessing the _Kubernetes_ master endpoint. Can be set with the `KUBE_USER` environment variable.",
						Optional:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "The password to use for HTTP basic authentication when accessing the _Kubernetes_ master endpoint. Can be set with the `KUBE_PASSWORD` environment variable.",
						Optional:            true,
					},
					"insecure": schema.BoolAttribute{
						MarkdownDescription: "Whether server should be accessed without verifying the TLS certificate. Can be set with the `KUBE_INSECURE` environment variable.",
						Optional:            true,
					},
					"tls_server_name": schema.StringAttribute{
						MarkdownDescription: "Server name passed to the server for SNI and is used in the client to check server certificates against. Can be set with the `KUBE_TLS_SERVER_NAME` environment variable.",
						Optional:            true,
					},
					"client_certificate": schema.StringAttribute{
						MarkdownDescription: "PEM-encoded client certificate for TLS authentication. Can be set with the `KUBE_CLIENT_CERT_DATA` environment variable.",
						Optional:            true,
					},
					"client_key": schema.StringAttribute{
						MarkdownDescription: "PEM-encoded client certificate key for TLS authentication. Can be set with the `KUBE_CLIENT_KEY_DATA` environment variable.",
						Optional:            true,
					},
					"cluster_ca_certificate": schema.StringAttribute{
						MarkdownDescription: "PEM-encoded root certificates bundle for TLS authentication. Can be set with the `KUBE_CLUSTER_CA_CERT_DATA` environment variable.",
						Optional:            true,
					},
					"config_paths": schema.ListAttribute{
						MarkdownDescription: "List of paths to the kube config file. Can be set with the `KUBE_CONFIG_PATHS` environment variable.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"config_context": schema.StringAttribute{
						MarkdownDescription: "Context to choose from the kube config file. Can be set with the `KUBE_CTX`environment variable.",
						Optional:            true,
					},
					"config_context_auth_info": schema.StringAttribute{
						MarkdownDescription: "Authentication info context of the kube config (name of the kube config user, --user flag in kubectl). Can be set with the `KUBE_CTX_AUTH_INFO` environment variable.",
						Optional:            true,
					},
					"config_context_cluster": schema.StringAttribute{
						MarkdownDescription: "Cluster context of the kube config (name of the kube config cluster, --cluster flag in kubectl). Can be set with the `KUBE_CTX_CLUSTER` environment variable.",
						Optional:            true,
					},
					"token": schema.StringAttribute{
						MarkdownDescription: "Token to authenticate a service account. Can be set with the `KUBE_TOKEN` environment variable.",
						Optional:            true,
					},
					"proxy_url": schema.StringAttribute{
						MarkdownDescription: "URL to the proxy to be used for all API requests. Can be set with the `KUBE_PROXY_URL` environment variable.",
						Optional:            true,
					},
					"exec": schema.SingleNestedAttribute{
						MarkdownDescription: "Exec configuration for Kubernetes authentication.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"api_version": schema.StringAttribute{
								MarkdownDescription: "API version for the exec plugin.",
								Required:            true,
							},
							"command": schema.StringAttribute{
								MarkdownDescription: "Command to run for _Kubernetes_ exec plugin.",
								Required:            true,
							},
							"env": schema.MapAttribute{
								MarkdownDescription: "Environment variables for the exec plugin.",
								ElementType:         types.StringType,
								Optional:            true,
							},
							"args": schema.ListAttribute{
								MarkdownDescription: "Arguments for the exec plugin.",
								ElementType:         types.StringType,
								Optional:            true,
							},
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "Timeout for resource creation; defaults to `10m`. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
				Read:              true,
				ReadDescription:   "Timeout for resource or data source reads; defaults to `10m`. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
				Update:            true,
				UpdateDescription: "Timeout for resource update; defaults to `10m`. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
				Delete:            true,
				DeleteDescription: "Timeout for resource deletion; defaults to `10m`. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
			}),
		},
	}
}

// Configure configures the provider.
func (p *HelmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	if req.ClientCapabilities.DeferralAllowed && !req.Config.Raw.IsFullyKnown() {
		resp.Deferred = &provider.Deferred{
			Reason: provider.DeferredReasonProviderConfigUnknown,
		}
	}

	// Load the provider config
	model := &HelmProviderModel{}
	if resp.Diagnostics.Append(req.Config.Get(ctx, model)...); resp.Diagnostics.HasError() {
		return
	}

	// Create a Kubernetes REST client configuration.
	client, diags := getHelmClient(ctx, *model)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	// Lookup timeouts
	createTimeout, diags := model.Timeouts.Create(ctx, 10*time.Minute)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	readTimeout, diags := model.Timeouts.Read(ctx, 10*time.Minute)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	updateTimeout, diags := model.Timeouts.Update(ctx, 10*time.Minute)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	deleteTimeout, diags := model.Timeouts.Delete(ctx, 10*time.Minute)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	// Configure provider data
	providerData := &HelmProviderData{
		provider: p,
		Model:    model,
		Client:   client,
		DefaultTimeouts: &Timeouts{
			Create: createTimeout,
			Read:   readTimeout,
			Update: updateTimeout,
			Delete: deleteTimeout,
		},
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *HelmProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewReleaseResource,
	}
}

func (p *HelmProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *HelmProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *HelmProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}
