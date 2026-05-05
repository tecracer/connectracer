// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appintegrations"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AppIntegrationsDataIntegrationResource{}
var _ resource.ResourceWithImportState = &AppIntegrationsDataIntegrationResource{}

func NewAppIntegrationsDataIntegrationResource() resource.Resource {
	return &AppIntegrationsDataIntegrationResource{}
}

// AppIntegrationsDataIntegrationResource defines the resource implementation.
type AppIntegrationsDataIntegrationResource struct {
	client *appintegrations.Client
}

// AppIntegrationsDataIntegrationResourceModel describes the resource data model.
type AppIntegrationsDataIntegrationResourceModel struct {
	ID          frameworktypes.String `tfsdk:"id"`
	Arn         frameworktypes.String `tfsdk:"arn"`
	Name        frameworktypes.String `tfsdk:"name"`
	Description frameworktypes.String `tfsdk:"description"`
	SourceURI   frameworktypes.String `tfsdk:"source_uri"`
	KmsKey      frameworktypes.String `tfsdk:"kms_key"`
	Tags        frameworktypes.Map    `tfsdk:"tags"`
}

func (r *AppIntegrationsDataIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_appintegrations_data_integration"
}

func (r *AppIntegrationsDataIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS AppIntegrations DataIntegration resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the data integration",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the data integration",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the data integration",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the data integration",
				Optional:            true,
			},
			"source_uri": schema.StringAttribute{
				MarkdownDescription: "The URI of the data source (e.g., s3://bucket-name)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kms_key": schema.StringAttribute{
				MarkdownDescription: "The KMS key for encryption",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the data integration. The `AmazonConnectEnabled = \"True\"` tag is automatically added if not present.",
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

func (r *AppIntegrationsDataIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.AppIntegrations
}

func (r *AppIntegrationsDataIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppIntegrationsDataIntegrationResourceModel

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

	input := &appintegrations.CreateDataIntegrationInput{
		Name:      aws.String(data.Name.ValueString()),
		SourceURI: aws.String(data.SourceURI.ValueString()),
		Tags:      tags,
	}

	if !data.Description.IsNull() {
		input.Description = aws.String(data.Description.ValueString())
	}

	if !data.KmsKey.IsNull() {
		input.KmsKey = aws.String(data.KmsKey.ValueString())
	}

	tflog.Debug(ctx, "Creating AWS AppIntegrations DataIntegration", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	output, err := r.client.CreateDataIntegration(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AppIntegrations DataIntegration",
			fmt.Sprintf("Unable to create data integration, got error: %s", err),
		)
		return
	}

	data.ID = frameworktypes.StringPointerValue(output.Id)
	data.Arn = frameworktypes.StringPointerValue(output.Arn)
	data.Name = frameworktypes.StringPointerValue(output.Name)
	data.SourceURI = frameworktypes.StringPointerValue(output.SourceURI)

	if output.Description != nil {
		data.Description = frameworktypes.StringPointerValue(output.Description)
	}

	if output.KmsKey != nil {
		data.KmsKey = frameworktypes.StringPointerValue(output.KmsKey)
	}

	// Store tags back in state
	if len(output.Tags) > 0 {
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	} else {
		// If API doesn't return tags, use what we sent
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	}

	tflog.Trace(ctx, "Created AppIntegrations DataIntegration resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppIntegrationsDataIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppIntegrationsDataIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS AppIntegrations DataIntegration", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	input := &appintegrations.GetDataIntegrationInput{
		Identifier: aws.String(data.ID.ValueString()),
	}

	output, err := r.client.GetDataIntegration(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading AppIntegrations DataIntegration",
			fmt.Sprintf("Unable to read data integration %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	data.Arn = frameworktypes.StringPointerValue(output.Arn)
	data.Name = frameworktypes.StringPointerValue(output.Name)
	data.SourceURI = frameworktypes.StringPointerValue(output.SourceURI)

	if output.Description != nil {
		data.Description = frameworktypes.StringPointerValue(output.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	if output.KmsKey != nil {
		data.KmsKey = frameworktypes.StringPointerValue(output.KmsKey)
	} else {
		data.KmsKey = frameworktypes.StringNull()
	}

	if len(output.Tags) > 0 {
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	} else {
		data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppIntegrationsDataIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppIntegrationsDataIntegrationResourceModel
	var state AppIntegrationsDataIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AWS AppIntegrations DataIntegration", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// Handle description update
	if !data.Description.Equal(state.Description) {
		input := &appintegrations.UpdateDataIntegrationInput{
			Identifier:  aws.String(data.ID.ValueString()),
			Description: aws.String(data.Description.ValueString()),
		}

		_, err := r.client.UpdateDataIntegration(ctx, input)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating AppIntegrations DataIntegration",
				fmt.Sprintf("Unable to update data integration, got error: %s", err),
			)
			return
		}
	}

	// Handle tag updates
	if !data.Tags.Equal(state.Tags) {
		dataIntegrationArn := state.Arn.ValueString()

		// Ensure required tags in new tags
		newTags, err := ensureRequiredTags(ctx, data.Tags)
		if err != nil {
			resp.Diagnostics.AddError("Tag Error", fmt.Sprintf("Unable to process tags: %s", err))
			return
		}

		var oldTags map[string]string
		if !state.Tags.IsNull() {
			diags := state.Tags.ElementsAs(ctx, &oldTags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		var tagsToRemove []string
		for key := range oldTags {
			if _, exists := newTags[key]; !exists {
				tagsToRemove = append(tagsToRemove, key)
			}
		}

		if len(tagsToRemove) > 0 {
			untagInput := &appintegrations.UntagResourceInput{
				ResourceArn: aws.String(dataIntegrationArn),
				TagKeys:     tagsToRemove,
			}
			_, err := r.client.UntagResource(ctx, untagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing tags from AppIntegrations DataIntegration",
					fmt.Sprintf("Unable to remove tags, got error: %s", err),
				)
				return
			}
		}

		if len(newTags) > 0 {
			tagInput := &appintegrations.TagResourceInput{
				ResourceArn: aws.String(dataIntegrationArn),
				Tags:        newTags,
			}
			_, err := r.client.TagResource(ctx, tagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error tagging AppIntegrations DataIntegration",
					fmt.Sprintf("Unable to add tags, got error: %s", err),
				)
				return
			}
		}

		// Update state with new tags
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, newTags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	}

	// Read the resource again to get the current state
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: req.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics

	tflog.Trace(ctx, "Updated AppIntegrations DataIntegration resource")
}

func (r *AppIntegrationsDataIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppIntegrationsDataIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AWS AppIntegrations DataIntegration", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	input := &appintegrations.DeleteDataIntegrationInput{
		DataIntegrationIdentifier: aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteDataIntegration(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting AppIntegrations DataIntegration",
			fmt.Sprintf("Unable to delete data integration %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted AppIntegrations DataIntegration resource")
}

func (r *AppIntegrationsDataIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
