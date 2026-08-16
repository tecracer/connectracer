// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectSecurityProfileFlowModuleResource{}
var _ resource.ResourceWithImportState = &ConnectSecurityProfileFlowModuleResource{}

func NewConnectSecurityProfileFlowModuleResource() resource.Resource {
	return &ConnectSecurityProfileFlowModuleResource{}
}

// ConnectSecurityProfileFlowModuleResource defines the resource implementation.
type ConnectSecurityProfileFlowModuleResource struct {
	client *connect.Client
}

// ConnectSecurityProfileFlowModuleResourceModel describes the resource data model.
// The ID is "instance_id/security_profile_id/flow_module_id".
type ConnectSecurityProfileFlowModuleResourceModel struct {
	ID                frameworktypes.String `tfsdk:"id"`
	InstanceID        frameworktypes.String `tfsdk:"instance_id"`
	SecurityProfileID frameworktypes.String `tfsdk:"security_profile_id"`
	FlowModuleID      frameworktypes.String `tfsdk:"flow_module_id"`
	Type              frameworktypes.String `tfsdk:"type"`
}

func (r *ConnectSecurityProfileFlowModuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_security_profile_flow_module"
}

func (r *ConnectSecurityProfileFlowModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants a Security Profile permission to invoke a Flow Module as a tool (e.g. by a Q in Connect " +
			"orchestration AI agent using that security profile). This is the same mechanism used to grant AI agents access " +
			"to MCP tools.\n\n" +
			"The underlying `UpdateSecurityProfile` API replaces the security profile's entire `AllowedFlowModules` list, so " +
			"this resource performs a read-modify-write on every create, update, and delete — re-supplying the security " +
			"profile's other fields (permissions, applications, description, ...) unchanged so it can coexist with an " +
			"`aws_connect_security_profile` resource managing the rest of the security profile. Multiple " +
			"`connectracer_connect_security_profile_flow_module` resources can target the same `security_profile_id` for " +
			"different flow modules, as long as they are applied sequentially (the default Terraform behaviour).\n\n" +
			"The flow module referenced by `flow_module_id` must have `external_invocation_enabled = true` (see " +
			"`connectracer_connect_flow_module_tool`).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier of the grant (`instance_id/security_profile_id/flow_module_id`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Amazon Connect instance.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_profile_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the security profile to grant flow-module-as-tool access to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flow_module_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the flow module the security profile is allowed to invoke as a tool.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of tool invocation. Only `MCP` is currently supported by the AWS API.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(types.FlowModuleTypeMcp)),
			},
		},
	}
}

func (r *ConnectSecurityProfileFlowModuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = clients.Connect
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *ConnectSecurityProfileFlowModuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectSecurityProfileFlowModuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := r.getSecurityProfileState(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Security Profile", err.Error())
		return
	}

	for _, fm := range sp.allowedFlowModules {
		if aws.ToString(fm.FlowModuleId) == data.FlowModuleID.ValueString() {
			resp.Diagnostics.AddError(
				"Flow Module Grant Already Exists",
				fmt.Sprintf("Security profile %s already allows flow module %s. Import it instead of creating a new one.",
					data.SecurityProfileID.ValueString(), data.FlowModuleID.ValueString()),
			)
			return
		}
	}

	sp.allowedFlowModules = append(sp.allowedFlowModules, types.FlowModule{
		FlowModuleId: aws.String(data.FlowModuleID.ValueString()),
		Type:         types.FlowModuleType(data.Type.ValueString()),
	})

	// UpdateSecurityProfile can reject a flow module ID as "not valid". Most of
	// the time this actually means the flow module itself isn't eligible yet —
	// missing description or no released version, see
	// connectracer_connect_flow_module_tool, which now handles both — but a
	// genuine eventual-consistency gap right after CreateContactFlowModule in
	// the same apply (the same kind observed between CreateSecurityProfile and
	// AssociateSecurityProfiles) remains plausible, so still retry a few times
	// before surfacing the error.
	flowModuleID := data.FlowModuleID.ValueString()
	err = retryOnEventualConsistency(ctx,
		func(err error) bool {
			return strings.Contains(err.Error(), "InvalidParameterException") && strings.Contains(err.Error(), flowModuleID)
		},
		func() error {
			return r.updateSecurityProfile(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), sp)
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error granting Flow Module access", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(r.composeID(data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), data.FlowModuleID.ValueString()))

	tflog.Trace(ctx, "Granted Flow Module access to Security Profile", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectSecurityProfileFlowModuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectSecurityProfileFlowModuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowed, err := r.listAllowedFlowModules(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Security Profile Flow Modules", err.Error())
		return
	}

	fm := r.findFlowModule(allowed, data.FlowModuleID.ValueString())
	if fm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Type = frameworktypes.StringValue(string(fm.Type))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectSecurityProfileFlowModuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectSecurityProfileFlowModuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := r.getSecurityProfileState(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Security Profile", err.Error())
		return
	}

	replaced := false
	for i, fm := range sp.allowedFlowModules {
		if aws.ToString(fm.FlowModuleId) == data.FlowModuleID.ValueString() {
			sp.allowedFlowModules[i].Type = types.FlowModuleType(data.Type.ValueString())
			replaced = true
			break
		}
	}
	if !replaced {
		// Grant disappeared between plan and apply; add it back.
		sp.allowedFlowModules = append(sp.allowedFlowModules, types.FlowModule{
			FlowModuleId: aws.String(data.FlowModuleID.ValueString()),
			Type:         types.FlowModuleType(data.Type.ValueString()),
		})
	}

	if err := r.updateSecurityProfile(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), sp); err != nil {
		resp.Diagnostics.AddError("Error updating Flow Module access", err.Error())
		return
	}

	tflog.Trace(ctx, "Updated Flow Module grant on Security Profile", map[string]any{
		"flow_module_id": data.FlowModuleID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectSecurityProfileFlowModuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectSecurityProfileFlowModuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := r.getSecurityProfileState(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			// Security profile already gone; nothing to clean up.
			return
		}
		resp.Diagnostics.AddError("Error reading Security Profile", err.Error())
		return
	}

	filtered := make([]types.FlowModule, 0, len(sp.allowedFlowModules))
	for _, fm := range sp.allowedFlowModules {
		if aws.ToString(fm.FlowModuleId) != data.FlowModuleID.ValueString() {
			filtered = append(filtered, fm)
		}
	}
	sp.allowedFlowModules = filtered

	if err := r.updateSecurityProfile(ctx, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), sp); err != nil {
		resp.Diagnostics.AddError("Error revoking Flow Module access", err.Error())
		return
	}

	tflog.Trace(ctx, "Revoked Flow Module grant from Security Profile", map[string]any{
		"flow_module_id": data.FlowModuleID.ValueString(),
	})
}

func (r *ConnectSecurityProfileFlowModuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: "instance_id/security_profile_id/flow_module_id"
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format instance_id/security_profile_id/flow_module_id, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("security_profile_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("flow_module_id"), parts[2])...)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// composeID returns the canonical composite ID for a flow module grant.
func (r *ConnectSecurityProfileFlowModuleResource) composeID(instanceID, securityProfileID, flowModuleID string) string {
	return instanceID + "/" + securityProfileID + "/" + flowModuleID
}

// findFlowModule returns a pointer to the FlowModule whose FlowModuleId matches id, or nil.
func (r *ConnectSecurityProfileFlowModuleResource) findFlowModule(modules []types.FlowModule, id string) *types.FlowModule {
	for i := range modules {
		if aws.ToString(modules[i].FlowModuleId) == id {
			return &modules[i]
		}
	}
	return nil
}

// listAllowedFlowModules returns the full, paginated AllowedFlowModules list for a security profile.
func (r *ConnectSecurityProfileFlowModuleResource) listAllowedFlowModules(ctx context.Context, instanceID, securityProfileID string) ([]types.FlowModule, error) {
	var out []types.FlowModule
	paginator := connect.NewListSecurityProfileFlowModulesPaginator(r.client, &connect.ListSecurityProfileFlowModulesInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile flow modules: %w", err)
		}
		out = append(out, page.AllowedFlowModules...)
	}
	return out, nil
}

// securityProfileState bundles every field UpdateSecurityProfile can set, so it can be
// re-supplied unchanged except for the one AllowedFlowModules entry this resource owns.
type securityProfileState struct {
	description                          *string
	allowedAccessControlHierarchyGroupID *string
	allowedAccessControlTags             map[string]string
	granularAccessControlConfiguration   *types.GranularAccessControlConfiguration
	hierarchyRestrictedResources         []string
	permissions                          []string
	applications                         []types.Application
	allowedFlowModules                   []types.FlowModule
}

// getSecurityProfileState fetches every field needed to safely round-trip an UpdateSecurityProfile
// call without clobbering fields owned by other resources (e.g. aws_connect_security_profile).
func (r *ConnectSecurityProfileFlowModuleResource) getSecurityProfileState(ctx context.Context, instanceID, securityProfileID string) (*securityProfileState, error) {
	described, err := r.client.DescribeSecurityProfile(ctx, &connect.DescribeSecurityProfileInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	if err != nil {
		return nil, fmt.Errorf("describing security profile %s: %w", securityProfileID, err)
	}
	if described.SecurityProfile == nil {
		return nil, fmt.Errorf("DescribeSecurityProfile returned empty response for %s", securityProfileID)
	}
	sp := described.SecurityProfile

	state := &securityProfileState{
		description:                          sp.Description,
		allowedAccessControlHierarchyGroupID: sp.AllowedAccessControlHierarchyGroupId,
		allowedAccessControlTags:             sp.AllowedAccessControlTags,
		granularAccessControlConfiguration:   sp.GranularAccessControlConfiguration,
		hierarchyRestrictedResources:         sp.HierarchyRestrictedResources,
	}

	permPaginator := connect.NewListSecurityProfilePermissionsPaginator(r.client, &connect.ListSecurityProfilePermissionsInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for permPaginator.HasMorePages() {
		page, err := permPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile permissions: %w", err)
		}
		state.permissions = append(state.permissions, page.Permissions...)
	}

	appPaginator := connect.NewListSecurityProfileApplicationsPaginator(r.client, &connect.ListSecurityProfileApplicationsInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for appPaginator.HasMorePages() {
		page, err := appPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile applications: %w", err)
		}
		state.applications = append(state.applications, page.Applications...)
	}

	allowedFlowModules, err := r.listAllowedFlowModules(ctx, instanceID, securityProfileID)
	if err != nil {
		return nil, err
	}
	state.allowedFlowModules = allowedFlowModules

	return state, nil
}

// updateSecurityProfile calls UpdateSecurityProfile, re-supplying every field on sp so that
// only the caller's intended AllowedFlowModules change takes effect.
func (r *ConnectSecurityProfileFlowModuleResource) updateSecurityProfile(ctx context.Context, instanceID, securityProfileID string, sp *securityProfileState) error {
	input := &connect.UpdateSecurityProfileInput{
		InstanceId:                           aws.String(instanceID),
		SecurityProfileId:                    aws.String(securityProfileID),
		Description:                          sp.description,
		AllowedAccessControlHierarchyGroupId: sp.allowedAccessControlHierarchyGroupID,
		AllowedAccessControlTags:             sp.allowedAccessControlTags,
		GranularAccessControlConfiguration:   sp.granularAccessControlConfiguration,
		HierarchyRestrictedResources:         sp.hierarchyRestrictedResources,
		Permissions:                          sp.permissions,
		Applications:                         sp.applications,
		AllowedFlowModules:                   sp.allowedFlowModules,
	}

	_, err := r.client.UpdateSecurityProfile(ctx, input)
	if err != nil {
		return fmt.Errorf("UpdateSecurityProfile failed: %w", err)
	}
	return nil
}
