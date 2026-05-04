// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &QConnectKnowledgeBaseResource{}
var _ resource.ResourceWithImportState = &QConnectKnowledgeBaseResource{}

func NewQConnectKnowledgeBaseResource() resource.Resource {
	return &QConnectKnowledgeBaseResource{}
}

// QConnectKnowledgeBaseResource defines the resource implementation.
type QConnectKnowledgeBaseResource struct {
	client *qconnect.Client
}

// QConnectKnowledgeBaseResourceModel describes the resource data model.
type QConnectKnowledgeBaseResourceModel struct {
	ID                                frameworktypes.String                            `tfsdk:"id"`
	KnowledgeBaseArn                  frameworktypes.String                            `tfsdk:"knowledge_base_arn"`
	Name                              frameworktypes.String                            `tfsdk:"name"`
	KnowledgeBaseType                 frameworktypes.String                            `tfsdk:"knowledge_base_type"`
	Description                       frameworktypes.String                            `tfsdk:"description"`
	Tags                              frameworktypes.Map                               `tfsdk:"tags"`
	RenderingConfiguration            *RenderingConfigurationModel                     `tfsdk:"rendering_configuration"`
	ServerSideEncryptionConfiguration *ServerSideEncryptionConfigurationModel          `tfsdk:"server_side_encryption_configuration"`
	SourceConfiguration               *SourceConfigurationModel                        `tfsdk:"source_configuration"`
	Status                            frameworktypes.String                            `tfsdk:"status"`
}

// RenderingConfigurationModel describes rendering configuration.
type RenderingConfigurationModel struct {
	TemplateUri frameworktypes.String `tfsdk:"template_uri"`
}

// SourceConfigurationModel describes source configuration.
type SourceConfigurationModel struct {
	AppIntegrationArn frameworktypes.String `tfsdk:"app_integration_arn"`
}

func (r *QConnectKnowledgeBaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_qconnect_knowledgebase"
}

func (r *QConnectKnowledgeBaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS Q Connect Knowledge Base resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the knowledge base",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"knowledge_base_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the knowledge base",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the knowledge base",
				Required:            true,
			},
			"knowledge_base_type": schema.StringAttribute{
				MarkdownDescription: "The type of knowledge base. Valid values: EXTERNAL, CUSTOM",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the knowledge base",
				Optional:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the knowledge base",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"rendering_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "Information about how to render the content",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"template_uri": schema.StringAttribute{
						MarkdownDescription: "A URI template containing exactly one variable in ${variableName} format",
						Optional:            true,
					},
				},
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
			"source_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The source of the knowledge base content. Only set for EXTERNAL knowledge bases",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"app_integration_arn": schema.StringAttribute{
						MarkdownDescription: "The Amazon Resource Name (ARN) of the AppIntegrations DataIntegration",
						Optional:            true,
					},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the knowledge base",
				Computed:            true,
			},
		},
	}
}

func (r *QConnectKnowledgeBaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.QConnect
}

func (r *QConnectKnowledgeBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data QConnectKnowledgeBaseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build CreateKnowledgeBase input
	input := &qconnect.CreateKnowledgeBaseInput{
		Name:              aws.String(data.Name.ValueString()),
		KnowledgeBaseType: types.KnowledgeBaseType(data.KnowledgeBaseType.ValueString()),
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

	// Add rendering configuration if provided
	if data.RenderingConfiguration != nil {
		input.RenderingConfiguration = &types.RenderingConfiguration{}
		if !data.RenderingConfiguration.TemplateUri.IsNull() {
			input.RenderingConfiguration.TemplateUri = aws.String(data.RenderingConfiguration.TemplateUri.ValueString())
		}
	}

	// Add server-side encryption configuration if provided
	if data.ServerSideEncryptionConfiguration != nil {
		input.ServerSideEncryptionConfiguration = &types.ServerSideEncryptionConfiguration{}
		if !data.ServerSideEncryptionConfiguration.KmsKeyId.IsNull() {
			input.ServerSideEncryptionConfiguration.KmsKeyId = aws.String(data.ServerSideEncryptionConfiguration.KmsKeyId.ValueString())
		}
	}

	// Add source configuration if provided
	if data.SourceConfiguration != nil {
		if !data.SourceConfiguration.AppIntegrationArn.IsNull() {
			input.SourceConfiguration = &types.SourceConfigurationMemberAppIntegrations{
				Value: types.AppIntegrationsConfiguration{
					AppIntegrationArn: aws.String(data.SourceConfiguration.AppIntegrationArn.ValueString()),
				},
			}
		}
	}

	tflog.Debug(ctx, "Creating AWS Q Connect Knowledge Base", map[string]interface{}{
		"name": data.Name.ValueString(),
		"type": data.KnowledgeBaseType.ValueString(),
	})

	// Create the knowledge base
	output, err := r.client.CreateKnowledgeBase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Q Connect Knowledge Base",
			fmt.Sprintf("Unable to create knowledge base, got error: %s", err),
		)
		return
	}

	// Map response to model
	if output.KnowledgeBase != nil {
		data.ID = frameworktypes.StringPointerValue(output.KnowledgeBase.KnowledgeBaseId)
		data.KnowledgeBaseArn = frameworktypes.StringPointerValue(output.KnowledgeBase.KnowledgeBaseArn)
		data.Name = frameworktypes.StringPointerValue(output.KnowledgeBase.Name)
		data.KnowledgeBaseType = frameworktypes.StringValue(string(output.KnowledgeBase.KnowledgeBaseType))
		data.Status = frameworktypes.StringValue(string(output.KnowledgeBase.Status))

		if output.KnowledgeBase.Description != nil {
			data.Description = frameworktypes.StringPointerValue(output.KnowledgeBase.Description)
		}

		// Map tags
		if len(output.KnowledgeBase.Tags) > 0 {
			tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.KnowledgeBase.Tags)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				data.Tags = tagsMap
			}
		}

		// Map rendering configuration
		if output.KnowledgeBase.RenderingConfiguration != nil && output.KnowledgeBase.RenderingConfiguration.TemplateUri != nil {
			data.RenderingConfiguration = &RenderingConfigurationModel{
				TemplateUri: frameworktypes.StringPointerValue(output.KnowledgeBase.RenderingConfiguration.TemplateUri),
			}
		}

		// Map server-side encryption configuration
		if output.KnowledgeBase.ServerSideEncryptionConfiguration != nil && output.KnowledgeBase.ServerSideEncryptionConfiguration.KmsKeyId != nil {
			data.ServerSideEncryptionConfiguration = &ServerSideEncryptionConfigurationModel{
				KmsKeyId: frameworktypes.StringPointerValue(output.KnowledgeBase.ServerSideEncryptionConfiguration.KmsKeyId),
			}
		}

		// Map source configuration
		if output.KnowledgeBase.SourceConfiguration != nil {
			switch v := output.KnowledgeBase.SourceConfiguration.(type) {
			case *types.SourceConfigurationMemberAppIntegrations:
				if v.Value.AppIntegrationArn != nil {
					data.SourceConfiguration = &SourceConfigurationModel{
						AppIntegrationArn: frameworktypes.StringPointerValue(v.Value.AppIntegrationArn),
					}
				}
			}
		}
	}

	tflog.Trace(ctx, "Created Q Connect Knowledge Base resource", map[string]interface{}{
		"knowledge_base_id": data.ID.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *QConnectKnowledgeBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data QConnectKnowledgeBaseResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS Q Connect Knowledge Base", map[string]interface{}{
		"knowledge_base_id": data.ID.ValueString(),
	})

	// Get the knowledge base
	input := &qconnect.GetKnowledgeBaseInput{
		KnowledgeBaseId: aws.String(data.ID.ValueString()),
	}

	output, err := r.client.GetKnowledgeBase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Q Connect Knowledge Base",
			fmt.Sprintf("Unable to read knowledge base %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Map response to model
	if output.KnowledgeBase != nil {
		data.KnowledgeBaseArn = frameworktypes.StringPointerValue(output.KnowledgeBase.KnowledgeBaseArn)
		data.Name = frameworktypes.StringPointerValue(output.KnowledgeBase.Name)
		data.KnowledgeBaseType = frameworktypes.StringValue(string(output.KnowledgeBase.KnowledgeBaseType))
		data.Status = frameworktypes.StringValue(string(output.KnowledgeBase.Status))

		if output.KnowledgeBase.Description != nil {
			data.Description = frameworktypes.StringPointerValue(output.KnowledgeBase.Description)
		} else {
			data.Description = frameworktypes.StringNull()
		}

		// Map tags
		if len(output.KnowledgeBase.Tags) > 0 {
			tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.KnowledgeBase.Tags)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				data.Tags = tagsMap
			}
		} else {
			data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
		}

		// Map rendering configuration
		if output.KnowledgeBase.RenderingConfiguration != nil && output.KnowledgeBase.RenderingConfiguration.TemplateUri != nil {
			data.RenderingConfiguration = &RenderingConfigurationModel{
				TemplateUri: frameworktypes.StringPointerValue(output.KnowledgeBase.RenderingConfiguration.TemplateUri),
			}
		} else {
			data.RenderingConfiguration = nil
		}

		// Map server-side encryption configuration
		if output.KnowledgeBase.ServerSideEncryptionConfiguration != nil && output.KnowledgeBase.ServerSideEncryptionConfiguration.KmsKeyId != nil {
			data.ServerSideEncryptionConfiguration = &ServerSideEncryptionConfigurationModel{
				KmsKeyId: frameworktypes.StringPointerValue(output.KnowledgeBase.ServerSideEncryptionConfiguration.KmsKeyId),
			}
		} else {
			data.ServerSideEncryptionConfiguration = nil
		}

		// Map source configuration
		if output.KnowledgeBase.SourceConfiguration != nil {
			switch v := output.KnowledgeBase.SourceConfiguration.(type) {
			case *types.SourceConfigurationMemberAppIntegrations:
				if v.Value.AppIntegrationArn != nil {
					data.SourceConfiguration = &SourceConfigurationModel{
						AppIntegrationArn: frameworktypes.StringPointerValue(v.Value.AppIntegrationArn),
					}
				}
			}
		} else {
			data.SourceConfiguration = nil
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *QConnectKnowledgeBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data QConnectKnowledgeBaseResourceModel
	var state QConnectKnowledgeBaseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AWS Q Connect Knowledge Base", map[string]interface{}{
		"knowledge_base_id": data.ID.ValueString(),
	})

	// Note: AWS Q Connect has limited update capabilities
	// We can only update tags and templateUri
	
	// Handle tag updates
	if !data.Tags.Equal(state.Tags) {
		knowledgeBaseArn := state.KnowledgeBaseArn.ValueString()

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
			untagInput := &qconnect.UntagResourceInput{
				ResourceArn: aws.String(knowledgeBaseArn),
				TagKeys:     tagsToRemove,
			}
			_, err := r.client.UntagResource(ctx, untagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing tags from Q Connect Knowledge Base",
					fmt.Sprintf("Unable to remove tags, got error: %s", err),
				)
				return
			}
		}

		// Add new/updated tags
		if len(newTags) > 0 {
			tagInput := &qconnect.TagResourceInput{
				ResourceArn: aws.String(knowledgeBaseArn),
				Tags:        newTags,
			}
			_, err := r.client.TagResource(ctx, tagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error tagging Q Connect Knowledge Base",
					fmt.Sprintf("Unable to add tags, got error: %s", err),
				)
				return
			}
		}
	}

	// Handle template URI updates using UpdateKnowledgeBaseTemplateUri
	var oldTemplateUri, newTemplateUri string
	if state.RenderingConfiguration != nil && !state.RenderingConfiguration.TemplateUri.IsNull() {
		oldTemplateUri = state.RenderingConfiguration.TemplateUri.ValueString()
	}
	if data.RenderingConfiguration != nil && !data.RenderingConfiguration.TemplateUri.IsNull() {
		newTemplateUri = data.RenderingConfiguration.TemplateUri.ValueString()
	}

	if oldTemplateUri != newTemplateUri {
		if newTemplateUri != "" {
			// Update template URI
			updateInput := &qconnect.UpdateKnowledgeBaseTemplateUriInput{
				KnowledgeBaseId: aws.String(data.ID.ValueString()),
				TemplateUri:     aws.String(newTemplateUri),
			}
			_, err := r.client.UpdateKnowledgeBaseTemplateUri(ctx, updateInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating Q Connect Knowledge Base template URI",
					fmt.Sprintf("Unable to update template URI, got error: %s", err),
				)
				return
			}
		} else if oldTemplateUri != "" {
			// Remove template URI
			removeInput := &qconnect.RemoveKnowledgeBaseTemplateUriInput{
				KnowledgeBaseId: aws.String(data.ID.ValueString()),
			}
			_, err := r.client.RemoveKnowledgeBaseTemplateUri(ctx, removeInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing Q Connect Knowledge Base template URI",
					fmt.Sprintf("Unable to remove template URI, got error: %s", err),
				)
				return
			}
		}
	}

	// Check for unsupported updates
	if !data.Name.Equal(state.Name) {
		resp.Diagnostics.AddWarning(
			"Unsupported Update",
			"AWS Q Connect does not support updating the knowledge base name. The knowledge base will need to be recreated to change the name.",
		)
	}

	if !data.Description.Equal(state.Description) {
		resp.Diagnostics.AddWarning(
			"Unsupported Update",
			"AWS Q Connect does not support updating the knowledge base description. The knowledge base will need to be recreated to change the description.",
		)
	}

	// Read the resource again to get the current state
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: req.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics

	tflog.Trace(ctx, "Updated Q Connect Knowledge Base resource")
}

func (r *QConnectKnowledgeBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data QConnectKnowledgeBaseResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AWS Q Connect Knowledge Base", map[string]interface{}{
		"knowledge_base_id": data.ID.ValueString(),
	})

	// Delete the knowledge base
	input := &qconnect.DeleteKnowledgeBaseInput{
		KnowledgeBaseId: aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteKnowledgeBase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Q Connect Knowledge Base",
			fmt.Sprintf("Unable to delete knowledge base %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Q Connect Knowledge Base resource")
}

func (r *QConnectKnowledgeBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
