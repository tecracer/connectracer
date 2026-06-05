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
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
var _ resource.Resource = &ConnectContactFlowResource{}
var _ resource.ResourceWithImportState = &ConnectContactFlowResource{}

func NewConnectContactFlowResource() resource.Resource {
	return &ConnectContactFlowResource{}
}

// ConnectContactFlowResource defines the resource implementation.
type ConnectContactFlowResource struct {
	client *connect.Client
}

// ConnectContactFlowResourceModel describes the resource data model.
type ConnectContactFlowResourceModel struct {
	// ID is "instance_id:contact_flow_id" — the format used by the AWS provider,
	// which makes it easy to import or reference contact flows managed by either
	// provider.
	ID            frameworktypes.String `tfsdk:"id"`
	InstanceID    frameworktypes.String `tfsdk:"instance_id"`
	ContactFlowID frameworktypes.String `tfsdk:"contact_flow_id"`
	ARN           frameworktypes.String `tfsdk:"arn"`
	Name          frameworktypes.String `tfsdk:"name"`
	Description   frameworktypes.String `tfsdk:"description"`
	Type          frameworktypes.String `tfsdk:"type"`
	Content       frameworktypes.String `tfsdk:"content"`
	State         frameworktypes.String `tfsdk:"state"`
	Tags          frameworktypes.Map    `tfsdk:"tags"`
}

func (r *ConnectContactFlowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_contact_flow"
}

func (r *ConnectContactFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Connect Contact Flow.\n\n" +
			"Unlike the standard `aws_connect_contact_flow`, this resource **always performs in-place " +
			"updates** for content and metadata changes, and never triggers resource replacement. This " +
			"avoids the `DuplicateResourceException` that occurs when a replace-triggered flow tries to " +
			"create a new flow before destroying the old one (AWS enforces unique names per instance).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Terraform resource ID in the format `instance_id:contact_flow_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Amazon Connect instance.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"contact_flow_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the contact flow.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"arn": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ARN of the contact flow.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the contact flow. Must be unique within the instance. Name changes are applied in-place via `UpdateContactFlowMetadata`.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Description of the contact flow.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Type of the contact flow. Valid values: `CONTACT_FLOW`, `CUSTOMER_QUEUE`, " +
					"`CUSTOMER_HOLD`, `CUSTOMER_WHISPER`, `AGENT_HOLD`, `AGENT_WHISPER`, `OUTBOUND_WHISPER`, " +
					"`AGENT_TRANSFER`, `QUEUE_TRANSFER`. Changing the type requires replacement.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "JSON-encoded contact flow content. Updates are applied in-place via `UpdateContactFlowContent` — no resource replacement needed.",
			},
			"state": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "State of the contact flow: `ACTIVE` or `ARCHIVED`. Updated in-place.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tags": schema.MapAttribute{
				Optional:            true,
				ElementType:         frameworktypes.StringType,
				MarkdownDescription: "Tags to assign to the contact flow.",
			},
		},
	}
}

func (r *ConnectContactFlowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			fmt.Sprintf("Expected *ProviderClients, got: %T", req.ProviderData),
		)
		return
	}

	r.client = clients.Connect
}

func (r *ConnectContactFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectContactFlowResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &connect.CreateContactFlowInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		Name:       aws.String(data.Name.ValueString()),
		Type:       types.ContactFlowType(data.Type.ValueString()),
		Content:    aws.String(data.Content.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make(map[string]string)
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Tags = tags
	}

	tflog.Debug(ctx, "Creating Connect Contact Flow", map[string]any{
		"instance_id": data.InstanceID.ValueString(),
		"name":        data.Name.ValueString(),
		"type":        data.Type.ValueString(),
	})

	out, err := r.client.CreateContactFlow(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Connect Contact Flow",
			fmt.Sprintf("Unable to create contact flow %q: %s", data.Name.ValueString(), err),
		)
		return
	}

	data.ContactFlowID = frameworktypes.StringPointerValue(out.ContactFlowId)
	data.ARN = frameworktypes.StringPointerValue(out.ContactFlowArn)
	data.ID = frameworktypes.StringValue(
		fmt.Sprintf("%s:%s", data.InstanceID.ValueString(), aws.ToString(out.ContactFlowId)),
	)

	// Read back to populate computed fields (State, Description, etc.).
	resp.Diagnostics.Append(r.readAndPopulateModel(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Created Connect Contact Flow", map[string]any{
		"contact_flow_id": data.ContactFlowID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectContactFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectContactFlowResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.readAndPopulateModel(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is called for every non-replacement change. Content, name, description,
// state, and tags are all handled in-place — no replacement is ever triggered.
func (r *ConnectContactFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ConnectContactFlowResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := state.InstanceID.ValueString()
	flowID := state.ContactFlowID.ValueString()

	tflog.Debug(ctx, "Updating Connect Contact Flow", map[string]any{
		"instance_id":     instanceID,
		"contact_flow_id": flowID,
	})

	// Update flow content when it has changed.
	if !plan.Content.Equal(state.Content) {
		_, err := r.client.UpdateContactFlowContent(ctx, &connect.UpdateContactFlowContentInput{
			InstanceId:    aws.String(instanceID),
			ContactFlowId: aws.String(flowID),
			Content:       aws.String(plan.Content.ValueString()),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Connect Contact Flow content",
				fmt.Sprintf("Unable to update content for flow %s: %s", flowID, err),
			)
			return
		}
	}

	// Update metadata (name / description / state) when any of those changed.
	nameChanged := !plan.Name.Equal(state.Name)
	descChanged := !plan.Description.Equal(state.Description)
	stateChanged := !plan.State.Equal(state.State)

	if nameChanged || descChanged || stateChanged {
		metaInput := &connect.UpdateContactFlowMetadataInput{
			InstanceId:    aws.String(instanceID),
			ContactFlowId: aws.String(flowID),
			Name:          aws.String(plan.Name.ValueString()),
		}
		if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
			metaInput.Description = aws.String(plan.Description.ValueString())
		}
		if !plan.State.IsNull() && !plan.State.IsUnknown() {
			metaInput.ContactFlowState = types.ContactFlowState(plan.State.ValueString())
		}

		_, err := r.client.UpdateContactFlowMetadata(ctx, metaInput)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Connect Contact Flow metadata",
				fmt.Sprintf("Unable to update metadata for flow %s: %s", flowID, err),
			)
			return
		}
	}

	// Sync tags.
	if err := r.syncContactFlowTags(ctx, state.Tags, plan.Tags, state.ARN.ValueString()); err != nil {
		tflog.Warn(ctx, "Unable to sync tags for contact flow; tags may be stale",
			map[string]any{"error": err.Error()})
	}

	// Carry identifiers forward so readAndPopulateModel can use them.
	plan.ID = state.ID
	plan.ContactFlowID = state.ContactFlowID
	plan.ARN = state.ARN

	resp.Diagnostics.Append(r.readAndPopulateModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated Connect Contact Flow")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectContactFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectContactFlowResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Connect Contact Flow", map[string]any{
		"instance_id":     data.InstanceID.ValueString(),
		"contact_flow_id": data.ContactFlowID.ValueString(),
	})

	_, err := r.client.DeleteContactFlow(ctx, &connect.DeleteContactFlowInput{
		InstanceId:    aws.String(data.InstanceID.ValueString()),
		ContactFlowId: aws.String(data.ContactFlowID.ValueString()),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			// Already gone — treat as success.
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Connect Contact Flow",
			fmt.Sprintf("Unable to delete flow %s: %s", data.ContactFlowID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Connect Contact Flow")
}

func (r *ConnectContactFlowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: instance_id:contact_flow_id
	// This matches the ID format used by the standard aws_connect_contact_flow,
	// so existing resources can be imported seamlessly.
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format instance_id:contact_flow_id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("contact_flow_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// readAndPopulateModel calls DescribeContactFlow and overwrites all computed
// fields in data. Caller must have set InstanceID and ContactFlowID.
func (r *ConnectContactFlowResource) readAndPopulateModel(ctx context.Context, data *ConnectContactFlowResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	out, err := r.client.DescribeContactFlow(ctx, &connect.DescribeContactFlowInput{
		InstanceId:    aws.String(data.InstanceID.ValueString()),
		ContactFlowId: aws.String(data.ContactFlowID.ValueString()),
	})
	if err != nil {
		diags.AddError(
			"Error reading Connect Contact Flow",
			fmt.Sprintf("Unable to describe flow %s: %s", data.ContactFlowID.ValueString(), err),
		)
		return diags
	}

	flow := out.ContactFlow
	data.ContactFlowID = frameworktypes.StringPointerValue(flow.Id)
	data.ARN = frameworktypes.StringPointerValue(flow.Arn)
	data.ID = frameworktypes.StringValue(
		fmt.Sprintf("%s:%s", data.InstanceID.ValueString(), aws.ToString(flow.Id)),
	)
	data.Name = frameworktypes.StringPointerValue(flow.Name)
	data.Description = frameworktypes.StringPointerValue(flow.Description)
	data.Type = frameworktypes.StringValue(string(flow.Type))
	data.Content = frameworktypes.StringPointerValue(flow.Content)
	data.State = frameworktypes.StringValue(string(flow.State))

	if len(flow.Tags) > 0 {
		tags, d := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, flow.Tags)
		diags.Append(d...)
		data.Tags = tags
	} else if data.Tags.IsNull() {
		data.Tags = frameworktypes.MapValueMust(frameworktypes.StringType, map[string]attr.Value{})
	}

	return diags
}

// syncContactFlowTags diffs old vs. new tags and calls TagResource / UntagResource
// on the Connect client.
func (r *ConnectContactFlowResource) syncContactFlowTags(
	ctx context.Context,
	oldTags, newTags frameworktypes.Map,
	arn string,
) error {
	if arn == "" {
		return nil
	}

	old := make(map[string]string)
	if !oldTags.IsNull() && !oldTags.IsUnknown() {
		oldTags.ElementsAs(ctx, &old, false) //nolint:errcheck
	}
	desired := make(map[string]string)
	if !newTags.IsNull() && !newTags.IsUnknown() {
		newTags.ElementsAs(ctx, &desired, false) //nolint:errcheck
	}

	// Tags to add or update.
	add := make(map[string]string)
	for k, v := range desired {
		if oldVal, exists := old[k]; !exists || oldVal != v {
			add[k] = v
		}
	}
	if len(add) > 0 {
		_, err := r.client.TagResource(ctx, &connect.TagResourceInput{
			ResourceArn: aws.String(arn),
			Tags:        add,
		})
		if err != nil {
			return fmt.Errorf("TagResource failed: %w", err)
		}
	}

	// Tags to remove.
	var remove []string
	for k := range old {
		if _, exists := desired[k]; !exists {
			remove = append(remove, k)
		}
	}
	if len(remove) > 0 {
		_, err := r.client.UntagResource(ctx, &connect.UntagResourceInput{
			ResourceArn: aws.String(arn),
			TagKeys:     remove,
		})
		if err != nil {
			return fmt.Errorf("UntagResource failed: %w", err)
		}
	}

	return nil
}
