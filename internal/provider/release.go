package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	helminternal "github.com/terr4m/terraform-provider-helm/internal/helm"
	"github.com/terr4m/terraform-provider-helm/internal/tfutils"

	releasecommon "helm.sh/helm/v4/pkg/release/common"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
)

var (
	_ resource.Resource              = &ReleaseResource{}
	_ resource.ResourceWithConfigure = &ReleaseResource{}
	// _ resource.ResourceWithValidateConfig = &ReleaseResource{}
	_ resource.ResourceWithModifyPlan = &ReleaseResource{}
	// _ resource.ResourceWithImportState    = &ReleaseResource{}
)

// NewReleaseResource creates a new release resource.
func NewReleaseResource() resource.Resource {
	return &ReleaseResource{}
}

// ReleaseResource defines the resource implementation.
type ReleaseResource struct {
	providerData *HelmProviderData
	service      *helminternal.Service
}

// ReleaseResourceModel describes the release data model.
type ReleaseResourceModel struct {
	Chart     types.String   `tfsdk:"chart"`
	Manifests types.Dynamic  `tfsdk:"manifests"`
	Name      types.String   `tfsdk:"name"`
	Namespace types.String   `tfsdk:"namespace"`
	Status    types.String   `tfsdk:"status"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
	Values    types.Dynamic  `tfsdk:"values"`
	Version   types.String   `tfsdk:"version"`
}

// Metadata returns the resource metadata.
func (d *ReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_release", req.ProviderTypeName)
}

// Schema returns the resource schema.
func (r *ReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "_Helm_ release TF resource.",
		Attributes: map[string]schema.Attribute{
			"chart": schema.StringAttribute{
				MarkdownDescription: "The _Helm_ chart to install.",
				Required:            true,
			},
			"manifests": schema.DynamicAttribute{
				MarkdownDescription: "Manifests that were created by the _Helm_ chart.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The _Helm_ release name.",
				Required:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "The namespace to install the _Helm_ chart into.",
				Required:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the _Helm_ release.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "Timeout for creating the resource; this defaults to the provider value if not set. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
				Read:              true,
				ReadDescription:   "Timeout for reading the resource; this defaults to the provider value if not set. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
				Update:            true,
				UpdateDescription: "Timeout for updating the resource; this defaults to the provider value if not set. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
				Delete:            true,
				DeleteDescription: "Timeout for deleting the resource; this defaults to the provider value if not set. This should be a string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as `30s` or `2h45m`. Valid time units are `s` (seconds), `m` (minutes), `h` (hours).",
			}),
			"values": schema.DynamicAttribute{
				MarkdownDescription: "Values for the _Helm_ chart.",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The _Helm_ chart version to install.",
				Required:            true,
			},
		},
	}
}

// Configure configures the resource.
func (r *ReleaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*HelmProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource provider data.", fmt.Sprintf("expected *HelmProviderData, got: %T", req.ProviderData))
		return
	}

	r.providerData = providerData
	if providerData.Client != nil {
		r.service = helminternal.NewService(providerData.Client, providerData.Client.RegistryClient(), providerData.Client.EnvSettings())
	}
}

// // ValidateConfig validates the resource config.
// func (r *ReleaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
// 	var data ReleaseResourceModel
// 	if resp.Diagnostics.Append(req.Config.Get(ctx, &data)...); resp.Diagnostics.HasError() {
// 		return
// 	}
// }

// ModifyPlan modifies the resource plan.
func (r *ReleaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	if r.providerData == nil {
		resp.Diagnostics.AddError("Missing provider configuration.", "The resource was not configured with provider data before plan modification.")
		return
	}

	var plan ReleaseResourceModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...); resp.Diagnostics.HasError() {
		return
	}

	if plan.Chart.IsNull() || plan.Chart.IsUnknown() ||
		plan.Name.IsNull() || plan.Name.IsUnknown() ||
		plan.Namespace.IsNull() || plan.Namespace.IsUnknown() ||
		plan.Version.IsNull() || plan.Version.IsUnknown() {
		return
	}

	if plan.Values.IsNull() || plan.Values.IsUnknown() || !tfutils.IsFullyKnown(plan.Values.UnderlyingValue()) {
		return
	}

	timeout, diags := plan.Timeouts.Read(ctx, r.providerData.DefaultTimeouts.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	spec, diags := releaseSpecFromModel(ctx, plan, timeout)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	rel, warnings, err := r.service.RenderInstall(ctx, spec)
	resp.Diagnostics.Append(warningsToDiagnostics(warnings)...)
	if err != nil {
		resp.Diagnostics.AddError("Failed to template Helm chart.", err.Error())
		return
	}

	dyn, diags := manifestDynamicValue(ctx, rel.Manifest)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	plan.Manifests = dyn
	plan.Status = types.StringValue(releasecommon.StatusDeployed.String())

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create creates the resource.
func (r *ReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.providerData == nil {
		resp.Diagnostics.AddError("Missing provider configuration.", "The resource was not configured with provider data before create.")
		return
	}

	var plan ReleaseResourceModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...); resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, r.providerData.DefaultTimeouts.Create)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	spec, diags := releaseSpecFromModel(ctx, plan, timeout)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	rel, warnings, err := r.service.Install(ctx, spec)
	resp.Diagnostics.Append(warningsToDiagnostics(warnings)...)
	if err != nil {
		resp.Diagnostics.AddError("Failed to install Helm release.", err.Error())
		return
	}

	state, diags := hydrateReleaseState(ctx, plan, rel)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read reads the resource state.
func (r *ReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.providerData == nil {
		resp.Diagnostics.AddError("Missing provider configuration.", "The resource was not configured with provider data before read.")
		return
	}

	var state ReleaseResourceModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &state)...); resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, r.providerData.DefaultTimeouts.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	spec, diags := releaseSpecFromModel(ctx, state, timeout)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	rel, found, err := r.service.Get(spec)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Helm release.", err.Error())
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	updatedState, diags := hydrateReleaseState(ctx, state, rel)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
}

// Update updates the resource.
func (r *ReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.providerData == nil {
		resp.Diagnostics.AddError("Missing provider configuration.", "The resource was not configured with provider data before update.")
		return
	}

	var plan ReleaseResourceModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...); resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, r.providerData.DefaultTimeouts.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	spec, diags := releaseSpecFromModel(ctx, plan, timeout)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	current, found, err := r.service.Get(spec)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Helm release.", err.Error())
		return
	}

	var rel *helmrelease.Release
	if !found {
		var warnings []string
		rel, warnings, err = r.service.Install(ctx, spec)
		resp.Diagnostics.Append(warningsToDiagnostics(warnings)...)
		if err != nil {
			resp.Diagnostics.AddError("Failed to install Helm release.", err.Error())
			return
		}
	} else {
		planned, warnings, err := r.service.PlanUpgrade(ctx, spec)
		resp.Diagnostics.Append(warningsToDiagnostics(warnings)...)
		if err != nil {
			resp.Diagnostics.AddError("Failed to plan Helm release update.", err.Error())
			return
		}

		if helminternal.NeedsUpgrade(current, planned) {
			var warnings []string
			rel, warnings, err = r.service.Upgrade(ctx, spec)
			resp.Diagnostics.Append(warningsToDiagnostics(warnings)...)
			if err != nil {
				resp.Diagnostics.AddError("Failed to update Helm release.", err.Error())
				return
			}
		} else {
			rel = current
		}
	}

	state, diags := hydrateReleaseState(ctx, plan, rel)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete deletes the resource.
func (r *ReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.providerData == nil {
		resp.Diagnostics.AddError("Missing provider configuration.", "The resource was not configured with provider data before delete.")
		return
	}

	var state ReleaseResourceModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &state)...); resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, r.providerData.DefaultTimeouts.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	deleteSpec, diags := releaseSpecFromModel(ctx, state, timeout)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	if err := r.service.Uninstall(deleteSpec); err != nil {
		resp.Diagnostics.AddError("Failed to uninstall Helm release.", err.Error())
	}
}

func encodeReleaseValues(ctx context.Context, values types.Dynamic) (map[string]interface{}, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	encoded, err := tfutils.EncodeDynamicObject(ctx, values)
	if err != nil {
		diagnostics.AddError("Failed to encode values.", err.Error())
		return nil, diagnostics
	}

	return encoded, diagnostics
}

func manifestDynamicValue(ctx context.Context, manifest string) (types.Dynamic, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	rendered, err := helminternal.DecodeManifest(manifest)
	if err != nil {
		diagnostics.AddError("Failed to unmarshal Helm chart manifest.", err.Error())
		return types.Dynamic{}, diagnostics
	}

	return tfutils.DecodeDynamic(ctx, rendered)
}

func hydrateReleaseState(ctx context.Context, base ReleaseResourceModel, rel *helmrelease.Release) (ReleaseResourceModel, diag.Diagnostics) {
	dyn, diags := manifestDynamicValue(ctx, rel.Manifest)
	if diags.HasError() {
		return ReleaseResourceModel{}, diags
	}

	state := base
	state.Manifests = dyn
	state.Name = types.StringValue(rel.Name)
	state.Namespace = types.StringValue(rel.Namespace)
	state.Status = types.StringValue(helminternal.ReleaseStatus(rel))

	return state, diags
}

func releaseSpecFromModel(ctx context.Context, model ReleaseResourceModel, timeout time.Duration) (helminternal.Spec, diag.Diagnostics) {
	values, diags := encodeReleaseValues(ctx, model.Values)
	if diags.HasError() {
		return helminternal.Spec{}, diags
	}

	return helminternal.Spec{
		ChartRef:  model.Chart.ValueString(),
		Name:      model.Name.ValueString(),
		Namespace: model.Namespace.ValueString(),
		Version:   model.Version.ValueString(),
		Values:    values,
		Timeout:   timeout,
	}, diags
}

func warningsToDiagnostics(warnings []string) diag.Diagnostics {
	var diagnostics diag.Diagnostics

	for _, warning := range warnings {
		diagnostics.AddWarning("Helm chart is deprecated.", warning)
	}

	return diagnostics
}

// // ImportState imports the resource state.
// func (r *ReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	idParts := strings.Split(req.ID, "|")

// 	if len(idParts) != 4 || len(idParts[0]) == 0 || len(idParts[1]) == 0 || len(idParts[2]) == 0 {
// 		resp.Diagnostics.AddError(
// 			"Unexpected import identifier.", fmt.Sprintf("expected import identifier with format: apiVersion|kind|name|namespace; got %q", req.ID),
// 		)
// 		return
// 	}

// 	var a map[string]any
// 	if len(idParts[3]) > 0 {
// 		a = map[string]any{
// 			"apiVersion": idParts[0],
// 			"kind":       idParts[1],
// 			"metadata": map[string]interface{}{
// 				"namespace": idParts[3],
// 				"name":      idParts[2],
// 			},
// 		}
// 	} else {
// 		a = map[string]any{
// 			"apiVersion": idParts[0],
// 			"kind":       idParts[1],
// 			"metadata": map[string]interface{}{
// 				"name": idParts[2],
// 			},
// 		}
// 	}

// 	obj, diags := tfutils.DecodeDynamic(ctx, a)
// 	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
// 		return
// 	}

// 	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("manifest"), &obj)...)
// }
