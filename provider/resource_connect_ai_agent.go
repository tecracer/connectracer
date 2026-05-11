// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectAIAgentResource{}
var _ resource.ResourceWithImportState = &ConnectAIAgentResource{}

func NewConnectAIAgentResource() resource.Resource {
	return &ConnectAIAgentResource{}
}

// ConnectAIAgentResource defines the resource implementation.
type ConnectAIAgentResource struct {
	client *qconnect.Client
}

// ConnectAIAgentResourceModel describes the resource data model.
type ConnectAIAgentResourceModel struct {
	ID                                frameworktypes.String                `tfsdk:"id"`
	AssistantID                       frameworktypes.String                `tfsdk:"assistant_id"`
	Name                              frameworktypes.String                `tfsdk:"name"`
	Description                       frameworktypes.String                `tfsdk:"description"`
	Type                              frameworktypes.String                `tfsdk:"type"`
	VisibilityStatus                  frameworktypes.String                `tfsdk:"visibility_status"`
	AIAgentArn                        frameworktypes.String                `tfsdk:"ai_agent_arn"`
	AssistantArn                      frameworktypes.String                `tfsdk:"assistant_arn"`
	Status                            frameworktypes.String                `tfsdk:"status"`
	Origin                            frameworktypes.String                `tfsdk:"origin"`
	ModifiedTime                      frameworktypes.String                `tfsdk:"modified_time"`
	Tags                              frameworktypes.Map                   `tfsdk:"tags"`
	CreateVersion                     frameworktypes.Bool                  `tfsdk:"create_version"`
	VersionNumber                     frameworktypes.Int64                 `tfsdk:"version_number"`
	QualifiedID                       frameworktypes.String                `tfsdk:"qualified_id"`
	AnswerRecommendationConfiguration []AnswerRecommendationConfigModel    `tfsdk:"answer_recommendation_configuration"`
	ManualSearchConfiguration         []ManualSearchConfigModel            `tfsdk:"manual_search_configuration"`
	SelfServiceConfiguration          []SelfServiceConfigModel             `tfsdk:"self_service_configuration"`
}

type AnswerRecommendationConfigModel struct {
	AnswerGenerationAIPromptId         frameworktypes.String    `tfsdk:"answer_generation_ai_prompt_id"`
	AnswerGenerationAIGuardrailId      frameworktypes.String    `tfsdk:"answer_generation_ai_guardrail_id"`
	IntentLabelingGenerationAIPromptId frameworktypes.String    `tfsdk:"intent_labeling_generation_ai_prompt_id"`
	QueryReformulationAIPromptId       frameworktypes.String    `tfsdk:"query_reformulation_ai_prompt_id"`
	Locale                             frameworktypes.String    `tfsdk:"locale"`
	AssociationConfigurations          []AssociationConfigModel `tfsdk:"association_configurations"`
}

type ManualSearchConfigModel struct {
	AnswerGenerationAIPromptId    frameworktypes.String    `tfsdk:"answer_generation_ai_prompt_id"`
	AnswerGenerationAIGuardrailId frameworktypes.String    `tfsdk:"answer_generation_ai_guardrail_id"`
	Locale                        frameworktypes.String    `tfsdk:"locale"`
	AssociationConfigurations     []AssociationConfigModel `tfsdk:"association_configurations"`
}

type SelfServiceConfigModel struct {
	SelfServiceAIGuardrailId              frameworktypes.String    `tfsdk:"self_service_ai_guardrail_id"`
	SelfServiceAnswerGenerationAIPromptId frameworktypes.String    `tfsdk:"self_service_answer_generation_ai_prompt_id"`
	SelfServicePreProcessingAIPromptId    frameworktypes.String    `tfsdk:"self_service_pre_processing_ai_prompt_id"`
	AssociationConfigurations             []AssociationConfigModel `tfsdk:"association_configurations"`
}

type AssociationConfigModel struct {
	AssociationID              frameworktypes.String        `tfsdk:"association_id"`
	AssociationType            frameworktypes.String        `tfsdk:"association_type"`
	KnowledgeBaseConfiguration []KnowledgeBaseConfigModel   `tfsdk:"knowledge_base_configuration"`
}

type KnowledgeBaseConfigModel struct {
	MaxResults                      frameworktypes.Int32  `tfsdk:"max_results"`
	OverrideKnowledgeBaseSearchType frameworktypes.String `tfsdk:"override_knowledge_base_search_type"`
}

func (r *ConnectAIAgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_ai_agent"
}

func (r *ConnectAIAgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	knowledgeBaseConfigBlock := schema.ListNestedBlock{
		MarkdownDescription: "Knowledge base association configuration data",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"max_results": schema.Int32Attribute{
					MarkdownDescription: "The maximum number of results to return from the knowledge base",
					Optional:            true,
				},
				"override_knowledge_base_search_type": schema.StringAttribute{
					MarkdownDescription: "Override the knowledge base search type. Valid values: `HYBRID`, `SEMANTIC`",
					Optional:            true,
				},
			},
		},
	}

	associationConfigBlock := schema.ListNestedBlock{
		MarkdownDescription: "Association configurations for knowledge bases",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"association_id": schema.StringAttribute{
					MarkdownDescription: "The identifier of the association (knowledge base ID)",
					Required:            true,
				},
				"association_type": schema.StringAttribute{
					MarkdownDescription: "The type of the association. Valid values: `KNOWLEDGE_BASE`",
					Required:            true,
				},
			},
			Blocks: map[string]schema.Block{
				"knowledge_base_configuration": knowledgeBaseConfigBlock,
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Q in Connect AI Agent resource.\n\n" +
			"AI Agents allow you to configure how Amazon Q in Connect processes queries " +
			"and generates answers, including manual search, answer recommendation, and self-service configurations.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the AI Agent",
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
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the AI Agent",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the AI Agent",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the AI Agent. Valid values: `MANUAL_SEARCH`, `ANSWER_RECOMMENDATION`, `SELF_SERVICE`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility_status": schema.StringAttribute{
				MarkdownDescription: "The visibility status of the AI Agent. Valid values: `SAVED`, `PUBLISHED`",
				Required:            true,
			},
			"ai_agent_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the AI Agent",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"assistant_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the Amazon Q in Connect assistant",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the AI Agent",
				Computed:            true,
			},
			"origin": schema.StringAttribute{
				MarkdownDescription: "The origin of the AI Agent",
				Computed:            true,
			},
			"modified_time": schema.StringAttribute{
				MarkdownDescription: "The time the AI Agent was last modified (RFC3339 format)",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the AI Agent",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"create_version": schema.BoolAttribute{
				MarkdownDescription: "Whether to create a version of the AI Agent after creation/update. Defaults to `false`",
				Optional:            true,
			},
			"version_number": schema.Int64Attribute{
				MarkdownDescription: "The version number of the AI Agent (populated when `create_version` is true)",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"qualified_id": schema.StringAttribute{
				MarkdownDescription: "The AI Agent ID with version qualifier appended (e.g., `id:version_number`). Use this to reference the agent from other resources",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"answer_recommendation_configuration": schema.ListNestedBlock{
				MarkdownDescription: "Configuration for answer recommendation AI agent type",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"answer_generation_ai_prompt_id": schema.StringAttribute{
							MarkdownDescription: "The AI Prompt ID for answer generation",
							Optional:            true,
						},
						"answer_generation_ai_guardrail_id": schema.StringAttribute{
							MarkdownDescription: "The AI Guardrail ID for answer generation",
							Optional:            true,
						},
						"intent_labeling_generation_ai_prompt_id": schema.StringAttribute{
							MarkdownDescription: "The AI Prompt ID for intent labeling generation",
							Optional:            true,
						},
						"query_reformulation_ai_prompt_id": schema.StringAttribute{
							MarkdownDescription: "The AI Prompt ID for query reformulation",
							Optional:            true,
						},
						"locale": schema.StringAttribute{
							MarkdownDescription: "The locale for the configuration",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"association_configurations": associationConfigBlock,
					},
				},
			},
			"manual_search_configuration": schema.ListNestedBlock{
				MarkdownDescription: "Configuration for manual search AI agent type",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"answer_generation_ai_prompt_id": schema.StringAttribute{
							MarkdownDescription: "The AI Prompt ID for answer generation",
							Optional:            true,
						},
						"answer_generation_ai_guardrail_id": schema.StringAttribute{
							MarkdownDescription: "The AI Guardrail ID for answer generation",
							Optional:            true,
						},
						"locale": schema.StringAttribute{
							MarkdownDescription: "The locale for the configuration",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"association_configurations": associationConfigBlock,
					},
				},
			},
			"self_service_configuration": schema.ListNestedBlock{
				MarkdownDescription: "Configuration for self-service AI agent type",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"self_service_ai_guardrail_id": schema.StringAttribute{
							MarkdownDescription: "The AI Guardrail ID for self-service",
							Optional:            true,
						},
						"self_service_answer_generation_ai_prompt_id": schema.StringAttribute{
							MarkdownDescription: "The AI Prompt ID for self-service answer generation",
							Optional:            true,
						},
						"self_service_pre_processing_ai_prompt_id": schema.StringAttribute{
							MarkdownDescription: "The AI Prompt ID for self-service pre-processing",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"association_configurations": associationConfigBlock,
					},
				},
			},
		},
	}
}

func (r *ConnectAIAgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectAIAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectAIAgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := &qconnect.CreateAIAgentInput{
		AssistantId:      aws.String(data.AssistantID.ValueString()),
		Name:             aws.String(data.Name.ValueString()),
		Type:             types.AIAgentType(data.Type.ValueString()),
		VisibilityStatus: types.VisibilityStatus(data.VisibilityStatus.ValueString()),
	}

	// Build configuration from nested blocks
	config := r.buildAIAgentConfiguration(&data)
	if config != nil {
		input.Configuration = config
	}

	// Optional description
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}

	// Optional tags
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make(map[string]string)
		diags := data.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Tags = tags
	}

	tflog.Debug(ctx, "Creating AI Agent", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"name":         data.Name.ValueString(),
		"type":         data.Type.ValueString(),
	})

	output, err := r.client.CreateAIAgent(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AI Agent",
			fmt.Sprintf("Unable to create AI Agent %q, got error: %s", data.Name.ValueString(), err),
		)
		return
	}

	if output.AiAgent == nil {
		resp.Diagnostics.AddError(
			"Error creating AI Agent",
			"CreateAIAgent response did not contain AI Agent data",
		)
		return
	}

	data.ID = frameworktypes.StringPointerValue(output.AiAgent.AiAgentId)
	data.AIAgentArn = frameworktypes.StringPointerValue(output.AiAgent.AiAgentArn)
	data.AssistantArn = frameworktypes.StringPointerValue(output.AiAgent.AssistantArn)

	tflog.Trace(ctx, "Created AI Agent", map[string]interface{}{
		"ai_agent_id": data.ID.ValueString(),
	})

	// Optionally create a version
	if !data.CreateVersion.IsNull() && data.CreateVersion.ValueBool() {
		versionNumber, err := r.createVersion(ctx, data.AssistantID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error creating AI Agent version",
				fmt.Sprintf("AI Agent was created but version creation failed: %s", err),
			)
		} else {
			data.VersionNumber = frameworktypes.Int64Value(versionNumber)
		}
	} else {
		data.VersionNumber = frameworktypes.Int64Null()
	}

	// Read back to populate all computed fields
	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute qualified_id (id:version_number)
	data.QualifiedID = r.computeQualifiedID(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectAIAgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AI Agent", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"ai_agent_id":  data.ID.ValueString(),
	})

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute qualified_id (id:version_number)
	data.QualifiedID = r.computeQualifiedID(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectAIAgentResourceModel
	var state ConnectAIAgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AI Agent", map[string]interface{}{
		"ai_agent_id": data.ID.ValueString(),
	})

	updateInput := &qconnect.UpdateAIAgentInput{
		AiAgentId:        aws.String(data.ID.ValueString()),
		AssistantId:      aws.String(data.AssistantID.ValueString()),
		VisibilityStatus: types.VisibilityStatus(data.VisibilityStatus.ValueString()),
	}

	// Optional description
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		updateInput.Description = aws.String(data.Description.ValueString())
	}

	// Build configuration from nested blocks
	config := r.buildAIAgentConfiguration(&data)
	if config != nil {
		updateInput.Configuration = config
	}

	_, err := r.client.UpdateAIAgent(ctx, updateInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating AI Agent",
			fmt.Sprintf("Unable to update AI Agent %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Optionally create a new version
	if !data.CreateVersion.IsNull() && data.CreateVersion.ValueBool() {
		versionNumber, err := r.createVersion(ctx, data.AssistantID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error creating AI Agent version",
				fmt.Sprintf("AI Agent was updated but version creation failed: %s", err),
			)
		} else {
			data.VersionNumber = frameworktypes.Int64Value(versionNumber)
		}
	} else {
		data.VersionNumber = frameworktypes.Int64Null()
	}

	// Read back to refresh state
	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute qualified_id (id:version_number)
	data.QualifiedID = r.computeQualifiedID(&data)

	tflog.Trace(ctx, "Updated AI Agent resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectAIAgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AI Agent", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"ai_agent_id":  data.ID.ValueString(),
	})

	input := &qconnect.DeleteAIAgentInput{
		AiAgentId:   aws.String(data.ID.ValueString()),
		AssistantId: aws.String(data.AssistantID.ValueString()),
	}

	_, err := r.client.DeleteAIAgent(ctx, input)
	if err != nil {
		// If the resource is already gone, just return
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			tflog.Debug(ctx, "AI Agent already deleted", map[string]interface{}{
				"ai_agent_id": data.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting AI Agent",
			fmt.Sprintf("Unable to delete AI Agent %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted AI Agent resource")
}

func (r *ConnectAIAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: assistant_id/ai_agent_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: assistant_id/ai_agent_id, got: %s", req.ID),
		)
		return
	}

	assistantID := parts[0]
	aiAgentID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("assistant_id"), assistantID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), aiAgentID)...)
}

// readAndPopulateModel calls GetAIAgent and populates the model with the response.
func (r *ConnectAIAgentResource) readAndPopulateModel(ctx context.Context, data *ConnectAIAgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	getInput := &qconnect.GetAIAgentInput{
		AiAgentId:   aws.String(data.ID.ValueString()),
		AssistantId: aws.String(data.AssistantID.ValueString()),
	}

	output, err := r.client.GetAIAgent(ctx, getInput)
	if err != nil {
		diags.AddError(
			"Error reading AI Agent",
			fmt.Sprintf("Unable to get AI Agent %s, got error: %s", data.ID.ValueString(), err),
		)
		return diags
	}

	if output.AiAgent == nil {
		diags.AddError(
			"Error reading AI Agent",
			"GetAIAgent response did not contain AI Agent data",
		)
		return diags
	}

	agent := output.AiAgent

	data.ID = frameworktypes.StringPointerValue(agent.AiAgentId)
	data.AIAgentArn = frameworktypes.StringPointerValue(agent.AiAgentArn)
	data.AssistantID = frameworktypes.StringPointerValue(agent.AssistantId)
	data.AssistantArn = frameworktypes.StringPointerValue(agent.AssistantArn)
	data.Name = frameworktypes.StringPointerValue(agent.Name)
	data.Type = frameworktypes.StringValue(string(agent.Type))
	data.VisibilityStatus = frameworktypes.StringValue(string(agent.VisibilityStatus))
	data.Status = frameworktypes.StringValue(string(agent.Status))
	data.Origin = frameworktypes.StringValue(string(agent.Origin))

	// Description
	if agent.Description != nil {
		data.Description = frameworktypes.StringPointerValue(agent.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	// Modified time
	if agent.ModifiedTime != nil {
		data.ModifiedTime = frameworktypes.StringValue(agent.ModifiedTime.Format(time.RFC3339))
	} else {
		data.ModifiedTime = frameworktypes.StringNull()
	}

	// Tags
	if len(agent.Tags) > 0 {
		tagsMap, tagDiags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, agent.Tags)
		diags.Append(tagDiags...)
		if diags.HasError() {
			return diags
		}
		data.Tags = tagsMap
	} else {
		data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
	}

	// Version number from response
	if output.VersionNumber != nil {
		data.VersionNumber = frameworktypes.Int64Value(*output.VersionNumber)
	}
	// Note: don't override VersionNumber if output.VersionNumber is nil,
	// because it may have been set by createVersion

	// Configuration - populate the correct nested block based on the configuration type
	r.populateConfigurationFromAPI(agent.Configuration, data)

	return diags
}

// populateConfigurationFromAPI reads the AIAgentConfiguration from the API response and
// populates the correct nested block model in the resource data.
func (r *ConnectAIAgentResource) populateConfigurationFromAPI(config types.AIAgentConfiguration, data *ConnectAIAgentResourceModel) {
	// Reset all configuration blocks
	data.AnswerRecommendationConfiguration = nil
	data.ManualSearchConfiguration = nil
	data.SelfServiceConfiguration = nil

	if config == nil {
		return
	}

	switch c := config.(type) {
	case *types.AIAgentConfigurationMemberAnswerRecommendationAIAgentConfiguration:
		model := AnswerRecommendationConfigModel{
			AnswerGenerationAIPromptId:         frameworktypes.StringPointerValue(c.Value.AnswerGenerationAIPromptId),
			AnswerGenerationAIGuardrailId:      frameworktypes.StringPointerValue(c.Value.AnswerGenerationAIGuardrailId),
			IntentLabelingGenerationAIPromptId: frameworktypes.StringPointerValue(c.Value.IntentLabelingGenerationAIPromptId),
			QueryReformulationAIPromptId:       frameworktypes.StringPointerValue(c.Value.QueryReformulationAIPromptId),
			Locale:                             frameworktypes.StringPointerValue(c.Value.Locale),
			AssociationConfigurations:          r.flattenAssociationConfigurations(c.Value.AssociationConfigurations),
		}
		data.AnswerRecommendationConfiguration = []AnswerRecommendationConfigModel{model}

	case *types.AIAgentConfigurationMemberManualSearchAIAgentConfiguration:
		model := ManualSearchConfigModel{
			AnswerGenerationAIPromptId:    frameworktypes.StringPointerValue(c.Value.AnswerGenerationAIPromptId),
			AnswerGenerationAIGuardrailId: frameworktypes.StringPointerValue(c.Value.AnswerGenerationAIGuardrailId),
			Locale:                        frameworktypes.StringPointerValue(c.Value.Locale),
			AssociationConfigurations:     r.flattenAssociationConfigurations(c.Value.AssociationConfigurations),
		}
		data.ManualSearchConfiguration = []ManualSearchConfigModel{model}

	case *types.AIAgentConfigurationMemberSelfServiceAIAgentConfiguration:
		model := SelfServiceConfigModel{
			SelfServiceAIGuardrailId:              frameworktypes.StringPointerValue(c.Value.SelfServiceAIGuardrailId),
			SelfServiceAnswerGenerationAIPromptId: frameworktypes.StringPointerValue(c.Value.SelfServiceAnswerGenerationAIPromptId),
			SelfServicePreProcessingAIPromptId:    frameworktypes.StringPointerValue(c.Value.SelfServicePreProcessingAIPromptId),
			AssociationConfigurations:             r.flattenAssociationConfigurations(c.Value.AssociationConfigurations),
		}
		data.SelfServiceConfiguration = []SelfServiceConfigModel{model}
	}
}

// flattenAssociationConfigurations converts API association configurations to the model representation.
func (r *ConnectAIAgentResource) flattenAssociationConfigurations(configs []types.AssociationConfiguration) []AssociationConfigModel {
	if len(configs) == 0 {
		return nil
	}

	result := make([]AssociationConfigModel, 0, len(configs))
	for _, ac := range configs {
		model := AssociationConfigModel{
			AssociationID:   frameworktypes.StringPointerValue(ac.AssociationId),
			AssociationType: frameworktypes.StringValue(string(ac.AssociationType)),
		}

		// Handle AssociationConfigurationData
		if ac.AssociationConfigurationData != nil {
			switch d := ac.AssociationConfigurationData.(type) {
			case *types.AssociationConfigurationDataMemberKnowledgeBaseAssociationConfigurationData:
				kbConfig := KnowledgeBaseConfigModel{
					OverrideKnowledgeBaseSearchType: frameworktypes.StringValue(string(d.Value.OverrideKnowledgeBaseSearchType)),
				}
				if d.Value.MaxResults != nil {
					kbConfig.MaxResults = frameworktypes.Int32Value(*d.Value.MaxResults)
				} else {
					kbConfig.MaxResults = frameworktypes.Int32Null()
				}
				if string(d.Value.OverrideKnowledgeBaseSearchType) == "" {
					kbConfig.OverrideKnowledgeBaseSearchType = frameworktypes.StringNull()
				}
				model.KnowledgeBaseConfiguration = []KnowledgeBaseConfigModel{kbConfig}
			}
		}

		result = append(result, model)
	}

	return result
}

// createVersion creates a version of the AI Agent and returns the version number.
func (r *ConnectAIAgentResource) createVersion(ctx context.Context, assistantID, aiAgentID string) (int64, error) {
	versionInput := &qconnect.CreateAIAgentVersionInput{
		AssistantId: aws.String(assistantID),
		AiAgentId:   aws.String(aiAgentID),
	}

	tflog.Debug(ctx, "Creating AI Agent version", map[string]interface{}{
		"assistant_id": assistantID,
		"ai_agent_id":  aiAgentID,
	})

	versionOutput, err := r.client.CreateAIAgentVersion(ctx, versionInput)
	if err != nil {
		return 0, fmt.Errorf("failed to create AI Agent version: %w", err)
	}

	var versionNumber int64
	if versionOutput.VersionNumber != nil {
		versionNumber = *versionOutput.VersionNumber
	}

	tflog.Debug(ctx, "Created AI Agent version", map[string]interface{}{
		"version_number": versionNumber,
	})

	return versionNumber, nil
}

// buildAIAgentConfiguration builds the AIAgentConfiguration from the model's nested blocks.
func (r *ConnectAIAgentResource) buildAIAgentConfiguration(data *ConnectAIAgentResourceModel) types.AIAgentConfiguration {
	if len(data.AnswerRecommendationConfiguration) > 0 {
		cfg := data.AnswerRecommendationConfiguration[0]
		value := types.AnswerRecommendationAIAgentConfiguration{
			AssociationConfigurations: r.expandAssociationConfigurations(cfg.AssociationConfigurations),
		}

		if !cfg.AnswerGenerationAIPromptId.IsNull() && !cfg.AnswerGenerationAIPromptId.IsUnknown() {
			value.AnswerGenerationAIPromptId = aws.String(cfg.AnswerGenerationAIPromptId.ValueString())
		}
		if !cfg.AnswerGenerationAIGuardrailId.IsNull() && !cfg.AnswerGenerationAIGuardrailId.IsUnknown() {
			value.AnswerGenerationAIGuardrailId = aws.String(cfg.AnswerGenerationAIGuardrailId.ValueString())
		}
		if !cfg.IntentLabelingGenerationAIPromptId.IsNull() && !cfg.IntentLabelingGenerationAIPromptId.IsUnknown() {
			value.IntentLabelingGenerationAIPromptId = aws.String(cfg.IntentLabelingGenerationAIPromptId.ValueString())
		}
		if !cfg.QueryReformulationAIPromptId.IsNull() && !cfg.QueryReformulationAIPromptId.IsUnknown() {
			value.QueryReformulationAIPromptId = aws.String(cfg.QueryReformulationAIPromptId.ValueString())
		}
		if !cfg.Locale.IsNull() && !cfg.Locale.IsUnknown() {
			value.Locale = aws.String(cfg.Locale.ValueString())
		}

		return &types.AIAgentConfigurationMemberAnswerRecommendationAIAgentConfiguration{
			Value: value,
		}
	}

	if len(data.ManualSearchConfiguration) > 0 {
		cfg := data.ManualSearchConfiguration[0]
		value := types.ManualSearchAIAgentConfiguration{
			AssociationConfigurations: r.expandAssociationConfigurations(cfg.AssociationConfigurations),
		}

		if !cfg.AnswerGenerationAIPromptId.IsNull() && !cfg.AnswerGenerationAIPromptId.IsUnknown() {
			value.AnswerGenerationAIPromptId = aws.String(cfg.AnswerGenerationAIPromptId.ValueString())
		}
		if !cfg.AnswerGenerationAIGuardrailId.IsNull() && !cfg.AnswerGenerationAIGuardrailId.IsUnknown() {
			value.AnswerGenerationAIGuardrailId = aws.String(cfg.AnswerGenerationAIGuardrailId.ValueString())
		}
		if !cfg.Locale.IsNull() && !cfg.Locale.IsUnknown() {
			value.Locale = aws.String(cfg.Locale.ValueString())
		}

		return &types.AIAgentConfigurationMemberManualSearchAIAgentConfiguration{
			Value: value,
		}
	}

	if len(data.SelfServiceConfiguration) > 0 {
		cfg := data.SelfServiceConfiguration[0]
		value := types.SelfServiceAIAgentConfiguration{
			AssociationConfigurations: r.expandAssociationConfigurations(cfg.AssociationConfigurations),
		}

		if !cfg.SelfServiceAIGuardrailId.IsNull() && !cfg.SelfServiceAIGuardrailId.IsUnknown() {
			value.SelfServiceAIGuardrailId = aws.String(cfg.SelfServiceAIGuardrailId.ValueString())
		}
		if !cfg.SelfServiceAnswerGenerationAIPromptId.IsNull() && !cfg.SelfServiceAnswerGenerationAIPromptId.IsUnknown() {
			value.SelfServiceAnswerGenerationAIPromptId = aws.String(cfg.SelfServiceAnswerGenerationAIPromptId.ValueString())
		}
		if !cfg.SelfServicePreProcessingAIPromptId.IsNull() && !cfg.SelfServicePreProcessingAIPromptId.IsUnknown() {
			value.SelfServicePreProcessingAIPromptId = aws.String(cfg.SelfServicePreProcessingAIPromptId.ValueString())
		}

		return &types.AIAgentConfigurationMemberSelfServiceAIAgentConfiguration{
			Value: value,
		}
	}

	return nil
}

// expandAssociationConfigurations converts model association configurations to API types.
func (r *ConnectAIAgentResource) expandAssociationConfigurations(models []AssociationConfigModel) []types.AssociationConfiguration {
	if len(models) == 0 {
		return nil
	}

	result := make([]types.AssociationConfiguration, 0, len(models))
	for _, m := range models {
		ac := types.AssociationConfiguration{
			AssociationId:   aws.String(m.AssociationID.ValueString()),
			AssociationType: types.AIAgentAssociationConfigurationType(m.AssociationType.ValueString()),
		}

		// Handle knowledge base configuration data
		if len(m.KnowledgeBaseConfiguration) > 0 {
			kbCfg := m.KnowledgeBaseConfiguration[0]
			kbData := types.KnowledgeBaseAssociationConfigurationData{}

			if !kbCfg.MaxResults.IsNull() && !kbCfg.MaxResults.IsUnknown() {
				val := kbCfg.MaxResults.ValueInt32()
				kbData.MaxResults = &val
			}
			if !kbCfg.OverrideKnowledgeBaseSearchType.IsNull() && !kbCfg.OverrideKnowledgeBaseSearchType.IsUnknown() {
				kbData.OverrideKnowledgeBaseSearchType = types.KnowledgeBaseSearchType(kbCfg.OverrideKnowledgeBaseSearchType.ValueString())
			}

			ac.AssociationConfigurationData = &types.AssociationConfigurationDataMemberKnowledgeBaseAssociationConfigurationData{
				Value: kbData,
			}
		}

		result = append(result, ac)
	}

	return result
}

// computeQualifiedID computes the qualified ID (id:version_number) for referencing
// the AI Agent from other resources.
func (r *ConnectAIAgentResource) computeQualifiedID(data *ConnectAIAgentResourceModel) frameworktypes.String {
	if data.ID.IsNull() || data.ID.IsUnknown() {
		return frameworktypes.StringNull()
	}
	if data.VersionNumber.IsNull() || data.VersionNumber.IsUnknown() {
		// No version available, return just the ID
		return data.ID
	}
	qualified := fmt.Sprintf("%s:%d", data.ID.ValueString(), data.VersionNumber.ValueInt64())
	return frameworktypes.StringValue(qualified)
}
