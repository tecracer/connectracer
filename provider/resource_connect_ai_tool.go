// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/document"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectAIToolResource{}
var _ resource.ResourceWithImportState = &ConnectAIToolResource{}

func NewConnectAIToolResource() resource.Resource {
	return &ConnectAIToolResource{}
}

// ConnectAIToolResource defines the resource implementation.
type ConnectAIToolResource struct {
	client *qconnect.Client
}

// ConnectAIToolResourceModel describes the resource data model.
// The ID is "assistant_id/ai_agent_id/tool_name" to uniquely identify a tool within an agent.
type ConnectAIToolResourceModel struct {
	ID                       frameworktypes.String `tfsdk:"id"`
	AssistantID              frameworktypes.String `tfsdk:"assistant_id"`
	AIAgentID                frameworktypes.String `tfsdk:"ai_agent_id"`
	ToolName                 frameworktypes.String `tfsdk:"tool_name"`
	ToolType                 frameworktypes.String `tfsdk:"tool_type"`
	ToolID                   frameworktypes.String `tfsdk:"tool_id"`
	Title                    frameworktypes.String `tfsdk:"title"`
	Description              frameworktypes.String `tfsdk:"description"`
	InputSchemaJSON          frameworktypes.String `tfsdk:"input_schema_json"`
	OutputSchemaJSON         frameworktypes.String `tfsdk:"output_schema_json"`
	Instruction              frameworktypes.String `tfsdk:"instruction"`
	InstructionExamples      frameworktypes.List   `tfsdk:"instruction_examples"`
	UserConfirmationRequired frameworktypes.Bool   `tfsdk:"user_confirmation_required"`
}

func (r *ConnectAIToolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_ai_tool"
}

func (r *ConnectAIToolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a tool configuration within an Amazon Q in Connect Orchestration AI Agent.\n\n" +
			"Tools are identified by `tool_name` within the agent. Because the AWS API stores tools as a list on the\n" +
			"agent, this resource performs a read-modify-write on every create, update, and delete.\n\n" +
			"Use `RETURN_TO_CONTROL` as `tool_type` to implement the escalation-to-human pattern described in the\n" +
			"Amazon Connect AI Agent workshop.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier of the tool (`assistant_id/ai_agent_id/tool_name`)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"assistant_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Amazon Q in Connect assistant",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ai_agent_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Orchestration AI Agent that owns this tool",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tool_name": schema.StringAttribute{
				MarkdownDescription: "The unique name of the tool within the agent",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tool_type": schema.StringAttribute{
				MarkdownDescription: "The type of the tool. Valid values: `RETURN_TO_CONTROL`, `MODEL_CONTEXT_PROTOCOL`, `CONSTANT`",
				Required:            true,
			},
			"tool_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the underlying tool on its MCP server — required when `tool_type` " +
					"is `MODEL_CONTEXT_PROTOCOL` (`UpdateAIAgent` rejects the tool otherwise with \"require toolId as input " +
					"for MCP identifier\"). For a flow module tool (`connectracer_connect_flow_module_tool`), this is " +
					"**not** the flow module's own `id` (`UpdateAIAgent` rejects that with \"not found in MCP tools\") — " +
					"use its computed `mcp_tool_id` attribute instead.",
				Optional: true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "A short human-readable title for the tool",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of what the tool does; shown to the AI model as part of the tool contract",
				Optional:            true,
			},
			"input_schema_json": schema.StringAttribute{
				MarkdownDescription: "The input schema for the tool as a JSON string (e.g., a JSON Schema object describing the parameters the AI agent must supply when calling this tool)",
				Optional:            true,
			},
			"output_schema_json": schema.StringAttribute{
				MarkdownDescription: "The output schema for the tool as a JSON string",
				Optional:            true,
			},
			"instruction": schema.StringAttribute{
				MarkdownDescription: "Free-form instruction text telling the AI model when and how to use this tool",
				Optional:            true,
			},
			"instruction_examples": schema.ListAttribute{
				MarkdownDescription: "A list of example strings illustrating tool usage",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"user_confirmation_required": schema.BoolAttribute{
				MarkdownDescription: "Whether the user must confirm the action before the tool is executed",
				Optional:            true,
			},
		},
	}
}

func (r *ConnectAIToolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = clients.QConnect
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *ConnectAIToolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectAIToolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agent, orchConfig, err := r.getOrchestrationConfig(ctx, data.AssistantID.ValueString(), data.AIAgentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading AI Agent", err.Error())
		return
	}

	// Prevent duplicate tool names.
	for _, t := range orchConfig.ToolConfigurations {
		if aws.ToString(t.ToolName) == data.ToolName.ValueString() {
			resp.Diagnostics.AddError(
				"Tool Already Exists",
				fmt.Sprintf("A tool named %q already exists in agent %s. Import it instead of creating a new one.", data.ToolName.ValueString(), data.AIAgentID.ValueString()),
			)
			return
		}
	}

	newTool, diags := r.buildToolFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orchConfig.ToolConfigurations = append(orchConfig.ToolConfigurations, newTool)

	// UpdateAIAgent can reject a MODEL_CONTEXT_PROTOCOL tool_id as "not found in
	// MCP tools" for a few seconds/minutes after the security profile grant that
	// makes it visible to QConnect (connectracer_connect_security_profile_flow_module
	// for a flow module tool) — the same kind of eventual-consistency gap observed
	// between CreateSecurityProfile/CreateContactFlowModule and their downstream
	// consumers elsewhere in this provider.
	toolID := data.ToolID.ValueString()
	err = retryOnEventualConsistency(ctx,
		func(err error) bool {
			return toolID != "" && strings.Contains(err.Error(), "not found in MCP tools") && strings.Contains(err.Error(), toolID)
		},
		func() error {
			return r.updateAgentTools(ctx, agent, orchConfig)
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AI Tool", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(r.composeID(data.AssistantID.ValueString(), data.AIAgentID.ValueString(), data.ToolName.ValueString()))

	tflog.Trace(ctx, "Created AI Tool", map[string]any{
		"id":        data.ID.ValueString(),
		"tool_name": data.ToolName.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIToolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectAIToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, orchConfig, err := r.getOrchestrationConfig(ctx, data.AssistantID.ValueString(), data.AIAgentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading AI Agent", err.Error())
		return
	}

	tool := r.findTool(orchConfig, data.ToolName.ValueString())
	if tool == nil {
		// Tool no longer exists; remove from state.
		resp.State.RemoveResource(ctx)
		return
	}

	prior := data

	diags := r.populateModelFromTool(ctx, tool, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Description, data.Instruction, data.InstructionExamples = resolveOmittedToolFields(
		tool.Description != nil, data.Description, prior.Description,
		tool.Instruction != nil, data.Instruction, prior.Instruction,
		data.InstructionExamples, prior.InstructionExamples,
	)
	data.InputSchemaJSON = resolveOmittedStringField(tool.InputSchema != nil, data.InputSchemaJSON, prior.InputSchemaJSON)
	data.OutputSchemaJSON = resolveOmittedStringField(tool.OutputSchema != nil, data.OutputSchemaJSON, prior.OutputSchemaJSON)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// resolveOmittedToolFields decides the description/instruction/instructionExamples to keep in
// state after a refresh. GetAIAgent does not reliably return description/instruction for a
// MODEL_CONTEXT_PROTOCOL tool backed by a flow module — confirmed by a persistent plan diff where
// a description/instruction that was just applied reads back as absent on the very next refresh.
// When AWS omits a field (awsHasX is false), this falls back to the prior state's value instead
// of treating the omission as the user having cleared it, the same way version_number is
// preserved elsewhere in this provider when AWS omits it on read.
func resolveOmittedToolFields(
	awsHasDescription bool, freshDescription, priorDescription frameworktypes.String,
	awsHasInstruction bool, freshInstruction, priorInstruction frameworktypes.String,
	freshInstructionExamples, priorInstructionExamples frameworktypes.List,
) (description, instruction frameworktypes.String, instructionExamples frameworktypes.List) {
	description = freshDescription
	if !awsHasDescription && !priorDescription.IsNull() {
		description = priorDescription
	}

	instruction, instructionExamples = freshInstruction, freshInstructionExamples
	if !awsHasInstruction && !priorInstruction.IsNull() {
		instruction, instructionExamples = priorInstruction, priorInstructionExamples
	}

	return description, instruction, instructionExamples
}

// resolveOmittedStringField applies the same fallback as resolveOmittedToolFields to a single
// string field — pulled out separately rather than folded into that function because
// InputSchemaJSON and OutputSchemaJSON have no paired "examples"-style sibling to carry along.
// Confirmed live for InputSchema: GetAIAgent returned no inputSchema for a MODEL_CONTEXT_PROTOCOL
// tool moments after an UpdateAIAgent had set one successfully (the live object still had it —
// verified independently via the raw API — so this is AWS's read path being unreliable, not the
// value actually having been lost).
func resolveOmittedStringField(awsHasField bool, fresh, prior frameworktypes.String) frameworktypes.String {
	if !awsHasField && !prior.IsNull() {
		return prior
	}
	return fresh
}

func (r *ConnectAIToolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectAIToolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agent, orchConfig, err := r.getOrchestrationConfig(ctx, data.AssistantID.ValueString(), data.AIAgentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading AI Agent", err.Error())
		return
	}

	updatedTool, diags := r.buildToolFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	replaced := false
	for i, t := range orchConfig.ToolConfigurations {
		if aws.ToString(t.ToolName) == data.ToolName.ValueString() {
			orchConfig.ToolConfigurations[i] = updatedTool
			replaced = true
			break
		}
	}
	if !replaced {
		// Tool disappeared between plan and apply; add it back.
		orchConfig.ToolConfigurations = append(orchConfig.ToolConfigurations, updatedTool)
	}

	toolID := data.ToolID.ValueString()
	err = retryOnEventualConsistency(ctx,
		func(err error) bool {
			return toolID != "" && strings.Contains(err.Error(), "not found in MCP tools") && strings.Contains(err.Error(), toolID)
		},
		func() error {
			return r.updateAgentTools(ctx, agent, orchConfig)
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AI Tool", err.Error())
		return
	}

	tflog.Trace(ctx, "Updated AI Tool", map[string]any{"tool_name": data.ToolName.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIToolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectAIToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agent, orchConfig, err := r.getOrchestrationConfig(ctx, data.AssistantID.ValueString(), data.AIAgentID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			// Agent already gone; nothing to clean up.
			return
		}
		resp.Diagnostics.AddError("Error reading AI Agent", err.Error())
		return
	}

	filtered := make([]types.ToolConfiguration, 0, len(orchConfig.ToolConfigurations))
	for _, t := range orchConfig.ToolConfigurations {
		if aws.ToString(t.ToolName) != data.ToolName.ValueString() {
			filtered = append(filtered, t)
		}
	}
	orchConfig.ToolConfigurations = filtered

	if err := r.updateAgentTools(ctx, agent, orchConfig); err != nil {
		resp.Diagnostics.AddError("Error deleting AI Tool", err.Error())
		return
	}

	tflog.Trace(ctx, "Deleted AI Tool", map[string]any{"tool_name": data.ToolName.ValueString()})
}

func (r *ConnectAIToolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: "assistant_id/ai_agent_id/tool_name"
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format assistant_id/ai_agent_id/tool_name, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("assistant_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ai_agent_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tool_name"), parts[2])...)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// composeID returns the canonical composite ID for a tool.
func (r *ConnectAIToolResource) composeID(assistantID, agentID, toolName string) string {
	return assistantID + "/" + agentID + "/" + toolName
}

// getOrchestrationConfig fetches the agent and returns the mutable OrchestrationAIAgentConfiguration.
// It errors if the agent is not of ORCHESTRATION type.
func (r *ConnectAIToolResource) getOrchestrationConfig(ctx context.Context, assistantID, agentID string) (
	*qconnect.GetAIAgentOutput, *types.OrchestrationAIAgentConfiguration, error,
) {
	out, err := r.client.GetAIAgent(ctx, &qconnect.GetAIAgentInput{
		AssistantId: aws.String(assistantID),
		AiAgentId:   aws.String(agentID),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get AI Agent %s: %w", agentID, err)
	}
	if out.AiAgent == nil {
		return nil, nil, fmt.Errorf("GetAIAgent returned empty response for agent %s", agentID)
	}

	orch, ok := out.AiAgent.Configuration.(*types.AIAgentConfigurationMemberOrchestrationAIAgentConfiguration)
	if !ok {
		return nil, nil, fmt.Errorf("AI Agent %s is not of type ORCHESTRATION (got %T)", agentID, out.AiAgent.Configuration)
	}

	// Return a copy so callers can modify ToolConfigurations freely.
	cfg := orch.Value
	cfg.ToolConfigurations = sanitizePreservedTools(cfg.ToolConfigurations)
	return out, &cfg, nil
}

// updateAgentTools calls UpdateAIAgent with the supplied orchestration config.
// It preserves the agent's current visibility status and description.
func (r *ConnectAIToolResource) updateAgentTools(
	ctx context.Context,
	current *qconnect.GetAIAgentOutput,
	orchConfig *types.OrchestrationAIAgentConfiguration,
) error {
	input := &qconnect.UpdateAIAgentInput{
		AiAgentId:        current.AiAgent.AiAgentId,
		AssistantId:      current.AiAgent.AssistantId,
		VisibilityStatus: current.AiAgent.VisibilityStatus,
		Configuration: &types.AIAgentConfigurationMemberOrchestrationAIAgentConfiguration{
			Value: *orchConfig,
		},
	}
	if current.AiAgent.Description != nil {
		input.Description = current.AiAgent.Description
	}
	_, err := r.client.UpdateAIAgent(ctx, input)
	if err != nil {
		return fmt.Errorf("UpdateAIAgent failed: %w", err)
	}
	return nil
}

// findTool returns a pointer to the ToolConfiguration whose ToolName matches name, or nil.
func (r *ConnectAIToolResource) findTool(orchConfig *types.OrchestrationAIAgentConfiguration, name string) *types.ToolConfiguration {
	for i := range orchConfig.ToolConfigurations {
		if aws.ToString(orchConfig.ToolConfigurations[i].ToolName) == name {
			return &orchConfig.ToolConfigurations[i]
		}
	}
	return nil
}

// buildToolFromModel converts the Terraform model into a types.ToolConfiguration.
func (r *ConnectAIToolResource) buildToolFromModel(ctx context.Context, data *ConnectAIToolResourceModel) (types.ToolConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics
	tc := types.ToolConfiguration{
		ToolName: aws.String(data.ToolName.ValueString()),
		ToolType: types.ToolType(data.ToolType.ValueString()),
	}

	if !data.Title.IsNull() && !data.Title.IsUnknown() {
		tc.Title = aws.String(data.Title.ValueString())
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		tc.Description = aws.String(data.Description.ValueString())
	}
	if !data.ToolID.IsNull() && !data.ToolID.IsUnknown() {
		tc.ToolId = aws.String(data.ToolID.ValueString())
	}

	// InputSchema: JSON string → smithy document
	if !data.InputSchemaJSON.IsNull() && !data.InputSchemaJSON.IsUnknown() {
		var raw any
		if err := json.Unmarshal([]byte(data.InputSchemaJSON.ValueString()), &raw); err != nil {
			diags.AddError("Invalid input_schema_json", fmt.Sprintf("input_schema_json is not valid JSON: %s", err))
			return tc, diags
		}
		tc.InputSchema = document.NewLazyDocument(raw)
	}

	// OutputSchema: JSON string → smithy document
	if !data.OutputSchemaJSON.IsNull() && !data.OutputSchemaJSON.IsUnknown() {
		var raw any
		if err := json.Unmarshal([]byte(data.OutputSchemaJSON.ValueString()), &raw); err != nil {
			diags.AddError("Invalid output_schema_json", fmt.Sprintf("output_schema_json is not valid JSON: %s", err))
			return tc, diags
		}
		tc.OutputSchema = document.NewLazyDocument(raw)
	}

	// Instruction text + examples
	var instr types.ToolInstruction
	hasInstr := false
	if !data.Instruction.IsNull() && !data.Instruction.IsUnknown() {
		instr.Instruction = aws.String(data.Instruction.ValueString())
		hasInstr = true
	}
	if !data.InstructionExamples.IsNull() && !data.InstructionExamples.IsUnknown() {
		var examples []string
		data.InstructionExamples.ElementsAs(ctx, &examples, false)
		if len(examples) > 0 {
			instr.Examples = examples
			hasInstr = true
		}
	}
	if hasInstr {
		tc.Instruction = &instr
	}

	// User confirmation
	if !data.UserConfirmationRequired.IsNull() && !data.UserConfirmationRequired.IsUnknown() {
		b := data.UserConfirmationRequired.ValueBool()
		tc.UserInteractionConfiguration = &types.UserInteractionConfiguration{
			IsUserConfirmationRequired: &b,
		}
	}

	return tc, diags
}

// populateModelFromTool reads a types.ToolConfiguration back into the Terraform model.
func (r *ConnectAIToolResource) populateModelFromTool(ctx context.Context, tc *types.ToolConfiguration, data *ConnectAIToolResourceModel) diag.Diagnostics {
	data.ToolName = frameworktypes.StringPointerValue(tc.ToolName)
	data.ToolType = frameworktypes.StringValue(string(tc.ToolType))

	if tc.Title != nil {
		data.Title = frameworktypes.StringPointerValue(tc.Title)
	} else {
		data.Title = frameworktypes.StringNull()
	}

	if tc.Description != nil {
		data.Description = frameworktypes.StringPointerValue(tc.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	if tc.ToolId != nil {
		data.ToolID = frameworktypes.StringPointerValue(tc.ToolId)
	} else {
		data.ToolID = frameworktypes.StringNull()
	}

	// InputSchema → JSON string
	if tc.InputSchema != nil {
		if b, err := tc.InputSchema.MarshalSmithyDocument(); err == nil {
			data.InputSchemaJSON = frameworktypes.StringValue(string(b))
		} else {
			data.InputSchemaJSON = frameworktypes.StringNull()
		}
	} else {
		data.InputSchemaJSON = frameworktypes.StringNull()
	}

	// OutputSchema → JSON string
	if tc.OutputSchema != nil {
		if b, err := tc.OutputSchema.MarshalSmithyDocument(); err == nil {
			data.OutputSchemaJSON = frameworktypes.StringValue(string(b))
		} else {
			data.OutputSchemaJSON = frameworktypes.StringNull()
		}
	} else {
		data.OutputSchemaJSON = frameworktypes.StringNull()
	}

	// Instruction
	if tc.Instruction != nil {
		data.Instruction = frameworktypes.StringPointerValue(tc.Instruction.Instruction)
		if len(tc.Instruction.Examples) > 0 {
			list, diags := frameworktypes.ListValueFrom(ctx, frameworktypes.StringType, tc.Instruction.Examples)
			if diags.HasError() {
				return diags
			}
			data.InstructionExamples = list
		} else {
			data.InstructionExamples = frameworktypes.ListNull(frameworktypes.StringType)
		}
	} else {
		data.Instruction = frameworktypes.StringNull()
		data.InstructionExamples = frameworktypes.ListNull(frameworktypes.StringType)
	}

	// User confirmation
	if tc.UserInteractionConfiguration != nil && tc.UserInteractionConfiguration.IsUserConfirmationRequired != nil {
		data.UserConfirmationRequired = frameworktypes.BoolPointerValue(tc.UserInteractionConfiguration.IsUserConfirmationRequired)
	} else {
		data.UserConfirmationRequired = frameworktypes.BoolNull()
	}

	return diag.Diagnostics{}
}
