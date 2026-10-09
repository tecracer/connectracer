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

var _ resource.Resource = &ConnectSecurityProfileAIAgentResource{}
var _ resource.ResourceWithImportState = &ConnectSecurityProfileAIAgentResource{}

func NewConnectSecurityProfileAIAgentResource() resource.Resource {
	return &ConnectSecurityProfileAIAgentResource{}
}

// ConnectSecurityProfileAIAgentResource allows one external AI agent on a security profile.
type ConnectSecurityProfileAIAgentResource struct {
	client *connect.Client
}

// ConnectSecurityProfileAIAgentResourceModel describes the resource data model.
// The ID is "instance_id/security_profile_id/ai_agent_arn".
type ConnectSecurityProfileAIAgentResourceModel struct {
	ID                frameworktypes.String `tfsdk:"id"`
	InstanceID        frameworktypes.String `tfsdk:"instance_id"`
	SecurityProfileID frameworktypes.String `tfsdk:"security_profile_id"`
	AIAgentArn        frameworktypes.String `tfsdk:"ai_agent_arn"`
	Type              frameworktypes.String `tfsdk:"type"`
}

func (r *ConnectSecurityProfileAIAgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_security_profile_ai_agent"
}

func (r *ConnectSecurityProfileAIAgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allows an external AI agent on a Security Profile (`AllowedAIAgents`). An orchestration AI agent " +
			"that carries this security profile may then collaborate with that agent, for example hand a call over to an " +
			"`A2A_SERVER` application. Without the entry the collaboration fails at runtime, when the call is placed.\n\n" +
			"The underlying `UpdateSecurityProfile` API replaces the security profile's entire `AllowedAIAgents` list, so " +
			"this resource performs a read-modify-write on every create, update, and delete — re-supplying the security " +
			"profile's other fields (permissions, applications, flow modules, ...) unchanged so it can coexist with an " +
			"`aws_connect_security_profile` resource and `connectracer_connect_security_profile_flow_module` resources on " +
			"the same security profile. Multiple `connectracer_connect_security_profile_ai_agent` resources can target the " +
			"same `security_profile_id`, as long as they are applied sequentially (the default Terraform behaviour).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier of the entry (`instance_id/security_profile_id/ai_agent_arn`).",
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
				MarkdownDescription: "The identifier of the security profile that allows the AI agent.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ai_agent_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the external AI agent, for an A2A agent the ARN of its AppIntegrations application.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the AI agent. Only `THIRD_PARTY` is currently supported by the AWS API.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(types.AIAgentTypeThirdParty)),
			},
		},
	}
}

func (r *ConnectSecurityProfileAIAgentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectSecurityProfileAIAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectSecurityProfileAIAgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := readSecurityProfileState(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Security Profile", err.Error())
		return
	}

	if findAIAgent(sp.allowedAIAgents, data.AIAgentArn.ValueString()) != nil {
		resp.Diagnostics.AddError(
			"AI Agent Already Allowed",
			fmt.Sprintf("Security profile %s already allows AI agent %s. Import it instead of creating a new one.",
				data.SecurityProfileID.ValueString(), data.AIAgentArn.ValueString()),
		)
		return
	}

	sp.allowedAIAgents = append(sp.allowedAIAgents, types.AIAgent{
		Arn:  aws.String(data.AIAgentArn.ValueString()),
		Type: types.AIAgentType(data.Type.ValueString()),
	})

	// An application created in the same apply can take a moment before Connect accepts it here.
	agentArn := data.AIAgentArn.ValueString()
	err = retryOnEventualConsistency(ctx,
		func(err error) bool {
			return strings.Contains(err.Error(), "InvalidParameterException") && strings.Contains(err.Error(), agentArn)
		},
		func() error {
			return writeSecurityProfileState(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), sp)
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error allowing AI Agent", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(composeSecurityProfileAIAgentID(data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), agentArn))

	tflog.Trace(ctx, "Allowed AI Agent on Security Profile", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectSecurityProfileAIAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectSecurityProfileAIAgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowed, err := listAllowedAIAgents(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Security Profile AI Agents", err.Error())
		return
	}

	agent := findAIAgent(allowed, data.AIAgentArn.ValueString())
	if agent == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Type = frameworktypes.StringValue(string(agent.Type))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectSecurityProfileAIAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectSecurityProfileAIAgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := readSecurityProfileState(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Security Profile", err.Error())
		return
	}

	if agent := findAIAgent(sp.allowedAIAgents, data.AIAgentArn.ValueString()); agent != nil {
		agent.Type = types.AIAgentType(data.Type.ValueString())
	} else {
		sp.allowedAIAgents = append(sp.allowedAIAgents, types.AIAgent{
			Arn:  aws.String(data.AIAgentArn.ValueString()),
			Type: types.AIAgentType(data.Type.ValueString()),
		})
	}

	if err := writeSecurityProfileState(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), sp); err != nil {
		resp.Diagnostics.AddError("Error updating AI Agent access", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectSecurityProfileAIAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectSecurityProfileAIAgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := readSecurityProfileState(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return
		}
		resp.Diagnostics.AddError("Error reading Security Profile", err.Error())
		return
	}

	sp.allowedAIAgents = withoutAIAgent(sp.allowedAIAgents, data.AIAgentArn.ValueString())

	if err := writeSecurityProfileState(ctx, r.client, data.InstanceID.ValueString(), data.SecurityProfileID.ValueString(), sp); err != nil {
		resp.Diagnostics.AddError("Error revoking AI Agent access", err.Error())
		return
	}
}

func (r *ConnectSecurityProfileAIAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The ARN carries slashes of its own, so only the first two separate the parts.
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format instance_id/security_profile_id/ai_agent_arn, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("security_profile_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ai_agent_arn"), parts[2])...)
}

func composeSecurityProfileAIAgentID(instanceID, securityProfileID, agentArn string) string {
	return instanceID + "/" + securityProfileID + "/" + agentArn
}

func findAIAgent(agents []types.AIAgent, arn string) *types.AIAgent {
	for i := range agents {
		if aws.ToString(agents[i].Arn) == arn {
			return &agents[i]
		}
	}
	return nil
}

// withoutAIAgent never returns nil: an empty list is what clears the last entry,
// while a nil list leaves AllowedAIAgents as it is.
func withoutAIAgent(agents []types.AIAgent, arn string) []types.AIAgent {
	kept := make([]types.AIAgent, 0, len(agents))
	for _, agent := range agents {
		if aws.ToString(agent.Arn) != arn {
			kept = append(kept, agent)
		}
	}
	return kept
}
