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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectAIPromptResource{}
var _ resource.ResourceWithImportState = &ConnectAIPromptResource{}

func NewConnectAIPromptResource() resource.Resource {
	return &ConnectAIPromptResource{}
}

// ConnectAIPromptResource defines the resource implementation.
type ConnectAIPromptResource struct {
	client *qconnect.Client
}

// ConnectAIPromptResourceModel describes the resource data model.
type ConnectAIPromptResourceModel struct {
	ID                     frameworktypes.String  `tfsdk:"id"`
	AssistantID            frameworktypes.String  `tfsdk:"assistant_id"`
	Name                   frameworktypes.String  `tfsdk:"name"`
	Description            frameworktypes.String  `tfsdk:"description"`
	Type                   frameworktypes.String  `tfsdk:"type"`
	TemplateType           frameworktypes.String  `tfsdk:"template_type"`
	ModelID                frameworktypes.String  `tfsdk:"model_id"`
	APIFormat              frameworktypes.String  `tfsdk:"api_format"`
	VisibilityStatus       frameworktypes.String  `tfsdk:"visibility_status"`
	TemplateText           frameworktypes.String  `tfsdk:"template_text"`
	AIPromptArn            frameworktypes.String  `tfsdk:"ai_prompt_arn"`
	AssistantArn           frameworktypes.String  `tfsdk:"assistant_arn"`
	Status                 frameworktypes.String  `tfsdk:"status"`
	ModifiedTime           frameworktypes.String  `tfsdk:"modified_time"`
	Tags                   frameworktypes.Map     `tfsdk:"tags"`
	CreateVersion          frameworktypes.Bool    `tfsdk:"create_version"`
	VersionNumber          frameworktypes.Int64   `tfsdk:"version_number"`
	QualifiedID            frameworktypes.String  `tfsdk:"qualified_id"`
	MaxTokensToSample      frameworktypes.Int32   `tfsdk:"max_tokens_to_sample"`
	Temperature            frameworktypes.Float32 `tfsdk:"temperature"`
	TopK                   frameworktypes.Int32   `tfsdk:"top_k"`
	TopP                   frameworktypes.Float32 `tfsdk:"top_p"`
}

func (r *ConnectAIPromptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_ai_prompt"
}

func (r *ConnectAIPromptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Q in Connect AI Prompt resource.\n\n" +
			"AI Prompts allow you to configure custom prompts for Amazon Q in Connect, " +
			"controlling how the AI generates responses for different use cases such as " +
			"answer generation, intent labeling, query reformulation, and self-service pre-processing.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the AI Prompt",
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
				MarkdownDescription: "The name of the AI Prompt",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the AI Prompt",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the AI Prompt. Valid values: `ANSWER_GENERATION`, `INTENT_LABELING_GENERATION`, `QUERY_REFORMULATION`, `SELF_SERVICE_PRE_PROCESSING`, `SELF_SERVICE_ANSWER_GENERATION`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_type": schema.StringAttribute{
				MarkdownDescription: "The type of the prompt template. Valid values: `TEXT`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"model_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the model used for this AI Prompt (e.g. `anthropic.claude-3-haiku--v1:0`, `us.anthropic.claude-3-7-sonnet-20250219-v1:0`)",
				Required:            true,
			},
			"api_format": schema.StringAttribute{
				MarkdownDescription: "The API format of the AI Prompt. Valid values: `MESSAGES`, `TEXT_COMPLETIONS`. The legacy values `ANTHROPIC_CLAUDE_MESSAGES` and `ANTHROPIC_CLAUDE_TEXT_COMPLETIONS` are deprecated",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility_status": schema.StringAttribute{
				MarkdownDescription: "The visibility status of the AI Prompt. Valid values: `SAVED`, `PUBLISHED`",
				Required:            true,
			},
			"template_text": schema.StringAttribute{
				MarkdownDescription: "The text content of the prompt template (supports YAML prompt format)",
				Required:            true,
			},
			"ai_prompt_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the AI Prompt",
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
				MarkdownDescription: "The status of the AI Prompt",
				Computed:            true,
			},
			"modified_time": schema.StringAttribute{
				MarkdownDescription: "The time the AI Prompt was last modified (RFC3339 format)",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the AI Prompt",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"create_version": schema.BoolAttribute{
				MarkdownDescription: "Whether to create a version of the AI Prompt after creation/update. Defaults to `false`",
				Optional:            true,
			},
			"version_number": schema.Int64Attribute{
				MarkdownDescription: "The version number of the AI Prompt (populated when `create_version` is true)",
				Computed:            true,
			},
			"qualified_id": schema.StringAttribute{
				MarkdownDescription: "The AI Prompt ID with version qualifier appended (e.g., `id:version_number`). Use this to reference the prompt from AI Agent resources",
				Computed:            true,
			},
			"max_tokens_to_sample": schema.Int32Attribute{
				MarkdownDescription: "The maximum number of tokens to generate in the response (inference configuration)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"temperature": schema.Float32Attribute{
				MarkdownDescription: "The temperature setting for controlling randomness in the generated response (inference configuration)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Float32{
					float32planmodifier.UseStateForUnknown(),
				},
			},
			"top_k": schema.Int32Attribute{
				MarkdownDescription: "The top-K sampling parameter for token selection (inference configuration)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"top_p": schema.Float32Attribute{
				MarkdownDescription: "The top-P sampling parameter for nucleus sampling (inference configuration)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Float32{
					float32planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ConnectAIPromptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func trimTrailingWhitespace(content string) string {
  return strings.TrimRight(content, " \t\n\r")
}

func (r *ConnectAIPromptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectAIPromptResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	templateText := trimTrailingWhitespace(data.TemplateText.ValueString())

	input := &qconnect.CreateAIPromptInput{
		AssistantId:      aws.String(data.AssistantID.ValueString()),
		Name:             aws.String(data.Name.ValueString()),
		Type:             types.AIPromptType(data.Type.ValueString()),
		TemplateType:     types.AIPromptTemplateType(data.TemplateType.ValueString()),
		ModelId:          aws.String(data.ModelID.ValueString()),
		ApiFormat:        types.AIPromptAPIFormat(data.APIFormat.ValueString()),
		VisibilityStatus: types.VisibilityStatus(data.VisibilityStatus.ValueString()),
		TemplateConfiguration: &types.AIPromptTemplateConfigurationMemberTextFullAIPromptEditTemplateConfiguration{
			Value: types.TextFullAIPromptEditTemplateConfiguration{
				Text: aws.String(templateText),
			},
		},
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

	// Optional inference configuration
	inferenceConfig := r.buildInferenceConfiguration(&data)
	if inferenceConfig != nil {
		input.InferenceConfiguration = inferenceConfig
	}

	tflog.Debug(ctx, "Creating AI Prompt", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"name":         data.Name.ValueString(),
		"type":         data.Type.ValueString(),
	})

	output, err := r.client.CreateAIPrompt(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AI Prompt",
			fmt.Sprintf("Unable to create AI Prompt %q, got error: %s", data.Name.ValueString(), err),
		)
		return
	}

	if output.AiPrompt == nil {
		resp.Diagnostics.AddError(
			"Error creating AI Prompt",
			"CreateAIPrompt response did not contain AI Prompt data",
		)
		return
	}

	data.ID = frameworktypes.StringPointerValue(output.AiPrompt.AiPromptId)
	data.AIPromptArn = frameworktypes.StringPointerValue(output.AiPrompt.AiPromptArn)
	data.AssistantArn = frameworktypes.StringPointerValue(output.AiPrompt.AssistantArn)

	tflog.Trace(ctx, "Created AI Prompt", map[string]interface{}{
		"ai_prompt_id": data.ID.ValueString(),
	})

	// Optionally create a version
	if !data.CreateVersion.IsNull() && data.CreateVersion.ValueBool() {
		versionNumber, err := r.createVersion(ctx, data.AssistantID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error creating AI Prompt version",
				fmt.Sprintf("AI Prompt was created but version creation failed: %s", err),
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

func (r *ConnectAIPromptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectAIPromptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AI Prompt", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"ai_prompt_id": data.ID.ValueString(),
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

func (r *ConnectAIPromptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectAIPromptResourceModel
	var state ConnectAIPromptResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AI Prompt", map[string]interface{}{
		"ai_prompt_id": data.ID.ValueString(),
	})

	templateText := strings.TrimRight(data.TemplateText.ValueString(), " \t\n\r")

	updateInput := &qconnect.UpdateAIPromptInput{
		AiPromptId:       aws.String(data.ID.ValueString()),
		AssistantId:      aws.String(data.AssistantID.ValueString()),
		VisibilityStatus: types.VisibilityStatus(data.VisibilityStatus.ValueString()),
		TemplateConfiguration: &types.AIPromptTemplateConfigurationMemberTextFullAIPromptEditTemplateConfiguration{
			Value: types.TextFullAIPromptEditTemplateConfiguration{
				Text: aws.String(templateText),
			},
		},
	}

	// Optional description
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		updateInput.Description = aws.String(data.Description.ValueString())
	}

	// Optional model_id update
	if !data.ModelID.IsNull() && !data.ModelID.IsUnknown() {
		updateInput.ModelId = aws.String(data.ModelID.ValueString())
	}

	// Optional inference configuration
	inferenceConfig := r.buildInferenceConfiguration(&data)
	if inferenceConfig != nil {
		updateInput.InferenceConfiguration = inferenceConfig
	}

	_, err := r.client.UpdateAIPrompt(ctx, updateInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating AI Prompt",
			fmt.Sprintf("Unable to update AI Prompt %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Optionally create a new version
	if !data.CreateVersion.IsNull() && data.CreateVersion.ValueBool() {
		versionNumber, err := r.createVersion(ctx, data.AssistantID.ValueString(), data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error creating AI Prompt version",
				fmt.Sprintf("AI Prompt was updated but version creation failed: %s", err),
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

	tflog.Trace(ctx, "Updated AI Prompt resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAIPromptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectAIPromptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AI Prompt", map[string]interface{}{
		"assistant_id": data.AssistantID.ValueString(),
		"ai_prompt_id": data.ID.ValueString(),
	})

	input := &qconnect.DeleteAIPromptInput{
		AiPromptId:  aws.String(data.ID.ValueString()),
		AssistantId: aws.String(data.AssistantID.ValueString()),
	}

	_, err := r.client.DeleteAIPrompt(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting AI Prompt",
			fmt.Sprintf("Unable to delete AI Prompt %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted AI Prompt resource")
}

func (r *ConnectAIPromptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: assistant_id/ai_prompt_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: assistant_id/ai_prompt_id, got: %s", req.ID),
		)
		return
	}

	assistantID := parts[0]
	aiPromptID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("assistant_id"), assistantID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), aiPromptID)...)
}

// readAndPopulateModel calls GetAIPrompt and populates the model with the response.
func (r *ConnectAIPromptResource) readAndPopulateModel(ctx context.Context, data *ConnectAIPromptResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	getInput := &qconnect.GetAIPromptInput{
		AiPromptId:  aws.String(data.ID.ValueString()),
		AssistantId: aws.String(data.AssistantID.ValueString()),
	}

	output, err := r.client.GetAIPrompt(ctx, getInput)
	if err != nil {
		diags.AddError(
			"Error reading AI Prompt",
			fmt.Sprintf("Unable to get AI Prompt %s, got error: %s", data.ID.ValueString(), err),
		)
		return diags
	}

	if output.AiPrompt == nil {
		diags.AddError(
			"Error reading AI Prompt",
			"GetAIPrompt response did not contain AI Prompt data",
		)
		return diags
	}

	prompt := output.AiPrompt

	data.ID = frameworktypes.StringPointerValue(prompt.AiPromptId)
	data.AIPromptArn = frameworktypes.StringPointerValue(prompt.AiPromptArn)
	data.AssistantID = frameworktypes.StringPointerValue(prompt.AssistantId)
	data.AssistantArn = frameworktypes.StringPointerValue(prompt.AssistantArn)
	data.Name = frameworktypes.StringPointerValue(prompt.Name)
	data.ModelID = frameworktypes.StringPointerValue(prompt.ModelId)
	data.Type = frameworktypes.StringValue(string(prompt.Type))
	data.TemplateType = frameworktypes.StringValue(string(prompt.TemplateType))
	data.APIFormat = frameworktypes.StringValue(string(prompt.ApiFormat))
	data.VisibilityStatus = frameworktypes.StringValue(string(prompt.VisibilityStatus))
	data.Status = frameworktypes.StringValue(string(prompt.Status))

	// Description
	if prompt.Description != nil {
		data.Description = frameworktypes.StringPointerValue(prompt.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	// Modified time
	if prompt.ModifiedTime != nil {
		data.ModifiedTime = frameworktypes.StringValue(prompt.ModifiedTime.Format(time.RFC3339))
	} else {
		data.ModifiedTime = frameworktypes.StringNull()
	}

	// Template text - extract from the template configuration
	if prompt.TemplateConfiguration != nil {
		switch tc := prompt.TemplateConfiguration.(type) {
		case *types.AIPromptTemplateConfigurationMemberTextFullAIPromptEditTemplateConfiguration:
			if tc.Value.Text != nil {
				apiValue := *tc.Value.Text
				// Compare trimmed versions: if they match, keep the original plan/state value
				// to avoid "inconsistent result after apply" errors caused by trailing whitespace differences
				if strings.TrimRight(apiValue, " \t\n\r") != strings.TrimRight(data.TemplateText.ValueString(), " \t\n\r") {
					data.TemplateText = frameworktypes.StringValue(apiValue)
				}
				// else: keep data.TemplateText as-is (preserves the user's config value)
			}
		}
	}

	// Inference configuration
	if prompt.InferenceConfiguration != nil {
		ic := prompt.InferenceConfiguration
		if ic.MaxTokensToSample != nil {
			data.MaxTokensToSample = frameworktypes.Int32Value(*ic.MaxTokensToSample)
		} else {
			data.MaxTokensToSample = frameworktypes.Int32Null()
		}
		if ic.Temperature != nil {
			data.Temperature = frameworktypes.Float32Value(*ic.Temperature)
		} else {
			data.Temperature = frameworktypes.Float32Null()
		}
		if ic.TopK != nil {
			data.TopK = frameworktypes.Int32Value(*ic.TopK)
		} else {
			data.TopK = frameworktypes.Int32Null()
		}
		if ic.TopP != nil {
			data.TopP = frameworktypes.Float32Value(*ic.TopP)
		} else {
			data.TopP = frameworktypes.Float32Null()
		}
	} else {
		data.MaxTokensToSample = frameworktypes.Int32Null()
		data.Temperature = frameworktypes.Float32Null()
		data.TopK = frameworktypes.Int32Null()
		data.TopP = frameworktypes.Float32Null()
	}

	// Tags
	if len(prompt.Tags) > 0 {
		tagsMap, tagDiags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, prompt.Tags)
		diags.Append(tagDiags...)
		if diags.HasError() {
			return diags
		}
		data.Tags = tagsMap
	} else {
		data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
	}

	return diags
}

// createVersion creates a version of the AI Prompt and returns the version number.
func (r *ConnectAIPromptResource) createVersion(ctx context.Context, assistantID, aiPromptID string) (int64, error) {
	versionInput := &qconnect.CreateAIPromptVersionInput{
		AssistantId: aws.String(assistantID),
		AiPromptId:  aws.String(aiPromptID),
	}

	tflog.Debug(ctx, "Creating AI Prompt version", map[string]interface{}{
		"assistant_id": assistantID,
		"ai_prompt_id": aiPromptID,
	})

	versionOutput, err := r.client.CreateAIPromptVersion(ctx, versionInput)
	if err != nil {
		return 0, fmt.Errorf("failed to create AI Prompt version: %w", err)
	}

	var versionNumber int64
	if versionOutput.VersionNumber != nil {
		versionNumber = *versionOutput.VersionNumber
	}

	tflog.Debug(ctx, "Created AI Prompt version", map[string]interface{}{
		"version_number": versionNumber,
	})

	return versionNumber, nil
}

// computeQualifiedID computes the qualified ID (id:version_number) for referencing
// the AI Prompt from other resources like AI Agents.
func (r *ConnectAIPromptResource) computeQualifiedID(data *ConnectAIPromptResourceModel) frameworktypes.String {
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

// buildInferenceConfiguration builds the inference configuration from the model.
func (r *ConnectAIPromptResource) buildInferenceConfiguration(data *ConnectAIPromptResourceModel) *types.AIPromptInferenceConfiguration {
	hasConfig := false
	config := &types.AIPromptInferenceConfiguration{}

	if !data.MaxTokensToSample.IsNull() && !data.MaxTokensToSample.IsUnknown() {
		val := data.MaxTokensToSample.ValueInt32()
		config.MaxTokensToSample = &val
		hasConfig = true
	}

	if !data.Temperature.IsNull() && !data.Temperature.IsUnknown() {
		val := data.Temperature.ValueFloat32()
		config.Temperature = &val
		hasConfig = true
	}

	if !data.TopK.IsNull() && !data.TopK.IsUnknown() {
		val := data.TopK.ValueInt32()
		config.TopK = &val
		hasConfig = true
	}

	if !data.TopP.IsNull() && !data.TopP.IsUnknown() {
		val := data.TopP.ValueFloat32()
		config.TopP = &val
		hasConfig = true
	}

	if !hasConfig {
		return nil
	}

	return config
}
