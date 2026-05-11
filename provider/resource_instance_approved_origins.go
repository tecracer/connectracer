// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &InstanceApprovedOriginsResource{}
var _ resource.ResourceWithImportState = &InstanceApprovedOriginsResource{}

func NewInstanceApprovedOriginsResource() resource.Resource {
	return &InstanceApprovedOriginsResource{}
}

// InstanceApprovedOriginsResource defines the resource implementation.
type InstanceApprovedOriginsResource struct {
	client *connect.Client
}

// InstanceApprovedOriginsResourceModel describes the resource data model.
type InstanceApprovedOriginsResourceModel struct {
	ID         frameworktypes.String `tfsdk:"id"`
	InstanceID frameworktypes.String `tfsdk:"instance_id"`
	Origin     frameworktypes.String `tfsdk:"origin"`
}

func (r *InstanceApprovedOriginsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_approved_origins"
}

func (r *InstanceApprovedOriginsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an approved origin (domain) for an Amazon Connect instance.\n\n" +
			"This resource associates a domain URL to the instance's allow list, enabling cross-origin " +
			"access for integrated applications.\n\n" +
			"**Note:** This API is in preview release for Amazon Connect and is subject to change.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The resource identifier (instance_id/origin)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Amazon Connect instance. You can find the instance ID in the Amazon Resource Name (ARN) of the instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"origin": schema.StringAttribute{
				MarkdownDescription: "The domain URL to add to the allow list (e.g. `https://example.com`). Maximum length of 267 characters",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *InstanceApprovedOriginsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InstanceApprovedOriginsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstanceApprovedOriginsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueString()
	origin := data.Origin.ValueString()

	input := &connect.AssociateApprovedOriginInput{
		InstanceId: aws.String(instanceID),
		Origin:     aws.String(origin),
	}

	tflog.Debug(ctx, "Associating approved origin with Connect instance", map[string]interface{}{
		"instance_id": instanceID,
		"origin":      origin,
	})

	_, err := r.client.AssociateApprovedOrigin(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error associating approved origin",
			fmt.Sprintf("Unable to associate origin %q with instance %s, got error: %s", origin, instanceID, err),
		)
		return
	}

	// Compose the ID from instance_id and origin
	data.ID = frameworktypes.StringValue(instanceID + "/" + origin)

	tflog.Trace(ctx, "Associated approved origin with Connect instance", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceApprovedOriginsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstanceApprovedOriginsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueString()
	origin := data.Origin.ValueString()

	tflog.Debug(ctx, "Reading approved origin for Connect instance", map[string]interface{}{
		"instance_id": instanceID,
		"origin":      origin,
	})

	// List approved origins and check if our origin exists
	found, err := r.findApprovedOrigin(ctx, instanceID, origin)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading approved origins",
			fmt.Sprintf("Unable to list approved origins for instance %s, got error: %s", instanceID, err),
		)
		return
	}

	if !found {
		// Origin no longer exists, remove from state
		tflog.Warn(ctx, "Approved origin not found, removing from state", map[string]interface{}{
			"instance_id": instanceID,
			"origin":      origin,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	// Data unchanged, save back
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceApprovedOriginsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both instance_id and origin have RequiresReplace, so Update should never be called.
	// If it is called unexpectedly, just read the plan into state.
	var data InstanceApprovedOriginsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceApprovedOriginsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data InstanceApprovedOriginsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueString()
	origin := data.Origin.ValueString()

	tflog.Debug(ctx, "Disassociating approved origin from Connect instance", map[string]interface{}{
		"instance_id": instanceID,
		"origin":      origin,
	})

	input := &connect.DisassociateApprovedOriginInput{
		InstanceId: aws.String(instanceID),
		Origin:     aws.String(origin),
	}

	_, err := r.client.DisassociateApprovedOrigin(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error disassociating approved origin",
			fmt.Sprintf("Unable to disassociate origin %q from instance %s, got error: %s", origin, instanceID, err),
		)
		return
	}

	tflog.Trace(ctx, "Disassociated approved origin from Connect instance")
}

func (r *InstanceApprovedOriginsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: instance_id/origin
	// The origin is a URL (e.g. https://example.com) so we split on the first "/" only
	// Actually the format is: instance_id/https://example.com
	// We need to split carefully: first segment is the instance_id, the rest is the origin
	idx := strings.Index(req.ID, "/")
	if idx <= 0 || idx == len(req.ID)-1 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: instance_id/origin (e.g. abc123/https://example.com), got: %s", req.ID),
		)
		return
	}

	instanceID := req.ID[:idx]
	origin := req.ID[idx+1:]

	if instanceID == "" || origin == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: instance_id/origin, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("origin"), origin)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// findApprovedOrigin checks if a given origin is in the approved list for the instance.
func (r *InstanceApprovedOriginsResource) findApprovedOrigin(ctx context.Context, instanceID, origin string) (bool, error) {
	paginator := connect.NewListApprovedOriginsPaginator(r.client, &connect.ListApprovedOriginsInput{
		InstanceId: aws.String(instanceID),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, o := range page.Origins {
			if o == origin {
				return true, nil
			}
		}
	}

	return false, nil
}
