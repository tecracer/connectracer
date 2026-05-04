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
	ID                                frameworktypes.String                            `tfsdk:"id"`
	AssistantArn                      frameworktypes.String                            `tfsdk:"assistant_arn"`
	Name                              frameworktypes.String                            `tfsdk:"name"`
	Type                              frameworktypes.String                            `tfsdk:"type"`
	Description                       frameworktypes.String                            `tfsdk:"description"`
	Tags                              frameworktypes.Map                               `tfsdk:"tags"`
	ServerSideEncryptionConfiguration *ServerSideEncryptionConfigurationModel          `tfsdk:"server_side_encryption_configuration"`
	Status                            frameworktypes.String                            `tfsdk:"status"`
}

// ServerSideEncryptionConfigurationModel describes server-side encryption configuration.
type ServerSideEncryptionConfigurationModel struct {
	KmsKeyId frameworktypes.String `tfsdk:"kms_key_id"`
}

func (r *WisdomAssistantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wisdom_assistant"
}

func (r *WisdomAssistantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS Wisdom Assistant resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the assistant",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"assistant_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the assistant",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the assistant",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of assistant. Valid values: AGENT",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the assistant",
				Optional:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the assistant",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"server_side_encryption_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The KMS key used for encryption",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"kms_key_id": schema.StringAttribute{
						MarkdownDescription: "The KMS key ID or ARN",
						Optional:            true,
					},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the assistant",
				Computed:            true,
			},
		},
	}
}

func (r *WisdomAssistantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build CreateAssistant input
	input := &wisdom.CreateAssistantInput{
		Name: aws.String(data.Name.ValueString()),
		Type: types.AssistantType(data.Type.ValueString()),
	}

	// Add optional fields
	if !data.Description.IsNull() {
		input.Description = aws.String(data.Description.ValueString())
	}

	// Add tags if provided
	if !data.Tags.IsNull() {
		tags := make(map[string]string)
		diags := data.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Tags = tags
	}

	// Add server-side encryption configuration if provided
	if data.ServerSideEncryptionConfiguration != nil {
		input.ServerSideEncryptionConfiguration = &types.ServerSideEncryptionConfiguration{}
		if !data.ServerSideEncryptionConfiguration.KmsKeyId.IsNull() {
			input.ServerSideEncryptionConfiguration.KmsKeyId = aws.String(data.ServerSideEncryptionConfiguration.KmsKeyId.ValueString())
		}
	}

	tflog.Debug(ctx, "Creating AWS Wisdom Assistant", map[string]interface{}{
		"name": data.Name.ValueString(),
		"type": data.Type.ValueString(),
	})

	// Create the assistant
	output, err := r.client.CreateAssistant(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Wisdom Assistant",
			fmt.Sprintf("Unable to create assistant, got error: %s", err),
		)
		return
	}

	// Map response to model
	if output.Assistant != nil {
		data.ID = frameworktypes.StringPointerValue(output.Assistant.AssistantId)
		data.AssistantArn = frameworktypes.StringPointerValue(output.Assistant.AssistantArn)
		data.Name = frameworktypes.StringPointerValue(output.Assistant.Name)
		data.Type = frameworktypes.StringValue(string(output.Assistant.Type))
		data.Status = frameworktypes.StringValue(string(output.Assistant.Status))

		if output.Assistant.Description != nil {
			data.Description = frameworktypes.StringPointerValue(output.Assistant.Description)
		}

		// Map tags
		if len(output.Assistant.Tags) > 0 {
			tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Assistant.Tags)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				data.Tags = tagsMap
			}
		}

		// Map server-side encryption configuration
		if output.Assistant.ServerSideEncryptionConfiguration != nil && output.Assistant.ServerSideEncryptionConfiguration.KmsKeyId != nil {
			data.ServerSideEncryptionConfiguration = &ServerSideEncryptionConfigurationModel{
				KmsKeyId: frameworktypes.StringPointerValue(output.Assistant.ServerSideEncryptionConfiguration.KmsKeyId),
			}
		}
	}

	tflog.Trace(ctx, "Created Wisdom Assistant resource", map[string]interface{}{
		"assistant_id": data.ID.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WisdomAssistantResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS Wisdom Assistant", map[string]interface{}{
		"assistant_id": data.ID.ValueString(),
	})

	// Get the assistant
	input := &wisdom.GetAssistantInput{
		AssistantId: aws.String(data.ID.ValueString()),
	}

	output, err := r.client.GetAssistant(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Wisdom Assistant",
			fmt.Sprintf("Unable to read assistant %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Map response to model
	if output.Assistant != nil {
		data.AssistantArn = frameworktypes.StringPointerValue(output.Assistant.AssistantArn)
		data.Name = frameworktypes.StringPointerValue(output.Assistant.Name)
		data.Type = frameworktypes.StringValue(string(output.Assistant.Type))
		data.Status = frameworktypes.StringValue(string(output.Assistant.Status))

		if output.Assistant.Description != nil {
			data.Description = frameworktypes.StringPointerValue(output.Assistant.Description)
		} else {
			data.Description = frameworktypes.StringNull()
		}

		// Map tags
		if len(output.Assistant.Tags) > 0 {
			tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Assistant.Tags)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				data.Tags = tagsMap
			}
		} else {
			data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
		}

		// Map server-side encryption configuration
		if output.Assistant.ServerSideEncryptionConfiguration != nil && output.Assistant.ServerSideEncryptionConfiguration.KmsKeyId != nil {
			data.ServerSideEncryptionConfiguration = &ServerSideEncryptionConfigurationModel{
				KmsKeyId: frameworktypes.StringPointerValue(output.Assistant.ServerSideEncryptionConfiguration.KmsKeyId),
			}
		} else {
			data.ServerSideEncryptionConfiguration = nil
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WisdomAssistantResourceModel
	var state WisdomAssistantResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AWS Wisdom Assistant", map[string]interface{}{
		"assistant_id": data.ID.ValueString(),
	})

	// Note: AWS Wisdom doesn't have an UpdateAssistant API
	// The only fields that can be updated are tags using TagResource/UntagResource
	
	// Handle tag updates
	if !data.Tags.Equal(state.Tags) {
		assistantArn := state.AssistantArn.ValueString()

		// Get new tags
		var newTags map[string]string
		if !data.Tags.IsNull() {
			diags := data.Tags.ElementsAs(ctx, &newTags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		// Get old tags
		var oldTags map[string]string
		if !state.Tags.IsNull() {
			diags := state.Tags.ElementsAs(ctx, &oldTags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		// Determine tags to remove
		var tagsToRemove []string
		for key := range oldTags {
			if _, exists := newTags[key]; !exists {
				tagsToRemove = append(tagsToRemove, key)
			}
		}

		// Remove old tags
		if len(tagsToRemove) > 0 {
			untagInput := &wisdom.UntagResourceInput{
				ResourceArn: aws.String(assistantArn),
				TagKeys:     tagsToRemove,
			}
			_, err := r.client.UntagResource(ctx, untagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing tags from Wisdom Assistant",
					fmt.Sprintf("Unable to remove tags, got error: %s", err),
				)
				return
			}
		}

		// Add new/updated tags
		if len(newTags) > 0 {
			tagInput := &wisdom.TagResourceInput{
				ResourceArn: aws.String(assistantArn),
				Tags:        newTags,
			}
			_, err := r.client.TagResource(ctx, tagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error tagging Wisdom Assistant",
					fmt.Sprintf("Unable to add tags, got error: %s", err),
				)
				return
			}
		}
	}

	// Check for unsupported updates
	if !data.Name.Equal(state.Name) {
		resp.Diagnostics.AddWarning(
			"Unsupported Update",
			"AWS Wisdom does not support updating the assistant name. The assistant will need to be recreated to change the name.",
		)
	}

	if !data.Description.Equal(state.Description) {
		resp.Diagnostics.AddWarning(
			"Unsupported Update",
			"AWS Wisdom does not support updating the assistant description. The assistant will need to be recreated to change the description.",
		)
	}

	// Read the resource again to get the current state
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: req.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics

	tflog.Trace(ctx, "Updated Wisdom Assistant resource")
}

func (r *WisdomAssistantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WisdomAssistantResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AWS Wisdom Assistant", map[string]interface{}{
		"assistant_id": data.ID.ValueString(),
	})

	// Delete the assistant
	input := &wisdom.DeleteAssistantInput{
		AssistantId: aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteAssistant(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Wisdom Assistant",
			fmt.Sprintf("Unable to delete assistant %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Wisdom Assistant resource")
}

func (r *WisdomAssistantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
