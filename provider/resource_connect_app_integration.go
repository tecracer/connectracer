// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appintegrations"
	"github.com/aws/aws-sdk-go-v2/service/appintegrations/types"
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
var _ resource.Resource = &ConnectAppIntegrationResource{}
var _ resource.ResourceWithImportState = &ConnectAppIntegrationResource{}

func NewConnectAppIntegrationResource() resource.Resource {
	return &ConnectAppIntegrationResource{}
}

// ConnectAppIntegrationResource defines the resource implementation.
type ConnectAppIntegrationResource struct {
	client *appintegrations.Client
}

// ConnectAppIntegrationResourceModel describes the resource data model.
type ConnectAppIntegrationResourceModel struct {
	ID              frameworktypes.String `tfsdk:"id"`
	Arn             frameworktypes.String `tfsdk:"arn"`
	Name            frameworktypes.String `tfsdk:"name"`
	Namespace       frameworktypes.String `tfsdk:"namespace"`
	Description     frameworktypes.String `tfsdk:"description"`
	AccessUrl       frameworktypes.String `tfsdk:"access_url"`
	ApplicationType frameworktypes.String `tfsdk:"application_type"`
	Permissions     frameworktypes.List   `tfsdk:"permissions"`
	Tags            frameworktypes.Map    `tfsdk:"tags"`
}

func (r *ConnectAppIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_app_integration"
}

func (r *ConnectAppIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS AppIntegrations Application resource.\n\n" +
			"This creates an Application (e.g., an MCP Server integration) that can then be associated " +
			"with an Amazon Connect instance using the `connectracer_connect_integration_association` resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the application.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the application.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the application.",
				Required:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "The namespace of the application.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the application.",
				Optional:            true,
			},
			"access_url": schema.StringAttribute{
				MarkdownDescription: "The URL to access the application (the external URL source).",
				Required:            true,
			},
			"application_type": schema.StringAttribute{
				MarkdownDescription: "The type of the application. Valid values: `STANDARD`, `SERVICE`, `MCP_SERVER`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: "The configuration of events or requests that the application has access to.",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the application.",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
		},
	}
}

func (r *ConnectAppIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectAppIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectAppIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &appintegrations.CreateApplicationInput{
		Name:      aws.String(data.Name.ValueString()),
		Namespace: aws.String(data.Namespace.ValueString()),
		ApplicationSourceConfig: &types.ApplicationSourceConfig{
			ExternalUrlConfig: &types.ExternalUrlConfig{
				AccessUrl: aws.String(data.AccessUrl.ValueString()),
			},
		},
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}

	if !data.ApplicationType.IsNull() && !data.ApplicationType.IsUnknown() {
		input.ApplicationType = types.ApplicationType(data.ApplicationType.ValueString())
	}

	// Permissions
	if !data.Permissions.IsNull() && !data.Permissions.IsUnknown() {
		var permissions []string
		diags := data.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Permissions = permissions
	}

	// Tags
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make(map[string]string)
		diags := data.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Tags = tags
	}

	tflog.Debug(ctx, "Creating AppIntegrations Application", map[string]interface{}{
		"name":      data.Name.ValueString(),
		"namespace": data.Namespace.ValueString(),
	})

	output, err := r.client.CreateApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AppIntegrations Application",
			fmt.Sprintf("Unable to create application %q, got error: %s", data.Name.ValueString(), err),
		)
		return
	}

	data.ID = frameworktypes.StringPointerValue(output.Id)
	data.Arn = frameworktypes.StringPointerValue(output.Arn)

	tflog.Trace(ctx, "Created AppIntegrations Application", map[string]interface{}{
		"id":  data.ID.ValueString(),
		"arn": data.Arn.ValueString(),
	})

	// Read back to populate all fields
	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAppIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectAppIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AppIntegrations Application", map[string]interface{}{
		"arn": data.Arn.ValueString(),
	})

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAppIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectAppIntegrationResourceModel
	var state ConnectAppIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AppIntegrations Application", map[string]interface{}{
		"arn": data.Arn.ValueString(),
	})

	updateInput := &appintegrations.UpdateApplicationInput{
		Arn: aws.String(data.Arn.ValueString()),
		ApplicationSourceConfig: &types.ApplicationSourceConfig{
			ExternalUrlConfig: &types.ExternalUrlConfig{
				AccessUrl: aws.String(data.AccessUrl.ValueString()),
			},
		},
	}

	// Always send name (it's required for update semantics)
	updateInput.Name = aws.String(data.Name.ValueString())

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		updateInput.Description = aws.String(data.Description.ValueString())
	} else {
		// Send empty string to clear description
		updateInput.Description = aws.String("")
	}

	// Permissions
	if !data.Permissions.IsNull() && !data.Permissions.IsUnknown() {
		var permissions []string
		diags := data.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateInput.Permissions = permissions
	}

	_, err := r.client.UpdateApplication(ctx, updateInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating AppIntegrations Application",
			fmt.Sprintf("Unable to update application %s, got error: %s", data.Arn.ValueString(), err),
		)
		return
	}

	// Handle tag changes
	if !data.Tags.Equal(state.Tags) {
		// Remove old tags
		if !state.Tags.IsNull() && !state.Tags.IsUnknown() {
			var oldTags map[string]string
			diags := state.Tags.ElementsAs(ctx, &oldTags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if len(oldTags) > 0 {
				tagKeys := make([]string, 0, len(oldTags))
				for k := range oldTags {
					tagKeys = append(tagKeys, k)
				}
				_, err := r.client.UntagResource(ctx, &appintegrations.UntagResourceInput{
					ResourceArn: aws.String(data.Arn.ValueString()),
					TagKeys:     tagKeys,
				})
				if err != nil {
					resp.Diagnostics.AddError(
						"Error removing tags from AppIntegrations Application",
						fmt.Sprintf("Unable to remove tags, got error: %s", err),
					)
					return
				}
			}
		}

		// Apply new tags
		if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
			var newTags map[string]string
			diags := data.Tags.ElementsAs(ctx, &newTags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if len(newTags) > 0 {
				_, err := r.client.TagResource(ctx, &appintegrations.TagResourceInput{
					ResourceArn: aws.String(data.Arn.ValueString()),
					Tags:        newTags,
				})
				if err != nil {
					resp.Diagnostics.AddError(
						"Error tagging AppIntegrations Application",
						fmt.Sprintf("Unable to add tags, got error: %s", err),
					)
					return
				}
			}
		}
	}

	// Read back to refresh state
	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated AppIntegrations Application", map[string]interface{}{
		"arn": data.Arn.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectAppIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectAppIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AppIntegrations Application", map[string]interface{}{
		"arn": data.Arn.ValueString(),
	})

	_, err := r.client.DeleteApplication(ctx, &appintegrations.DeleteApplicationInput{
		Arn: aws.String(data.Arn.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting AppIntegrations Application",
			fmt.Sprintf("Unable to delete application %s, got error: %s. "+
				"Note: Applications with existing IntegrationAssociations must have those removed first.", data.Arn.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted AppIntegrations Application", map[string]interface{}{
		"arn": data.Arn.ValueString(),
	})
}

func (r *ConnectAppIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ARN since that's what the API uses for Get/Update/Delete
	resource.ImportStatePassthroughID(ctx, path.Root("arn"), req, resp)
}

// readAndPopulateModel reads the application from the API and populates the model.
// It preserves the user's config values for tags and permissions when the API values match.
func (r *ConnectAppIntegrationResource) readAndPopulateModel(ctx context.Context, data *ConnectAppIntegrationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// GetApplication requires the ARN
	arn := data.Arn.ValueString()
	if arn == "" {
		diags.AddError(
			"Error reading AppIntegrations Application",
			"ARN is empty, cannot read application",
		)
		return diags
	}

	output, err := r.client.GetApplication(ctx, &appintegrations.GetApplicationInput{
		Arn: aws.String(arn),
	})
	if err != nil {
		diags.AddError(
			"Error reading AppIntegrations Application",
			fmt.Sprintf("Unable to read application %s, got error: %s", arn, err),
		)
		return diags
	}

	data.ID = frameworktypes.StringPointerValue(output.Id)
	data.Arn = frameworktypes.StringPointerValue(output.Arn)
	data.Name = frameworktypes.StringPointerValue(output.Name)
	data.Namespace = frameworktypes.StringPointerValue(output.Namespace)

	if output.Description != nil && *output.Description != "" {
		data.Description = frameworktypes.StringPointerValue(output.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	// Application type
	if output.ApplicationType != "" {
		data.ApplicationType = frameworktypes.StringValue(string(output.ApplicationType))
	} else {
		data.ApplicationType = frameworktypes.StringNull()
	}

	// Access URL from ApplicationSourceConfig
	if output.ApplicationSourceConfig != nil && output.ApplicationSourceConfig.ExternalUrlConfig != nil {
		data.AccessUrl = frameworktypes.StringPointerValue(output.ApplicationSourceConfig.ExternalUrlConfig.AccessUrl)
	}

	// Permissions
	if len(output.Permissions) > 0 {
		permsList, permDiags := frameworktypes.ListValueFrom(ctx, frameworktypes.StringType, output.Permissions)
		diags.Append(permDiags...)
		if diags.HasError() {
			return diags
		}
		data.Permissions = permsList
	} else if !data.Permissions.IsNull() {
		// Preserve null if user didn't specify permissions
		data.Permissions = frameworktypes.ListNull(frameworktypes.StringType)
	}

	// Tags - preserve user's config value, only update if API has different tags
	if len(output.Tags) > 0 {
		tagsMap, tagDiags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Tags)
		diags.Append(tagDiags...)
		if diags.HasError() {
			return diags
		}
		data.Tags = tagsMap
	} else if !data.Tags.IsNull() {
		data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
	}

	return diags
}
