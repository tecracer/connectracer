// Copyright IBM Corp. 2021, 2025
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
var _ resource.Resource = &ConnectIntegrationAssociationResource{}
var _ resource.ResourceWithImportState = &ConnectIntegrationAssociationResource{}

func NewConnectIntegrationAssociationResource() resource.Resource {
	return &ConnectIntegrationAssociationResource{}
}

// ConnectIntegrationAssociationResource defines the resource implementation.
type ConnectIntegrationAssociationResource struct {
	client *connect.Client
}

// ConnectIntegrationAssociationResourceModel describes the resource data model.
type ConnectIntegrationAssociationResourceModel struct {
	ID                         frameworktypes.String `tfsdk:"id"`
	IntegrationAssociationArn  frameworktypes.String `tfsdk:"integration_association_arn"`
	InstanceID                 frameworktypes.String `tfsdk:"instance_id"`
	IntegrationType            frameworktypes.String `tfsdk:"integration_type"`
	IntegrationArn             frameworktypes.String `tfsdk:"integration_arn"`
	SourceApplicationURL       frameworktypes.String `tfsdk:"source_application_url"`
	SourceApplicationName      frameworktypes.String `tfsdk:"source_application_name"`
	SourceType                 frameworktypes.String `tfsdk:"source_type"`
	Tags                       frameworktypes.Map    `tfsdk:"tags"`
}

func (r *ConnectIntegrationAssociationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_integration_association"
}

func (r *ConnectIntegrationAssociationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS Connect Integration Association resource\n\n" +
			"Creates an integration association between an Amazon Connect instance and external services such as " +
			"Wisdom knowledge bases, Q Connect, or other supported integrations.\n\n" +
			"Supported integration types include:\n" +
			"- `WISDOM_ASSISTANT`\n" +
			"- `WISDOM_KNOWLEDGE_BASE`\n" +
			"- `CASES_DOMAIN`\n" +
			"- `APPLICATION`\n" +
			"- `FILE_SCANNER`",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier for the integration association (IntegrationAssociationId)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"integration_association_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the integration association",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Amazon Connect instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_type": schema.StringAttribute{
				MarkdownDescription: "The type of integration. Valid values: `WISDOM_ASSISTANT`, `WISDOM_KNOWLEDGE_BASE`, `CASES_DOMAIN`, `APPLICATION`, `FILE_SCANNER`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the integration. " +
					"For WISDOM_ASSISTANT or WISDOM_KNOWLEDGE_BASE, this should be the ARN of the Wisdom assistant or knowledge base",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_application_url": schema.StringAttribute{
				MarkdownDescription: "The URL for the external application. Required for APPLICATION integration type",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_application_name": schema.StringAttribute{
				MarkdownDescription: "The name of the external application. Required for APPLICATION integration type",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_type": schema.StringAttribute{
				MarkdownDescription: "The type of the data source. Valid values: `SALESFORCE`, `ZENDESK`, `CASES`",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the integration association",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
		},
	}
}

func (r *ConnectIntegrationAssociationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.Connect
}

func (r *ConnectIntegrationAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectIntegrationAssociationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build CreateIntegrationAssociation input
	input := &connect.CreateIntegrationAssociationInput{
		InstanceId:      aws.String(data.InstanceID.ValueString()),
		IntegrationType: types.IntegrationType(data.IntegrationType.ValueString()),
		IntegrationArn:  aws.String(data.IntegrationArn.ValueString()),
	}

	// Add optional fields
	if !data.SourceApplicationURL.IsNull() {
		input.SourceApplicationUrl = aws.String(data.SourceApplicationURL.ValueString())
	}

	if !data.SourceApplicationName.IsNull() {
		input.SourceApplicationName = aws.String(data.SourceApplicationName.ValueString())
	}

	if !data.SourceType.IsNull() {
		input.SourceType = types.SourceType(data.SourceType.ValueString())
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

	tflog.Debug(ctx, "Creating AWS Connect Integration Association", map[string]interface{}{
		"instance_id":      data.InstanceID.ValueString(),
		"integration_type": data.IntegrationType.ValueString(),
		"integration_arn":  data.IntegrationArn.ValueString(),
	})

	// Create the integration association
	output, err := r.client.CreateIntegrationAssociation(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Connect Integration Association",
			fmt.Sprintf("Unable to create integration association, got error: %s", err),
		)
		return
	}

	// Map response to model
	if output.IntegrationAssociationId != nil {
		data.ID = frameworktypes.StringPointerValue(output.IntegrationAssociationId)
	}
	if output.IntegrationAssociationArn != nil {
		data.IntegrationAssociationArn = frameworktypes.StringPointerValue(output.IntegrationAssociationArn)
	}

	tflog.Trace(ctx, "Created Connect Integration Association resource", map[string]interface{}{
		"integration_association_id": data.ID.ValueString(),
	})

	// List integration associations to get complete state
	// Note: AWS Connect doesn't have a DescribeIntegrationAssociation API,
	// so we need to list and filter
	listInput := &connect.ListIntegrationAssociationsInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
	}

	var found bool
	paginator := connect.NewListIntegrationAssociationsPaginator(r.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading Connect Integration Association",
				fmt.Sprintf("Unable to list integration associations, got error: %s", err),
			)
			return
		}

		for _, association := range page.IntegrationAssociationSummaryList {
			if association.IntegrationAssociationId != nil &&
				*association.IntegrationAssociationId == data.ID.ValueString() {
				// Found our association - populate all fields
				found = true
				data.IntegrationAssociationArn = frameworktypes.StringPointerValue(association.IntegrationAssociationArn)
				data.IntegrationType = frameworktypes.StringValue(string(association.IntegrationType))
				data.IntegrationArn = frameworktypes.StringPointerValue(association.IntegrationArn)

				// Handle optional fields - use null if not present
				if association.SourceApplicationUrl != nil && *association.SourceApplicationUrl != "" {
					data.SourceApplicationURL = frameworktypes.StringPointerValue(association.SourceApplicationUrl)
				} else {
					data.SourceApplicationURL = frameworktypes.StringNull()
				}

				if association.SourceApplicationName != nil && *association.SourceApplicationName != "" {
					data.SourceApplicationName = frameworktypes.StringPointerValue(association.SourceApplicationName)
				} else {
					data.SourceApplicationName = frameworktypes.StringNull()
				}

				// SourceType might be empty for WISDOM integrations
				if association.SourceType != "" {
					data.SourceType = frameworktypes.StringValue(string(association.SourceType))
				} else {
					data.SourceType = frameworktypes.StringNull()
				}
				break
			}
		}

		if found {
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Connect Integration Association Not Found",
			fmt.Sprintf("Integration association %s not found after creation in instance %s", data.ID.ValueString(), data.InstanceID.ValueString()),
		)
		return
	}

	// Get tags using ListTagsForResource
	tagsInput := &connect.ListTagsForResourceInput{
		ResourceArn: aws.String(data.IntegrationAssociationArn.ValueString()),
	}

	tagsOutput, err := r.client.ListTagsForResource(ctx, tagsInput)
	if err != nil {
		// Tags might not be supported for this resource type, just log and continue
		tflog.Warn(ctx, "Unable to read tags for integration association", map[string]interface{}{
			"error": err.Error(),
		})
	} else if tagsOutput != nil && len(tagsOutput.Tags) > 0 {
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tagsOutput.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Tags = tagsMap
	} else {
		data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
	}

	// Save complete data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectIntegrationAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectIntegrationAssociationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS Connect Integration Association", map[string]interface{}{
		"instance_id":                data.InstanceID.ValueString(),
		"integration_association_id": data.ID.ValueString(),
	})

	// List integration associations and find the one we're looking for
	// Note: AWS Connect doesn't have a DescribeIntegrationAssociation API,
	// so we need to list and filter
	listInput := &connect.ListIntegrationAssociationsInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
	}

	var found bool
	paginator := connect.NewListIntegrationAssociationsPaginator(r.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading Connect Integration Association",
				fmt.Sprintf("Unable to list integration associations, got error: %s", err),
			)
			return
		}

		for _, association := range page.IntegrationAssociationSummaryList {
			if association.IntegrationAssociationId != nil &&
				*association.IntegrationAssociationId == data.ID.ValueString() {
				// Found our association
				found = true
				data.IntegrationAssociationArn = frameworktypes.StringPointerValue(association.IntegrationAssociationArn)
				data.IntegrationType = frameworktypes.StringValue(string(association.IntegrationType))
				data.IntegrationArn = frameworktypes.StringPointerValue(association.IntegrationArn)
				
				// Handle optional fields - use null if not present
				if association.SourceApplicationUrl != nil && *association.SourceApplicationUrl != "" {
					data.SourceApplicationURL = frameworktypes.StringPointerValue(association.SourceApplicationUrl)
				} else {
					data.SourceApplicationURL = frameworktypes.StringNull()
				}
				
				if association.SourceApplicationName != nil && *association.SourceApplicationName != "" {
					data.SourceApplicationName = frameworktypes.StringPointerValue(association.SourceApplicationName)
				} else {
					data.SourceApplicationName = frameworktypes.StringNull()
				}
				
				// SourceType might be empty for WISDOM integrations
				if association.SourceType != "" {
					data.SourceType = frameworktypes.StringValue(string(association.SourceType))
				} else {
					data.SourceType = frameworktypes.StringNull()
				}
				break
			}
		}

		if found {
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Connect Integration Association Not Found",
			fmt.Sprintf("Integration association %s not found in instance %s", data.ID.ValueString(), data.InstanceID.ValueString()),
		)
		return
	}

	// Get tags using ListTagsForResource
	tagsInput := &connect.ListTagsForResourceInput{
		ResourceArn: aws.String(data.IntegrationAssociationArn.ValueString()),
	}

	tagsOutput, err := r.client.ListTagsForResource(ctx, tagsInput)
	if err != nil {
		// Tags might not be supported for this resource type, just log and continue
		tflog.Warn(ctx, "Unable to read tags for integration association", map[string]interface{}{
			"error": err.Error(),
		})
	} else if tagsOutput != nil && len(tagsOutput.Tags) > 0 {
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tagsOutput.Tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	} else {
		data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectIntegrationAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectIntegrationAssociationResourceModel
	var state ConnectIntegrationAssociationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AWS Connect Integration Association", map[string]interface{}{
		"integration_association_id": data.ID.ValueString(),
	})

	// Note: AWS Connect doesn't have an UpdateIntegrationAssociation API
	// The only field that can be updated is tags using TagResource/UntagResource

	// Handle tag updates
	if !data.Tags.Equal(state.Tags) {
		resourceArn := state.IntegrationAssociationArn.ValueString()

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
			untagInput := &connect.UntagResourceInput{
				ResourceArn: aws.String(resourceArn),
				TagKeys:     tagsToRemove,
			}
			_, err := r.client.UntagResource(ctx, untagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing tags from Connect Integration Association",
					fmt.Sprintf("Unable to remove tags, got error: %s", err),
				)
				return
			}
		}

		// Add new/updated tags
		if len(newTags) > 0 {
			tagInput := &connect.TagResourceInput{
				ResourceArn: aws.String(resourceArn),
				Tags:        newTags,
			}
			_, err := r.client.TagResource(ctx, tagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error tagging Connect Integration Association",
					fmt.Sprintf("Unable to add tags, got error: %s", err),
				)
				return
			}
		}
	}

	// Read the resource again to get the current state
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: req.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics

	tflog.Trace(ctx, "Updated Connect Integration Association resource")
}

func (r *ConnectIntegrationAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectIntegrationAssociationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AWS Connect Integration Association", map[string]interface{}{
		"instance_id":                data.InstanceID.ValueString(),
		"integration_association_id": data.ID.ValueString(),
	})

	// Delete the integration association
	input := &connect.DeleteIntegrationAssociationInput{
		InstanceId:                aws.String(data.InstanceID.ValueString()),
		IntegrationAssociationId:  aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteIntegrationAssociation(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Connect Integration Association",
			fmt.Sprintf("Unable to delete integration association %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Connect Integration Association resource")
}

func (r *ConnectIntegrationAssociationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by composite ID: instance_id/integration_association_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: instance_id/integration_association_id, got: %s", req.ID),
		)
		return
	}

	instanceID := parts[0]
	integrationAssociationID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), integrationAssociationID)...)
}
