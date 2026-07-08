// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectViewResource{}
var _ resource.ResourceWithImportState = &ConnectViewResource{}

func NewConnectViewResource() resource.Resource {
	return &ConnectViewResource{}
}

// ConnectViewResource defines the resource implementation.
type ConnectViewResource struct {
	client *connect.Client
}

// ConnectViewResourceModel describes the resource data model.
type ConnectViewResourceModel struct {
	ID                 frameworktypes.String `tfsdk:"id"`
	InstanceID         frameworktypes.String `tfsdk:"instance_id"`
	Name               frameworktypes.String `tfsdk:"name"`
	Description        frameworktypes.String `tfsdk:"description"`
	Status             frameworktypes.String `tfsdk:"status"`
	Template           frameworktypes.String `tfsdk:"template"`
	Actions            frameworktypes.List   `tfsdk:"actions"`
	ViewArn            frameworktypes.String `tfsdk:"view_arn"`
	ViewType           frameworktypes.String `tfsdk:"view_type"`
	Version            frameworktypes.Int64  `tfsdk:"version"`
	VersionDescription frameworktypes.String `tfsdk:"version_description"`
	ViewContentSha256  frameworktypes.String `tfsdk:"view_content_sha256"`
	InputSchema        frameworktypes.String `tfsdk:"input_schema"`
	QualifiedID        frameworktypes.String `tfsdk:"qualified_id"`
	CreatedTime        frameworktypes.String `tfsdk:"created_time"`
	LastModifiedTime   frameworktypes.String `tfsdk:"last_modified_time"`
	Tags               frameworktypes.Map    `tfsdk:"tags"`
	CreateVersion      frameworktypes.Bool   `tfsdk:"create_version"`
}

func (r *ConnectViewResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_view"
}

func (r *ConnectViewResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Connect View resource.\n\n" +
			"Connect Views allow you to define UI templates for the Connect agent workspace, " +
			"controlling what agents see and can act on during customer interactions.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the View",
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
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the View. Must be unique per Connect instance",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the View",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Indicates the view status as either `SAVED` or `PUBLISHED`. " +
					"The `PUBLISHED` status initiates full content validation",
				Required: true,
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "The view template JSON string representing the structure of the view",
				Required:            true,
			},
			"actions": schema.ListAttribute{
				MarkdownDescription: "A list of possible actions from the view",
				Required:            true,
				ElementType:         frameworktypes.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"view_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the View",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_type": schema.StringAttribute{
				MarkdownDescription: "The type of the view, e.g. `CUSTOMER_MANAGED`",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "The current version number of the View",
				Computed:            true,
				// No UseStateForUnknown: Update always calls createVersion, which
				// changes the version number. Keeping UseStateForUnknown here
				// causes "provider produced inconsistent result after apply" because
				// the plan would show the old version while the actual result differs.
				},
			"version_description": schema.StringAttribute{
				MarkdownDescription: "An optional description for the version being published via CreateViewVersion",
				Optional:            true,
			},
			"view_content_sha256": schema.StringAttribute{
				MarkdownDescription: "Checksum of the latest published view content",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"input_schema": schema.StringAttribute{
				MarkdownDescription: "The data schema matching data that the view template must be provided to render",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"qualified_id": schema.StringAttribute{
				MarkdownDescription: "The qualified ID in the format `<view_id>:<version>`, suitable for use in Contact Flows",
				Computed:            true,
			},
			"created_time": schema.StringAttribute{
				MarkdownDescription: "The timestamp of when the View was created (RFC3339)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_modified_time": schema.StringAttribute{
				MarkdownDescription: "The timestamp of the last UpdateViewContent or CreateViewVersion operation (RFC3339)",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "A map of tags to assign to the View resource",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"create_version": schema.BoolAttribute{
				MarkdownDescription: "Whether to publish a new immutable version after create. " +
					"On every update a new version is always created when content or metadata changes",
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ConnectViewResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectViewResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actions, diags := r.expandActions(ctx, data.Actions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &connect.CreateViewInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		Name:       aws.String(data.Name.ValueString()),
		Status:     types.ViewStatus(data.Status.ValueString()),
		Content: &types.ViewInputContent{
			Template: aws.String(data.Template.ValueString()),
			Actions:  actions,
		},
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make(map[string]string)
		diags := data.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Tags = tags
	}

	tflog.Debug(ctx, "Creating Connect View", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"name":        data.Name.ValueString(),
		"status":      data.Status.ValueString(),
	})

	output, err := r.client.CreateView(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Connect View",
			fmt.Sprintf("Unable to create Connect View %q, got error: %s", data.Name.ValueString(), err),
		)
		return
	}

	if output.View == nil {
		resp.Diagnostics.AddError(
			"Error creating Connect View",
			"CreateView response did not contain View data",
		)
		return
	}

	data.ID = frameworktypes.StringPointerValue(output.View.Id)
	data.ViewArn = frameworktypes.StringPointerValue(output.View.Arn)

	tflog.Trace(ctx, "Created Connect View", map[string]interface{}{
		"view_id": data.ID.ValueString(),
	})

	// Read back to populate all computed fields
	readDiags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Optionally create a version after creation
	if !data.CreateVersion.IsNull() && data.CreateVersion.ValueBool() {
		versionNumber, err := r.createVersion(ctx, data.InstanceID.ValueString(), data.ID.ValueString(), data)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error creating Connect View version",
				fmt.Sprintf("Connect View was created but version creation failed: %s", err),
			)
		} else {
			data.Version = frameworktypes.Int64Value(versionNumber)
		}
	}

	data.QualifiedID = r.computeQualifiedID(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectViewResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Connect View", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"view_id":     data.ID.ValueString(),
	})

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.QualifiedID = r.computeQualifiedID(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectViewResourceModel
	var state ConnectViewResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry over the view ID and ARN from state (they are computed on create and not re-computed on plan)
	data.ID = state.ID
	data.ViewArn = state.ViewArn

	tflog.Debug(ctx, "Updating Connect View", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"view_id":     data.ID.ValueString(),
	})

	// --- UpdateViewMetadata (name and/or description) ---
	if !data.Name.Equal(state.Name) || !data.Description.Equal(state.Description) {
		metaInput := &connect.UpdateViewMetadataInput{
			InstanceId: aws.String(data.InstanceID.ValueString()),
			ViewId:     aws.String(data.ID.ValueString()),
			Name:       aws.String(data.Name.ValueString()),
		}
		if !data.Description.IsNull() && !data.Description.IsUnknown() {
			metaInput.Description = aws.String(data.Description.ValueString())
		}

		_, err := r.client.UpdateViewMetadata(ctx, metaInput)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Connect View metadata",
				fmt.Sprintf("Unable to update metadata for Connect View %s, got error: %s", data.ID.ValueString(), err),
			)
			return
		}
	}

	// --- UpdateViewContent (template, actions, and/or status) ---
	if !data.Template.Equal(state.Template) || !data.Actions.Equal(state.Actions) || !data.Status.Equal(state.Status) {
		actions, diags := r.expandActions(ctx, data.Actions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		contentInput := &connect.UpdateViewContentInput{
			InstanceId: aws.String(data.InstanceID.ValueString()),
			ViewId:     aws.String(data.ID.ValueString()),
			Status:     types.ViewStatus(data.Status.ValueString()),
			Content: &types.ViewInputContent{
				Template: aws.String(data.Template.ValueString()),
				Actions:  actions,
			},
		}

		_, err := r.client.UpdateViewContent(ctx, contentInput)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Connect View content",
				fmt.Sprintf("Unable to update content for Connect View %s, got error: %s", data.ID.ValueString(), err),
			)
			return
		}
	}

	// Read back refreshed state before versioning so we have the latest SHA256
	readDiags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always create a new version after an update since something has changed
	versionNumber, err := r.createVersion(ctx, data.InstanceID.ValueString(), data.ID.ValueString(), data)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Error creating Connect View version",
			fmt.Sprintf("Connect View was updated but version creation failed: %s", err),
		)
	} else {
		data.Version = frameworktypes.Int64Value(versionNumber)
	}

	data.QualifiedID = r.computeQualifiedID(&data)

	tflog.Trace(ctx, "Updated Connect View resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectViewResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Connect View", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"view_id":     data.ID.ValueString(),
	})

	input := &connect.DeleteViewInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		ViewId:     aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteView(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			tflog.Debug(ctx, "Connect View already deleted", map[string]interface{}{
				"view_id": data.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Connect View",
			fmt.Sprintf("Unable to delete Connect View %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Connect View resource")
}

func (r *ConnectViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: instance_id/view_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: instance_id/view_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// readAndPopulateModel calls DescribeView and populates the model with the response.
func (r *ConnectViewResource) readAndPopulateModel(ctx context.Context, data *ConnectViewResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	output, err := r.client.DescribeView(ctx, &connect.DescribeViewInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		ViewId:     aws.String(data.ID.ValueString()),
	})
	if err != nil {
		diags.AddError(
			"Error reading Connect View",
			fmt.Sprintf("Unable to describe Connect View %s, got error: %s", data.ID.ValueString(), err),
		)
		return diags
	}

	if output.View == nil {
		diags.AddError(
			"Error reading Connect View",
			"DescribeView response did not contain View data",
		)
		return diags
	}

	v := output.View

	data.ID = frameworktypes.StringPointerValue(v.Id)
	data.ViewArn = frameworktypes.StringPointerValue(v.Arn)
	data.InstanceID = frameworktypes.StringValue(data.InstanceID.ValueString()) // preserved; API doesn't echo it
	data.Name = frameworktypes.StringPointerValue(v.Name)
	data.Status = frameworktypes.StringValue(string(v.Status))
	data.ViewType = frameworktypes.StringValue(string(v.Type))
	data.Version = frameworktypes.Int64Value(int64(v.Version))

	if v.Description != nil {
		data.Description = frameworktypes.StringPointerValue(v.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	if v.VersionDescription != nil {
		data.VersionDescription = frameworktypes.StringPointerValue(v.VersionDescription)
	} else {
		// keep user-provided value so plan doesn't show a diff on optional input
		if data.VersionDescription.IsUnknown() {
			data.VersionDescription = frameworktypes.StringNull()
		}
	}

	if v.ViewContentSha256 != nil {
		data.ViewContentSha256 = frameworktypes.StringPointerValue(v.ViewContentSha256)
	} else {
		data.ViewContentSha256 = frameworktypes.StringNull()
	}

	if v.CreatedTime != nil {
		data.CreatedTime = frameworktypes.StringValue(v.CreatedTime.Format(time.RFC3339))
	} else {
		data.CreatedTime = frameworktypes.StringNull()
	}

	if v.LastModifiedTime != nil {
		data.LastModifiedTime = frameworktypes.StringValue(v.LastModifiedTime.Format(time.RFC3339))
	} else {
		data.LastModifiedTime = frameworktypes.StringNull()
	}

	// Content fields
	if v.Content != nil {
		if v.Content.Template != nil {
			data.Template = frameworktypes.StringPointerValue(v.Content.Template)
		}

		if v.Content.InputSchema != nil {
			data.InputSchema = frameworktypes.StringPointerValue(v.Content.InputSchema)
		} else {
			data.InputSchema = frameworktypes.StringNull()
		}

		if len(v.Content.Actions) > 0 {
			actionsList, actionDiags := frameworktypes.ListValueFrom(ctx, frameworktypes.StringType, v.Content.Actions)
			diags.Append(actionDiags...)
			if diags.HasError() {
				return diags
			}
			data.Actions = actionsList
		} else {
			data.Actions = frameworktypes.ListValueMust(frameworktypes.StringType, nil)
		}
	} else {
		data.InputSchema = frameworktypes.StringNull()
		data.Actions = frameworktypes.ListValueMust(frameworktypes.StringType, nil)
	}

	// Tags — strip AWS-internal tags (e.g. "resourceArn") that are
	// automatically added by the Connect service and are not user-managed.
	// Leaving them in state causes a "provider produced inconsistent result
	// after apply" error when the user hasn't configured any tags, because the
	// plan shows tags = null but the post-apply read returns the injected keys.
	userTags := make(map[string]string)
	for k, v := range v.Tags {
		if k == "resourceArn" {
			continue
		}
		userTags[k] = v
	}
	if len(userTags) > 0 {
		tagsMap, tagDiags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, userTags)
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

// createVersion publishes a new immutable version of the Connect View.
func (r *ConnectViewResource) createVersion(ctx context.Context, instanceID, viewID string, data ConnectViewResourceModel) (int64, error) {
	input := &connect.CreateViewVersionInput{
		InstanceId: aws.String(instanceID),
		ViewId:     aws.String(viewID),
	}

	if !data.VersionDescription.IsNull() && !data.VersionDescription.IsUnknown() {
		input.VersionDescription = aws.String(data.VersionDescription.ValueString())
	}

	if !data.ViewContentSha256.IsNull() && !data.ViewContentSha256.IsUnknown() {
		input.ViewContentSha256 = aws.String(data.ViewContentSha256.ValueString())
	}

	tflog.Debug(ctx, "Creating Connect View version", map[string]interface{}{
		"instance_id": instanceID,
		"view_id":     viewID,
	})

	output, err := r.client.CreateViewVersion(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to create Connect View version: %w", err)
	}

	var versionNumber int64
	if output.View != nil {
		versionNumber = int64(output.View.Version)
	}

	tflog.Debug(ctx, "Created Connect View version", map[string]interface{}{
		"version_number": versionNumber,
	})

	return versionNumber, nil
}

// expandActions converts the framework List of strings to a plain []string for the AWS SDK.
func (r *ConnectViewResource) expandActions(ctx context.Context, actionsList frameworktypes.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if actionsList.IsNull() || actionsList.IsUnknown() {
		return nil, diags
	}

	var actions []string
	diags = actionsList.ElementsAs(ctx, &actions, false)
	return actions, diags
}

// computeQualifiedID returns "<view_id>:<version>" if a version exists, otherwise just the view ID.
func (r *ConnectViewResource) computeQualifiedID(data *ConnectViewResourceModel) frameworktypes.String {
	if data.ID.IsNull() || data.ID.IsUnknown() {
		return frameworktypes.StringNull()
	}
	if data.Version.IsNull() || data.Version.IsUnknown() || data.Version.ValueInt64() == 0 {
		return data.ID
	}
	qualified := fmt.Sprintf("%s:%d", data.ID.ValueString(), data.Version.ValueInt64())
	return frameworktypes.StringValue(qualified)
}
