// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wisdom"
	"github.com/aws/aws-sdk-go-v2/service/wisdom/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WisdomAssistantResource{}
var _ resource.ResourceWithImportState = &WisdomAssistantResource{}

func NewWisdomAssistantResource() resource.Resource {
	return &WisdomAssistantResource{}
}

// WisdomAssistantResource defines the resource implementation.
type WisdomAssistantResource struct {
	client *wisdom.Client
}

// WisdomAssistantResourceModel describes the resource data model.
type WisdomAssistantResourceModel struct {
	ID           frameworktypes.String `tfsdk:"id"`
	AssistantArn frameworktypes.String `tfsdk:"assistant_arn"`
	Name         frameworktypes.String `tfsdk:"name"`
	Type         frameworktypes.String `tfsdk:"type"`
	Description  frameworktypes.String `tfsdk:"description"`
	Tags         frameworktypes.Map    `tfsdk:"tags"`
}

func (r *WisdomAssistantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wisdom_assistant"
}

func (r *WisdomAssistantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS Wisdom Assistant. Automatically sets the required `AmazonConnectEnabled = \"True\"` tag.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the assistant",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"assistant_arn": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ARN of the assistant",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the assistant",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the assistant (AGENT)",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the assistant",
				Optional:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the assistant. The `AmazonConnectEnabled = \"True\"` tag is automatically added if not present.",
				Optional:            true,
				Computed:            true,
				ElementType:         frameworktypes.StringType,
				PlanModifiers: []planmodifier.Map{
					&ensureConnectEnabledTagModifier{},
				},
			},
		},
	}
}

func (r *WisdomAssistantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.Wisdom
}

func (r *WisdomAssistantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WisdomAssistantResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure required tags
	tags, err := ensureRequiredTags(ctx, data.Tags)
	if err != nil {
		resp.Diagnostics.AddError("Tag Error", fmt.Sprintf("Unable to process tags: %s", err))
		return
	}

	// Create the assistant
	input := &wisdom.CreateAssistantInput{
		Name: aws.String(data.Name.ValueString()),
		Type: types.AssistantType(data.Type.ValueString()),
		Tags: tags,
	}

	if !data.Description.IsNull() {
		input.Description = aws.String(data.Description.ValueString())
	}

	tflog.Debug(ctx, "Creating Wisdom Assistant", map[string]interface{}{
		"name": data.Name.ValueString(),
		"type": data.Type.ValueString(),
	})

	result, err := r.client.CreateAssistant(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create assistant, got error: %s", err),
		)
		return
	}

	// Update model with response
	data.ID = frameworktypes.StringValue(*result.Assistant.AssistantId)
	data.AssistantArn = frameworktypes.StringValue(*result.Assistant.AssistantArn)

	// Store tags back in state
	tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tags)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		data.Tags = tagsMap
	}

	tflog.Trace(ctx, "Created Wisdom Assistant", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WisdomAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get assistant details
	result, err := r.client.GetAssistant(ctx, &wisdom.GetAssistantInput{
		AssistantId: aws.String(data.ID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read assistant, got error: %s", err),
		)
		return
	}

	// Update model
	data.Name = frameworktypes.StringValue(*result.Assistant.Name)
	data.Type = frameworktypes.StringValue(string(result.Assistant.Type))
	data.AssistantArn = frameworktypes.StringValue(*result.Assistant.AssistantArn)

	if result.Assistant.Description != nil {
		data.Description = frameworktypes.StringValue(*result.Assistant.Description)
	}

	if result.Assistant.Tags != nil {
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, result.Assistant.Tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WisdomAssistantResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure required tags
	tags, err := ensureRequiredTags(ctx, data.Tags)
	if err != nil {
		resp.Diagnostics.AddError("Tag Error", fmt.Sprintf("Unable to process tags: %s", err))
		return
	}

	// Note: AWS Wisdom doesn't support updating assistant properties other than tags
	// The name, type, and description cannot be changed after creation

	// Update tags
	_, err = r.client.TagResource(ctx, &wisdom.TagResourceInput{
		ResourceArn: aws.String(data.AssistantArn.ValueString()),
		Tags:        tags,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update assistant tags, got error: %s", err),
		)
		return
	}

	// Store tags back in state
	tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tags)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		data.Tags = tagsMap
	}

	tflog.Trace(ctx, "Updated Wisdom Assistant", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WisdomAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAssistant(ctx, &wisdom.DeleteAssistantInput{
		AssistantId: aws.String(data.ID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete assistant, got error: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Wisdom Assistant", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *WisdomAssistantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
