// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// flowModuleErrorDetail returns err's message, plus any Problems details AWS
// attached to an InvalidContactFlowModuleException. The generic error string
// (via Error()) omits Problems entirely, which is often where the actual
// validation reason lives — without this, failures surface with an empty
// message and no indication of what's actually wrong with the module content.
func flowModuleErrorDetail(err error) string {
	var invalidModuleErr *types.InvalidContactFlowModuleException
	if !errors.As(err, &invalidModuleErr) || len(invalidModuleErr.Problems) == 0 {
		return err.Error()
	}

	problems := make([]string, 0, len(invalidModuleErr.Problems))
	for _, p := range invalidModuleErr.Problems {
		problems = append(problems, aws.ToString(p.Message))
	}
	return fmt.Sprintf("%s (problems: %s)", err, strings.Join(problems, "; "))
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectFlowModuleToolResource{}
var _ resource.ResourceWithImportState = &ConnectFlowModuleToolResource{}

func NewConnectFlowModuleToolResource() resource.Resource {
	return &ConnectFlowModuleToolResource{}
}

// ConnectFlowModuleToolResource defines the resource implementation.
type ConnectFlowModuleToolResource struct {
	client *connect.Client
}

// ConnectFlowModuleToolResourceModel describes the resource data model.
type ConnectFlowModuleToolResourceModel struct {
	ID                        frameworktypes.String `tfsdk:"id"`
	InstanceID                frameworktypes.String `tfsdk:"instance_id"`
	Name                      frameworktypes.String `tfsdk:"name"`
	Description               frameworktypes.String `tfsdk:"description"`
	Content                   frameworktypes.String `tfsdk:"content"`
	Settings                  frameworktypes.String `tfsdk:"settings"`
	ExternalInvocationEnabled frameworktypes.Bool   `tfsdk:"external_invocation_enabled"`
	Arn                       frameworktypes.String `tfsdk:"arn"`
	FlowModuleContentSha256   frameworktypes.String `tfsdk:"flow_module_content_sha256"`
	Status                    frameworktypes.String `tfsdk:"status"`
	Tags                      frameworktypes.Map    `tfsdk:"tags"`
	Version                   frameworktypes.Int64  `tfsdk:"version"`
	MCPToolID                 frameworktypes.String `tfsdk:"mcp_tool_id"`
}

func (r *ConnectFlowModuleToolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_flow_module_tool"
}

func (r *ConnectFlowModuleToolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Connect Flow Module with `ExternalInvocationConfiguration` enabled, so it can be " +
			"invoked outside of a flow as a tool — for example by a Q in Connect orchestration AI agent (see " +
			"`connectracer_connect_security_profile_flow_module` to grant a security profile access to invoke it).\n\n" +
			"`external_invocation_enabled` can only be set at creation time; the underlying `CreateContactFlowModule` " +
			"API has no corresponding update operation, so changing it forces replacement.\n\n" +
			"Module-as-tool flows only support a restricted set of blocks (see the Amazon Connect admin guide's " +
			"\"Module as tool supported blocks\" list, e.g. `CheckStaffing`, `CheckHoursOfOperation`, `GetQueueMetrics`, " +
			"`InvokeLambdaFunction`) — invalid content is rejected by the API at create/update time.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the flow module.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Amazon Connect instance.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the flow module.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the flow module. Required — a flow module without a description " +
					"cannot be invoked as a tool (it won't be selectable when granting a security profile access to it, " +
					"even though the underlying `CreateContactFlowModule` API accepts an empty description).",
				Required: true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The JSON string that represents the content of the flow module (Amazon Connect Flow language).",
				Required:            true,
			},
			"settings": schema.StringAttribute{
				MarkdownDescription: "Serialized JSON string of the flow module Settings schema (input/output types for custom block modules).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_invocation_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the flow module can be invoked externally (as a tool), e.g. by a Q in Connect " +
					"orchestration AI agent. Defaults to `false`. Cannot be changed after creation — flipping this value " +
					"forces replacement of the flow module.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the flow module.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"flow_module_content_sha256": schema.StringAttribute{
				MarkdownDescription: "Hash of the module content for integrity verification.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the flow module (`SAVED` or `PUBLISHED`).",
				Computed:            true,
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "The version number released via `CreateContactFlowModuleVersion` for the current " +
					"`content`/`settings`. A flow module with `external_invocation_enabled = true` but no released version " +
					"cannot actually be invoked as a tool — it won't appear as grantable in any security profile's flow " +
					"module list, even though `ExternalInvocationConfiguration.Enabled` reads back as `true`. This resource " +
					"releases a new version automatically on creation and whenever `content`/`settings` change.",
				Computed: true,
			},
			"mcp_tool_id": schema.StringAttribute{
				MarkdownDescription: "The value to use as `tool_id` on `connectracer_connect_ai_tool` when registering this " +
					"flow module as a `MODEL_CONTEXT_PROTOCOL` tool on an AI agent. This is AWS's own format, reverse-engineered " +
					"from a console-created tool module (undocumented): `aws_custom_flows__<flow_module_id>_<version>`. The raw " +
					"flow module `id` alone is rejected by `UpdateAIAgent` with \"not found in MCP tools\".",
				Computed: true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "A map of tags to assign to the flow module.",
				Optional:            true,
				Computed:            true,
				ElementType:         frameworktypes.StringType,
			},
		},
	}
}

func (r *ConnectFlowModuleToolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectFlowModuleToolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectFlowModuleToolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &connect.CreateContactFlowModuleInput{
		Content:     aws.String(data.Content.ValueString()),
		InstanceId:  aws.String(data.InstanceID.ValueString()),
		Name:        aws.String(data.Name.ValueString()),
		Description: aws.String(data.Description.ValueString()),
	}

	if !data.Settings.IsNull() && !data.Settings.IsUnknown() {
		input.Settings = aws.String(data.Settings.ValueString())
	}
	if !data.ExternalInvocationEnabled.IsNull() && !data.ExternalInvocationEnabled.IsUnknown() {
		input.ExternalInvocationConfiguration = &types.ExternalInvocationConfiguration{
			Enabled: data.ExternalInvocationEnabled.ValueBool(),
		}
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

	tflog.Debug(ctx, "Creating Connect Flow Module Tool", map[string]any{
		"instance_id": data.InstanceID.ValueString(),
		"name":        data.Name.ValueString(),
	})

	output, err := r.client.CreateContactFlowModule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Connect Flow Module",
			fmt.Sprintf("Unable to create flow module %q: %s", data.Name.ValueString(), flowModuleErrorDetail(err)),
		)
		return
	}

	data.ID = frameworktypes.StringPointerValue(output.Id)
	data.Arn = frameworktypes.StringPointerValue(output.Arn)

	tflog.Trace(ctx, "Created Connect Flow Module Tool", map[string]any{"id": data.ID.ValueString()})

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	version, err := r.createVersion(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error releasing Connect Flow Module version",
			fmt.Sprintf("Unable to release a version for flow module %s: %s", data.ID.ValueString(), flowModuleErrorDetail(err)),
		)
		return
	}
	data.Version = frameworktypes.Int64Value(version)
	data.MCPToolID = frameworktypes.StringValue(mcpToolID(data.ID.ValueString(), version))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectFlowModuleToolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectFlowModuleToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.readAndPopulateModel(ctx, &data)
	if diags.HasError() {
		for _, d := range diags {
			if strings.Contains(d.Detail(), "ResourceNotFoundException") {
				resp.State.RemoveResource(ctx)
				return
			}
		}
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectFlowModuleToolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectFlowModuleToolResourceModel
	var state ConnectFlowModuleToolResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID
	data.Arn = state.Arn

	tflog.Debug(ctx, "Updating Connect Flow Module Tool", map[string]any{"id": data.ID.ValueString()})

	if !data.Name.Equal(state.Name) || !data.Description.Equal(state.Description) {
		metaInput := &connect.UpdateContactFlowModuleMetadataInput{
			ContactFlowModuleId: aws.String(data.ID.ValueString()),
			InstanceId:          aws.String(data.InstanceID.ValueString()),
			Name:                aws.String(data.Name.ValueString()),
			Description:         aws.String(data.Description.ValueString()),
		}

		_, err := r.client.UpdateContactFlowModuleMetadata(ctx, metaInput)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Connect Flow Module metadata",
				fmt.Sprintf("Unable to update metadata for flow module %s: %s", data.ID.ValueString(), flowModuleErrorDetail(err)),
			)
			return
		}
	}

	contentChanged := !data.Content.Equal(state.Content) || !data.Settings.Equal(state.Settings)
	if contentChanged {
		contentInput := &connect.UpdateContactFlowModuleContentInput{
			ContactFlowModuleId: aws.String(data.ID.ValueString()),
			InstanceId:          aws.String(data.InstanceID.ValueString()),
			Content:             aws.String(data.Content.ValueString()),
		}
		if !data.Settings.IsNull() && !data.Settings.IsUnknown() {
			contentInput.Settings = aws.String(data.Settings.ValueString())
		}

		_, err := r.client.UpdateContactFlowModuleContent(ctx, contentInput)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Connect Flow Module content",
				fmt.Sprintf("Unable to update content for flow module %s: %s", data.ID.ValueString(), flowModuleErrorDetail(err)),
			)
			return
		}
	}

	if !data.Tags.Equal(state.Tags) {
		if err := r.updateTags(ctx, data, state); err != nil {
			resp.Diagnostics.AddError("Error updating Connect Flow Module tags", err.Error())
			return
		}
	}

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if contentChanged {
		version, err := r.createVersion(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error releasing Connect Flow Module version",
				fmt.Sprintf("Unable to release a version for flow module %s: %s", data.ID.ValueString(), flowModuleErrorDetail(err)),
			)
			return
		}
		data.Version = frameworktypes.Int64Value(version)
	} else {
		data.Version = state.Version
	}
	data.MCPToolID = frameworktypes.StringValue(mcpToolID(data.ID.ValueString(), data.Version.ValueInt64()))

	tflog.Trace(ctx, "Updated Connect Flow Module Tool", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectFlowModuleToolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectFlowModuleToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Connect Flow Module Tool", map[string]any{"id": data.ID.ValueString()})

	_, err := r.client.DeleteContactFlowModule(ctx, &connect.DeleteContactFlowModuleInput{
		ContactFlowModuleId: aws.String(data.ID.ValueString()),
		InstanceId:          aws.String(data.InstanceID.ValueString()),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Connect Flow Module",
			fmt.Sprintf("Unable to delete flow module %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Connect Flow Module Tool", map[string]any{"id": data.ID.ValueString()})
}

func (r *ConnectFlowModuleToolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: instance_id/flow_module_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format instance_id/flow_module_id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// mcpToolID formats the value AWS expects as tool_id on connectracer_connect_ai_tool when
// registering a flow module as a MODEL_CONTEXT_PROTOCOL tool. Reverse-engineered from a
// console-created tool module's persisted ToolConfiguration (undocumented anywhere) —
// UpdateAIAgent rejects the raw flow module id with "not found in MCP tools".
func mcpToolID(flowModuleID string, version int64) string {
	return fmt.Sprintf("aws_custom_flows__%s_%d", flowModuleID, version)
}

// createVersion releases a new flow module version for the module's current content, keyed to
// its content hash. A flow module with external_invocation_enabled = true is not actually
// invocable as a tool — and won't appear as grantable in any security profile's flow module list
// — until at least one version has been released, even though ExternalInvocationConfiguration.Enabled
// already reads back as true beforehand.
func (r *ConnectFlowModuleToolResource) createVersion(ctx context.Context, data *ConnectFlowModuleToolResourceModel) (int64, error) {
	output, err := r.client.CreateContactFlowModuleVersion(ctx, &connect.CreateContactFlowModuleVersionInput{
		InstanceId:              aws.String(data.InstanceID.ValueString()),
		ContactFlowModuleId:     aws.String(data.ID.ValueString()),
		Description:             aws.String(data.Description.ValueString()),
		FlowModuleContentSha256: aws.String(data.FlowModuleContentSha256.ValueString()),
	})
	if err != nil {
		return 0, err
	}
	return aws.ToInt64(output.Version), nil
}

// readAndPopulateModel calls DescribeContactFlowModule and populates the model with the response.
func (r *ConnectFlowModuleToolResource) readAndPopulateModel(ctx context.Context, data *ConnectFlowModuleToolResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	output, err := r.client.DescribeContactFlowModule(ctx, &connect.DescribeContactFlowModuleInput{
		ContactFlowModuleId: aws.String(data.ID.ValueString()),
		InstanceId:          aws.String(data.InstanceID.ValueString()),
	})
	if err != nil {
		diags.AddError(
			"Error reading Connect Flow Module",
			fmt.Sprintf("Unable to describe flow module %s: %s", data.ID.ValueString(), err),
		)
		return diags
	}

	m := output.ContactFlowModule
	if m == nil {
		diags.AddError(
			"Error reading Connect Flow Module",
			"DescribeContactFlowModule response did not contain flow module data",
		)
		return diags
	}

	data.ID = frameworktypes.StringPointerValue(m.Id)
	data.Arn = frameworktypes.StringPointerValue(m.Arn)
	data.Name = frameworktypes.StringPointerValue(m.Name)
	data.Content = frameworktypes.StringPointerValue(m.Content)
	data.Status = frameworktypes.StringValue(string(m.Status))

	if m.Description != nil {
		data.Description = frameworktypes.StringPointerValue(m.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	if m.Settings != nil {
		data.Settings = frameworktypes.StringPointerValue(m.Settings)
	} else {
		data.Settings = frameworktypes.StringNull()
	}

	if m.FlowModuleContentSha256 != nil {
		data.FlowModuleContentSha256 = frameworktypes.StringPointerValue(m.FlowModuleContentSha256)
	} else {
		data.FlowModuleContentSha256 = frameworktypes.StringNull()
	}

	if m.ExternalInvocationConfiguration != nil {
		data.ExternalInvocationEnabled = frameworktypes.BoolValue(m.ExternalInvocationConfiguration.Enabled)
	} else {
		data.ExternalInvocationEnabled = frameworktypes.BoolValue(false)
	}

	// Always resolve to a (possibly empty) map, never null: tags is
	// Optional+Computed, and config commonly sets tags = {} (e.g. a variable
	// defaulting to {}) rather than omitting it — collapsing "no tags" to null
	// would then mismatch the planned empty-map value and Terraform reports
	// "Provider produced inconsistent result after apply". A nil Go map
	// reflects to a null MapValue, so normalize it to a non-nil empty map first.
	tags := m.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	tagsMap, tagDiags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tags)
	diags.Append(tagDiags...)
	if diags.HasError() {
		return diags
	}
	data.Tags = tagsMap

	return diags
}

// updateTags reconciles tag changes between state and plan via TagResource/UntagResource.
func (r *ConnectFlowModuleToolResource) updateTags(ctx context.Context, plan, state ConnectFlowModuleToolResourceModel) error {
	oldTags := make(map[string]string)
	newTags := make(map[string]string)

	if !state.Tags.IsNull() && !state.Tags.IsUnknown() {
		if diags := state.Tags.ElementsAs(ctx, &oldTags, false); diags.HasError() {
			return fmt.Errorf("failed to read prior tags")
		}
	}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		if diags := plan.Tags.ElementsAs(ctx, &newTags, false); diags.HasError() {
			return fmt.Errorf("failed to read planned tags")
		}
	}

	arn := plan.Arn.ValueString()
	add, remove := diffTags(oldTags, newTags)

	if len(add) > 0 {
		if _, err := r.client.TagResource(ctx, &connect.TagResourceInput{
			ResourceArn: aws.String(arn),
			Tags:        add,
		}); err != nil {
			return fmt.Errorf("TagResource failed: %w", err)
		}
	}

	if len(remove) > 0 {
		if _, err := r.client.UntagResource(ctx, &connect.UntagResourceInput{
			ResourceArn: aws.String(arn),
			TagKeys:     remove,
		}); err != nil {
			return fmt.Errorf("UntagResource failed: %w", err)
		}
	}

	return nil
}

// diffTags computes which tags need to be added/updated (add) and which keys need to be
// removed (remove) to bring oldTags to newTags. Pure and independent of any AWS client so
// it can be unit tested directly.
func diffTags(oldTags, newTags map[string]string) (add map[string]string, remove []string) {
	add = make(map[string]string)
	for k, v := range newTags {
		if oldVal, exists := oldTags[k]; !exists || oldVal != v {
			add[k] = v
		}
	}

	for k := range oldTags {
		if _, exists := newTags[k]; !exists {
			remove = append(remove, k)
		}
	}

	return add, remove
}
