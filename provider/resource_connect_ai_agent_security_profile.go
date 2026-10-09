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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectAIAgentSecurityProfileResource{}
var _ resource.ResourceWithImportState = &ConnectAIAgentSecurityProfileResource{}

func NewConnectAIAgentSecurityProfileResource() resource.Resource {
	return &ConnectAIAgentSecurityProfileResource{}
}

// ConnectAIAgentSecurityProfileResource defines the resource implementation.
type ConnectAIAgentSecurityProfileResource struct {
	client *connect.Client
}

// ConnectAIAgentSecurityProfileResourceModel describes the resource data model.
// The ID is "instance_id/ai_agent_arn/security_profile_id".
type ConnectAIAgentSecurityProfileResourceModel struct {
	ID                frameworktypes.String `tfsdk:"id"`
	InstanceID        frameworktypes.String `tfsdk:"instance_id"`
	AIAgentArn        frameworktypes.String `tfsdk:"ai_agent_arn"`
	SecurityProfileID frameworktypes.String `tfsdk:"security_profile_id"`
	AIAgentVersion    frameworktypes.Int64  `tfsdk:"ai_agent_version"`
}

func (r *ConnectAIAgentSecurityProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_ai_agent_security_profile"
}

func (r *ConnectAIAgentSecurityProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Associates a Security Profile with a Q in Connect AI Agent (via `AssociateSecurityProfiles`, " +
			"`EntityType=AI_AGENT`). Without this association the AI Agent has no security profile at all, so it cannot " +
			"invoke governed tools — Flow Module tools (see `connectracer_connect_security_profile_flow_module`), " +
			"AgentCore/MCP tools, and out-of-the-box tools (Cases, Tasks, Customer Profiles, Knowledge Base retrieval) — " +
			"even if the flow module or MCP tool itself is otherwise fully configured.\n\n" +
			"`RETURN_TO_CONTROL` tools (`connectracer_connect_ai_tool` with `tool_type = \"RETURN_TO_CONTROL\"`, e.g. " +
			"`Complete`/`Escalate`) are not governed this way — they don't access any protected resource, so they work " +
			"without any security profile association.\n\n" +
			"An agent that hands over to an external collaborator (A2A) also needs the association on its versions. " +
			"Set `ai_agent_version` to the agent's published version and the resource associates `:$LATEST`, `:$SAVED` " +
			"and `:<version>` as well, and moves the numbered association along when the version changes.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier of the association (`instance_id/ai_agent_arn/security_profile_id`).",
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
			"ai_agent_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the Q in Connect AI Agent (unqualified — no `:$LATEST` or version suffix).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_profile_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the security profile to associate with the AI Agent.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ai_agent_version": schema.Int64Attribute{
				MarkdownDescription: "The published version of the AI Agent, usually `connectracer_connect_ai_agent.<name>.version_number`. " +
					"When set, the security profile is associated with `:$LATEST`, `:$SAVED` and `:<version>` of the agent as well. " +
					"`:$SAVED` is skipped while the agent has no saved state.",
				Optional: true,
			},
		},
	}
}

func (r *ConnectAIAgentSecurityProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectAIAgentSecurityProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectAIAgentSecurityProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	associated, err := r.listAssociatedSecurityProfiles(ctx, data.InstanceID.ValueString(), data.AIAgentArn.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading AI Agent Security Profiles", err.Error())
		return
	}
	for _, sp := range associated {
		if aws.ToString(sp.Id) == data.SecurityProfileID.ValueString() {
			resp.Diagnostics.AddError(
				"Security Profile Association Already Exists",
				fmt.Sprintf("AI Agent %s is already associated with security profile %s. Import it instead of creating a new one.",
					data.AIAgentArn.ValueString(), data.SecurityProfileID.ValueString()),
			)
			return
		}
	}

	tflog.Debug(ctx, "Associating Security Profile with AI Agent", map[string]any{
		"instance_id":         data.InstanceID.ValueString(),
		"ai_agent_arn":        data.AIAgentArn.ValueString(),
		"security_profile_id": data.SecurityProfileID.ValueString(),
	})

	for _, entityArn := range associatedAIAgentArns(data.AIAgentArn.ValueString(), data.AIAgentVersion) {
		if err := r.associate(ctx, data.InstanceID.ValueString(), entityArn, data.SecurityProfileID.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				"Error associating Security Profile with AI Agent",
				fmt.Sprintf("Unable to associate security profile %s with AI agent %s: %s",
					data.SecurityProfileID.ValueString(), entityArn, err),
			)
			return
		}
	}

	data.ID = frameworktypes.StringValue(r.composeID(data.InstanceID.ValueString(), data.AIAgentArn.ValueString(), data.SecurityProfileID.ValueString()))

	tflog.Trace(ctx, "Associated Security Profile with AI Agent", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIAgentSecurityProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectAIAgentSecurityProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	associated, err := r.listAssociatedSecurityProfiles(ctx, data.InstanceID.ValueString(), data.AIAgentArn.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AI Agent Security Profiles", err.Error())
		return
	}

	found := false
	for _, sp := range associated {
		if aws.ToString(sp.Id) == data.SecurityProfileID.ValueString() {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// A version association that went missing reads as no version, so the next plan puts it back.
	for _, entityArn := range versionArnsToCheck(data.AIAgentArn.ValueString(), data.AIAgentVersion) {
		profiles, err := r.listAssociatedSecurityProfiles(ctx, data.InstanceID.ValueString(), entityArn)
		if err != nil || !containsSecurityProfile(profiles, data.SecurityProfileID.ValueString()) {
			data.AIAgentVersion = frameworktypes.Int64Null()
			break
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update only ever sees ai_agent_version change: every other attribute replaces the resource.
func (r *ConnectAIAgentSecurityProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ConnectAIAgentSecurityProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned := associatedAIAgentArns(data.AIAgentArn.ValueString(), data.AIAgentVersion)
	previous := associatedAIAgentArns(state.AIAgentArn.ValueString(), state.AIAgentVersion)

	for _, entityArn := range arnsMissingFrom(previous, planned) {
		if err := r.disassociate(ctx, data.InstanceID.ValueString(), entityArn, data.SecurityProfileID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error disassociating Security Profile from AI Agent version", err.Error())
			return
		}
	}
	for _, entityArn := range planned {
		if err := r.associate(ctx, data.InstanceID.ValueString(), entityArn, data.SecurityProfileID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error associating Security Profile with AI Agent version", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIAgentSecurityProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectAIAgentSecurityProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disassociating Security Profile from AI Agent", map[string]any{
		"ai_agent_arn":        data.AIAgentArn.ValueString(),
		"security_profile_id": data.SecurityProfileID.ValueString(),
	})

	for _, entityArn := range associatedAIAgentArns(data.AIAgentArn.ValueString(), data.AIAgentVersion) {
		if err := r.disassociate(ctx, data.InstanceID.ValueString(), entityArn, data.SecurityProfileID.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				"Error disassociating Security Profile from AI Agent",
				fmt.Sprintf("Unable to disassociate security profile %s from AI agent %s: %s",
					data.SecurityProfileID.ValueString(), entityArn, err),
			)
			return
		}
	}

	tflog.Trace(ctx, "Disassociated Security Profile from AI Agent")
}

func (r *ConnectAIAgentSecurityProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	instanceID, aiAgentArn, securityProfileID, err := parseAIAgentSecurityProfileImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ai_agent_arn"), aiAgentArn)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("security_profile_id"), securityProfileID)...)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// composeID returns the canonical composite ID for an AI Agent <-> Security Profile association.
func (r *ConnectAIAgentSecurityProfileResource) composeID(instanceID, aiAgentArn, securityProfileID string) string {
	return instanceID + "/" + aiAgentArn + "/" + securityProfileID
}

// listAssociatedSecurityProfiles returns the full, paginated list of security profiles
// associated with the given AI Agent.
func (r *ConnectAIAgentSecurityProfileResource) listAssociatedSecurityProfiles(ctx context.Context, instanceID, aiAgentArn string) ([]types.SecurityProfileItem, error) {
	var out []types.SecurityProfileItem
	paginator := connect.NewListEntitySecurityProfilesPaginator(r.client, &connect.ListEntitySecurityProfilesInput{
		InstanceId: aws.String(instanceID),
		EntityArn:  aws.String(aiAgentArn),
		EntityType: types.EntityTypeAiAgent,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing AI agent security profiles: %w", err)
		}
		out = append(out, page.SecurityProfiles...)
	}
	return out, nil
}

// parseAIAgentSecurityProfileImportID splits an import ID of the form
// "instance_id/ai_agent_arn/security_profile_id" into its three parts. The middle
// segment (an ARN) may itself contain "/", so this can't be a plain SplitN(3): the
// instance_id is taken up to the first "/", the security_profile_id after the last "/",
// and whatever remains in between is the ARN.
func parseAIAgentSecurityProfileImportID(id string) (instanceID, aiAgentArn, securityProfileID string, err error) {
	firstSlash := strings.Index(id, "/")
	lastSlash := strings.LastIndex(id, "/")

	if firstSlash <= 0 || lastSlash <= firstSlash || lastSlash == len(id)-1 {
		return "", "", "", fmt.Errorf("expected format instance_id/ai_agent_arn/security_profile_id, got: %q", id)
	}

	instanceID = id[:firstSlash]
	aiAgentArn = id[firstSlash+1 : lastSlash]
	securityProfileID = id[lastSlash+1:]

	if instanceID == "" || aiAgentArn == "" || securityProfileID == "" {
		return "", "", "", fmt.Errorf("expected format instance_id/ai_agent_arn/security_profile_id, got: %q", id)
	}

	return instanceID, aiAgentArn, securityProfileID, nil
}

// associatedAIAgentArns is every ARN of the agent the security profile is associated with:
// the agent itself, and with a version also :$LATEST, :$SAVED and :<version>.
func associatedAIAgentArns(agentArn string, version frameworktypes.Int64) []string {
	arns := []string{agentArn}
	if version.IsNull() || version.IsUnknown() {
		return arns
	}
	return append(arns,
		agentArn+":$LATEST",
		agentArn+savedQualifier,
		fmt.Sprintf("%s:%d", agentArn, version.ValueInt64()),
	)
}

const savedQualifier = ":$SAVED"

// versionArnsToCheck leaves out :$SAVED, which exists only while the agent has a saved state.
func versionArnsToCheck(agentArn string, version frameworktypes.Int64) []string {
	var arns []string
	for _, arn := range associatedAIAgentArns(agentArn, version)[1:] {
		if !strings.HasSuffix(arn, savedQualifier) {
			arns = append(arns, arn)
		}
	}
	return arns
}

func arnsMissingFrom(previous, planned []string) []string {
	kept := make(map[string]bool, len(planned))
	for _, arn := range planned {
		kept[arn] = true
	}
	var gone []string
	for _, arn := range previous {
		if !kept[arn] {
			gone = append(gone, arn)
		}
	}
	return gone
}

func containsSecurityProfile(profiles []types.SecurityProfileItem, securityProfileID string) bool {
	for _, profile := range profiles {
		if aws.ToString(profile.Id) == securityProfileID {
			return true
		}
	}
	return false
}

// associate tolerates an association that already exists, and a :$SAVED the agent does not have.
func (r *ConnectAIAgentSecurityProfileResource) associate(ctx context.Context, instanceID, entityArn, securityProfileID string) error {
	input := &connect.AssociateSecurityProfilesInput{
		EntityArn:        aws.String(entityArn),
		EntityType:       types.EntityTypeAiAgent,
		InstanceId:       aws.String(instanceID),
		SecurityProfiles: []types.SecurityProfileItem{{Id: aws.String(securityProfileID)}},
	}
	saved := strings.HasSuffix(entityArn, savedQualifier)

	err := retryOnEventualConsistency(ctx,
		func(err error) bool { return !saved && strings.Contains(err.Error(), "ResourceNotFoundException") },
		func() error {
			_, err := r.client.AssociateSecurityProfiles(ctx, input)
			return err
		},
	)
	if err == nil || isAlreadyAssociated(err) || (saved && strings.Contains(err.Error(), "ResourceNotFoundException")) {
		return nil
	}
	return err
}

func (r *ConnectAIAgentSecurityProfileResource) disassociate(ctx context.Context, instanceID, entityArn, securityProfileID string) error {
	_, err := r.client.DisassociateSecurityProfiles(ctx, &connect.DisassociateSecurityProfilesInput{
		EntityArn:        aws.String(entityArn),
		EntityType:       types.EntityTypeAiAgent,
		InstanceId:       aws.String(instanceID),
		SecurityProfiles: []types.SecurityProfileItem{{Id: aws.String(securityProfileID)}},
	})
	if err != nil && !strings.Contains(err.Error(), "ResourceNotFoundException") {
		return err
	}
	return nil
}

func isAlreadyAssociated(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already") || strings.Contains(message, "duplicate")
}
