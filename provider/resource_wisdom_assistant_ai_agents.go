// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
	qconnecttypes "github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WisdomAssistantAIAgentsResource{}
var _ resource.ResourceWithImportState = &WisdomAssistantAIAgentsResource{}

func NewWisdomAssistantAIAgentsResource() resource.Resource {
	return &WisdomAssistantAIAgentsResource{}
}

// WisdomAssistantAIAgentsResource defines the resource implementation.
type WisdomAssistantAIAgentsResource struct {
	client *qconnect.Client
}

// WisdomAssistantAIAgentsResourceModel describes the resource data model.
type WisdomAssistantAIAgentsResourceModel struct {
	AssistantID          frameworktypes.String `tfsdk:"assistant_id"`
	AIAgentConfiguration frameworktypes.Map    `tfsdk:"ai_agent_configuration"`
}

func (r *WisdomAssistantAIAgentsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wisdom_assistant_ai_agents"
}

func (r *WisdomAssistantAIAgentsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the AI Agent configuration for an existing Amazon Q in Connect (Wisdom) Assistant. This resource manages AI Agent assignments separately from the assistant itself to avoid dependency cycles.",

		Attributes: map[string]schema.Attribute{
			"assistant_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Amazon Q in Connect assistant.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ai_agent_configuration": schema.MapAttribute{
				MarkdownDescription: "The AI Agent configuration for the assistant. A map of AI Agent type to the qualified AI Agent ID " +
					"(with version qualifier, e.g., `agent_id:$LATEST` or `agent_id:version_number`). " +
					"Valid keys: `ANSWER_RECOMMENDATION`, `MANUAL_SEARCH`, `SELF_SERVICE`",
				Required:    true,
				ElementType: frameworktypes.StringType,
			},
		},
	}
}

func (r *WisdomAssistantAIAgentsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WisdomAssistantAIAgentsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WisdomAssistantAIAgentsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agentConfig := make(map[string]string)
	diags := data.AIAgentConfiguration.ElementsAs(ctx, &agentConfig, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"agent_count":  len(agentConfig),
	})

	for agentType, agentID := range agentConfig {
		_, err := r.client.UpdateAssistantAIAgent(ctx, &qconnect.UpdateAssistantAIAgentInput{
			AssistantId: aws.String(data.AssistantID.ValueString()),
			AiAgentType: qconnecttypes.AIAgentType(agentType),
			Configuration: &qconnecttypes.AIAgentConfigurationData{
				AiAgentId: aws.String(agentID),
			},
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error configuring AI Agent",
				fmt.Sprintf("Unable to set AI Agent %q for type %q on assistant %s: %s", agentID, agentType, data.AssistantID.ValueString(), err),
			)
			return
		}
		tflog.Debug(ctx, "Configured AI Agent on assistant", map[string]interface{}{
			"assistant_id": data.AssistantID.ValueString(),
			"agent_type":   agentType,
			"agent_id":     agentID,
		})
	}

	tflog.Trace(ctx, "Created Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantAIAgentsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WisdomAssistantAIAgentsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
	})

	result, err := r.client.GetAssistant(ctx, &qconnect.GetAssistantInput{
		AssistantId: aws.String(data.AssistantID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read assistant AI agent configuration, got error: %s", err),
		)
		return
	}

	if result.Assistant != nil && len(result.Assistant.AiAgentConfiguration) > 0 {
		// Only include agent types that the user has configured to avoid drift from system-managed agents
		existingConfig := make(map[string]string)
		if !data.AIAgentConfiguration.IsNull() && !data.AIAgentConfiguration.IsUnknown() {
			data.AIAgentConfiguration.ElementsAs(ctx, &existingConfig, false)
		}

		configuredTypes := make(map[string]string)
		for agentType, agentData := range result.Assistant.AiAgentConfiguration {
			// Only track agent types that the user has configured
			if _, userManages := existingConfig[agentType]; !userManages {
				continue
			}
			if agentData.AiAgentId != nil {
				configuredTypes[agentType] = *agentData.AiAgentId
			}
		}

		if len(configuredTypes) > 0 {
			agentConfigMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, configuredTypes)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.AIAgentConfiguration = agentConfigMap
		} else {
			data.AIAgentConfiguration = frameworktypes.MapNull(frameworktypes.StringType)
		}
	} else {
		data.AIAgentConfiguration = frameworktypes.MapNull(frameworktypes.StringType)
	}

	tflog.Trace(ctx, "Read Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantAIAgentsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WisdomAssistantAIAgentsResourceModel
	var state WisdomAssistantAIAgentsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldAgentConfig := make(map[string]string)
	newAgentConfig := make(map[string]string)

	if !state.AIAgentConfiguration.IsNull() && !state.AIAgentConfiguration.IsUnknown() {
		diags := state.AIAgentConfiguration.ElementsAs(ctx, &oldAgentConfig, false)
		resp.Diagnostics.Append(diags...)
	}
	if !data.AIAgentConfiguration.IsNull() && !data.AIAgentConfiguration.IsUnknown() {
		diags := data.AIAgentConfiguration.ElementsAs(ctx, &newAgentConfig, false)
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"old_count":    len(oldAgentConfig),
		"new_count":    len(newAgentConfig),
	})

	// Remove agent types that are no longer in the config
	for agentType := range oldAgentConfig {
		if _, exists := newAgentConfig[agentType]; !exists {
			_, err := r.client.RemoveAssistantAIAgent(ctx, &qconnect.RemoveAssistantAIAgentInput{
				AssistantId: aws.String(data.AssistantID.ValueString()),
				AiAgentType: qconnecttypes.AIAgentType(agentType),
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing AI Agent",
					fmt.Sprintf("Unable to remove AI Agent for type %q from assistant %s: %s", agentType, data.AssistantID.ValueString(), err),
				)
				return
			}
			tflog.Debug(ctx, "Removed AI Agent from assistant", map[string]interface{}{
				"assistant_id": data.AssistantID.ValueString(),
				"agent_type":   agentType,
			})
		}
	}

	// Add or update agent types
	for agentType, agentID := range newAgentConfig {
		if oldID, exists := oldAgentConfig[agentType]; !exists || oldID != agentID {
			_, err := r.client.UpdateAssistantAIAgent(ctx, &qconnect.UpdateAssistantAIAgentInput{
				AssistantId: aws.String(data.AssistantID.ValueString()),
				AiAgentType: qconnecttypes.AIAgentType(agentType),
				Configuration: &qconnecttypes.AIAgentConfigurationData{
					AiAgentId: aws.String(agentID),
				},
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Error configuring AI Agent",
					fmt.Sprintf("Unable to set AI Agent %q for type %q on assistant %s: %s", agentID, agentType, data.AssistantID.ValueString(), err),
				)
				return
			}
			tflog.Debug(ctx, "Updated AI Agent on assistant", map[string]interface{}{
				"assistant_id": data.AssistantID.ValueString(),
				"agent_type":   agentType,
				"agent_id":     agentID,
			})
		}
	}

	tflog.Trace(ctx, "Updated Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantAIAgentsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WisdomAssistantAIAgentsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agentConfig := make(map[string]string)
	if !data.AIAgentConfiguration.IsNull() && !data.AIAgentConfiguration.IsUnknown() {
		diags := data.AIAgentConfiguration.ElementsAs(ctx, &agentConfig, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Debug(ctx, "Deleting Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"agent_count":  len(agentConfig),
	})

	for agentType := range agentConfig {
		_, err := r.client.RemoveAssistantAIAgent(ctx, &qconnect.RemoveAssistantAIAgentInput{
			AssistantId: aws.String(data.AssistantID.ValueString()),
			AiAgentType: qconnecttypes.AIAgentType(agentType),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing AI Agent",
				fmt.Sprintf("Unable to remove AI Agent for type %q from assistant %s: %s", agentType, data.AssistantID.ValueString(), err),
			)
			return
		}
		tflog.Debug(ctx, "Removed AI Agent from assistant", map[string]interface{}{
			"assistant_id": data.AssistantID.ValueString(),
			"agent_type":   agentType,
		})
	}

	tflog.Trace(ctx, "Deleted Wisdom Assistant AI Agents configuration", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
	})
}

func (r *WisdomAssistantAIAgentsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("assistant_id"), req, resp)
}
