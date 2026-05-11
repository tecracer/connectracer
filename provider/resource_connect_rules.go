// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConnectRuleResource{}
var _ resource.ResourceWithImportState = &ConnectRuleResource{}

func NewConnectRuleResource() resource.Resource {
	return &ConnectRuleResource{}
}

// ConnectRuleResource defines the resource implementation.
type ConnectRuleResource struct {
	client *connect.Client
}

// ConnectRuleResourceModel describes the resource data model.
type ConnectRuleResourceModel struct {
	ID                       frameworktypes.String `tfsdk:"id"`
	InstanceID               frameworktypes.String `tfsdk:"instance_id"`
	Name                     frameworktypes.String `tfsdk:"name"`
	Function                 frameworktypes.String `tfsdk:"function"`
	PublishStatus            frameworktypes.String `tfsdk:"publish_status"`
	EventSourceName          frameworktypes.String `tfsdk:"event_source_name"`
	IntegrationAssociationId frameworktypes.String `tfsdk:"integration_association_id"`
	ActionsJSON              frameworktypes.String `tfsdk:"actions_json"`
	RuleArn                  frameworktypes.String `tfsdk:"rule_arn"`
	Tags                     frameworktypes.Map    `tfsdk:"tags"`
	CreatedTime              frameworktypes.String `tfsdk:"created_time"`
	LastUpdatedTime          frameworktypes.String `tfsdk:"last_updated_time"`
	LastUpdatedBy            frameworktypes.String `tfsdk:"last_updated_by"`
}

// JSON model types for actions serialization/deserialization.

type RuleActionModel struct {
	ActionType                  string                           `json:"action_type"`
	EventBridgeAction           *EventBridgeActionModel          `json:"event_bridge_action,omitempty"`
	TaskAction                  *TaskActionModel                 `json:"task_action,omitempty"`
	SendNotificationAction      *SendNotificationActionModel     `json:"send_notification_action,omitempty"`
	AssignContactCategoryAction *struct{}                         `json:"assign_contact_category_action,omitempty"`
	EndAssociatedTasksAction    *struct{}                         `json:"end_associated_tasks_action,omitempty"`
	SubmitAutoEvaluationAction  *SubmitAutoEvaluationActionModel `json:"submit_auto_evaluation_action,omitempty"`
	CreateCaseAction            *CreateCaseActionModel           `json:"create_case_action,omitempty"`
	UpdateCaseAction            *UpdateCaseActionModel           `json:"update_case_action,omitempty"`
}

type EventBridgeActionModel struct {
	Name string `json:"name"`
}

type TaskActionModel struct {
	ContactFlowId string                    `json:"contact_flow_id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description,omitempty"`
	References    map[string]ReferenceModel `json:"references,omitempty"`
}

type ReferenceModel struct {
	Type         string `json:"type"`
	Value        string `json:"value,omitempty"`
	Arn          string `json:"arn,omitempty"`
	Status       string `json:"status,omitempty"`
	StatusReason string `json:"status_reason,omitempty"`
}

type SendNotificationActionModel struct {
	Content        string                      `json:"content"`
	ContentType    string                      `json:"content_type"`
	DeliveryMethod string                      `json:"delivery_method"`
	Recipient      *NotificationRecipientModel `json:"recipient,omitempty"`
	Subject        string                      `json:"subject,omitempty"`
}

type NotificationRecipientModel struct {
	UserIds  []string          `json:"user_ids,omitempty"`
	UserTags map[string]string `json:"user_tags,omitempty"`
}

type SubmitAutoEvaluationActionModel struct {
	EvaluationFormId string `json:"evaluation_form_id"`
}

type CreateCaseActionModel struct {
	Fields     []FieldValueModel `json:"fields,omitempty"`
	TemplateId string            `json:"template_id"`
}

type UpdateCaseActionModel struct {
	Fields []FieldValueModel `json:"fields,omitempty"`
}

type FieldValueModel struct {
	Id    string               `json:"id"`
	Value *FieldValueUnionModel `json:"value,omitempty"`
}

type FieldValueUnionModel struct {
	BooleanValue *bool    `json:"boolean_value,omitempty"`
	DoubleValue  *float64 `json:"double_value,omitempty"`
	EmptyValue   *bool    `json:"empty_value,omitempty"`
	StringValue  *string  `json:"string_value,omitempty"`
}

func (r *ConnectRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_rule"
}

func (r *ConnectRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS Connect Rule resource.\n\n" +
			"Rules allow you to trigger actions based on events in your Amazon Connect instance, " +
			"such as contact events, agent events, and more.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier for the rule (RuleId)",
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
				MarkdownDescription: "The name of the rule",
				Required:            true,
			},
			"function": schema.StringAttribute{
				MarkdownDescription: "The conditions of the rule (rule function as a JSON-like string)",
				Required:            true,
			},
			"publish_status": schema.StringAttribute{
				MarkdownDescription: "The publish status of the rule. Valid values: `DRAFT`, `PUBLISHED`",
				Required:            true,
			},
			"event_source_name": schema.StringAttribute{
				MarkdownDescription: "The name of the event source. The trigger event source cannot be changed after creation",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_association_id": schema.StringAttribute{
				MarkdownDescription: "The identifier for the integration association (only needed for certain event sources)",
				Optional:            true,
			},
			"actions_json": schema.StringAttribute{
				MarkdownDescription: "A JSON-encoded string representing the array of rule actions. " +
					"Each action object must include an `action_type` field and the corresponding action definition",
				Required: true,
			},
			"rule_arn": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the rule",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the rule",
				Optional:            true,
				ElementType:         frameworktypes.StringType,
			},
			"created_time": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the rule was created (RFC3339 format)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated_time": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the rule was last updated (RFC3339 format)",
				Computed:            true,
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "The Amazon Resource Name (ARN) of the user who last updated the rule",
				Computed:            true,
			},
		},
	}
}

func (r *ConnectRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectRuleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse actions JSON
	actions, err := expandRuleActions(data.ActionsJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing actions_json",
			fmt.Sprintf("Unable to parse actions_json: %s", err),
		)
		return
	}

	// Build the trigger event source
	triggerEventSource := &types.RuleTriggerEventSource{
		EventSourceName: types.EventSourceName(data.EventSourceName.ValueString()),
	}
	if !data.IntegrationAssociationId.IsNull() && !data.IntegrationAssociationId.IsUnknown() {
		triggerEventSource.IntegrationAssociationId = aws.String(data.IntegrationAssociationId.ValueString())
	}

	// Build CreateRule input
	input := &connect.CreateRuleInput{
		Actions:            actions,
		Function:           aws.String(data.Function.ValueString()),
		InstanceId:         aws.String(data.InstanceID.ValueString()),
		Name:               aws.String(data.Name.ValueString()),
		PublishStatus:      types.RulePublishStatus(data.PublishStatus.ValueString()),
		TriggerEventSource: triggerEventSource,
	}

	tflog.Debug(ctx, "Creating AWS Connect Rule", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"name":        data.Name.ValueString(),
	})

	// Create the rule
	output, err := r.client.CreateRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Connect Rule",
			fmt.Sprintf("Unable to create rule, got error: %s", err),
		)
		return
	}

	// Set the ID and ARN from creation response
	if output.RuleId != nil {
		data.ID = frameworktypes.StringPointerValue(output.RuleId)
	}
	if output.RuleArn != nil {
		data.RuleArn = frameworktypes.StringPointerValue(output.RuleArn)
	}

	tflog.Trace(ctx, "Created Connect Rule resource", map[string]interface{}{
		"rule_id": data.ID.ValueString(),
	})

	// Apply tags if provided
	if !data.Tags.IsNull() {
		tags := make(map[string]string)
		diags := data.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(tags) > 0 {
			tagInput := &connect.TagResourceInput{
				ResourceArn: aws.String(data.RuleArn.ValueString()),
				Tags:        tags,
			}
			_, err := r.client.TagResource(ctx, tagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error tagging Connect Rule",
					fmt.Sprintf("Unable to tag rule, got error: %s", err),
				)
				return
			}
		}
	}

	// Read back the rule to populate all computed fields
	describeInput := &connect.DescribeRuleInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		RuleId:     aws.String(data.ID.ValueString()),
	}

	describeOutput, err := r.client.DescribeRule(ctx, describeInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Connect Rule after creation",
			fmt.Sprintf("Unable to describe rule, got error: %s", err),
		)
		return
	}

	// Populate model from describe response
	diags := r.populateModelFromRule(ctx, describeOutput.Rule, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectRuleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS Connect Rule", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"rule_id":     data.ID.ValueString(),
	})

	// Describe the rule
	describeInput := &connect.DescribeRuleInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		RuleId:     aws.String(data.ID.ValueString()),
	}

	describeOutput, err := r.client.DescribeRule(ctx, describeInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Connect Rule",
			fmt.Sprintf("Unable to describe rule %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Populate model from describe response
	diags := r.populateModelFromRule(ctx, describeOutput.Rule, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectRuleResourceModel
	var state ConnectRuleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AWS Connect Rule", map[string]interface{}{
		"rule_id": data.ID.ValueString(),
	})

	// Parse actions JSON
	actions, err := expandRuleActions(data.ActionsJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing actions_json",
			fmt.Sprintf("Unable to parse actions_json: %s", err),
		)
		return
	}

	// Build UpdateRule input
	updateInput := &connect.UpdateRuleInput{
		Actions:       actions,
		Function:      aws.String(data.Function.ValueString()),
		InstanceId:    aws.String(data.InstanceID.ValueString()),
		Name:          aws.String(data.Name.ValueString()),
		PublishStatus: types.RulePublishStatus(data.PublishStatus.ValueString()),
		RuleId:        aws.String(data.ID.ValueString()),
	}

	_, err = r.client.UpdateRule(ctx, updateInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Connect Rule",
			fmt.Sprintf("Unable to update rule, got error: %s", err),
		)
		return
	}

	// Handle tag updates
	if !data.Tags.Equal(state.Tags) {
		resourceArn := state.RuleArn.ValueString()

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
					"Error removing tags from Connect Rule",
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
					"Error tagging Connect Rule",
					fmt.Sprintf("Unable to add tags, got error: %s", err),
				)
				return
			}
		}
	}

	// Read back the rule to refresh state
	describeInput := &connect.DescribeRuleInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		RuleId:     aws.String(data.ID.ValueString()),
	}

	describeOutput, err := r.client.DescribeRule(ctx, describeInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Connect Rule after update",
			fmt.Sprintf("Unable to describe rule, got error: %s", err),
		)
		return
	}

	// Populate model from describe response
	diags := r.populateModelFromRule(ctx, describeOutput.Rule, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated Connect Rule resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectRuleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AWS Connect Rule", map[string]interface{}{
		"instance_id": data.InstanceID.ValueString(),
		"rule_id":     data.ID.ValueString(),
	})

	// Delete the rule
	input := &connect.DeleteRuleInput{
		InstanceId: aws.String(data.InstanceID.ValueString()),
		RuleId:     aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Connect Rule",
			fmt.Sprintf("Unable to delete rule %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Connect Rule resource")
}

func (r *ConnectRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by composite ID: instance_id/rule_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: instance_id/rule_id, got: %s", req.ID),
		)
		return
	}

	instanceID := parts[0]
	ruleID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ruleID)...)
}

// populateModelFromRule populates the Terraform resource model from the AWS SDK Rule type.
func (r *ConnectRuleResource) populateModelFromRule(ctx context.Context, rule *types.Rule, data *ConnectRuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if rule == nil {
		diags.AddError(
			"Connect Rule Not Found",
			"The DescribeRule response did not contain a rule",
		)
		return diags
	}

	data.ID = frameworktypes.StringPointerValue(rule.RuleId)
	data.Name = frameworktypes.StringPointerValue(rule.Name)
	data.Function = frameworktypes.StringPointerValue(rule.Function)
	data.PublishStatus = frameworktypes.StringValue(string(rule.PublishStatus))
	data.RuleArn = frameworktypes.StringPointerValue(rule.RuleArn)

	// Trigger event source
	if rule.TriggerEventSource != nil {
		data.EventSourceName = frameworktypes.StringValue(string(rule.TriggerEventSource.EventSourceName))
		if rule.TriggerEventSource.IntegrationAssociationId != nil {
			data.IntegrationAssociationId = frameworktypes.StringPointerValue(rule.TriggerEventSource.IntegrationAssociationId)
		} else {
			data.IntegrationAssociationId = frameworktypes.StringNull()
		}
	}

	// Actions
	actionsJSON, err := flattenRuleActions(rule.Actions)
	if err != nil {
		diags.AddError(
			"Error serializing actions",
			fmt.Sprintf("Unable to serialize rule actions to JSON: %s", err),
		)
		return diags
	}
	data.ActionsJSON = frameworktypes.StringValue(actionsJSON)

	// Timestamps
	if rule.CreatedTime != nil {
		data.CreatedTime = frameworktypes.StringValue(rule.CreatedTime.Format(time.RFC3339))
	}
	if rule.LastUpdatedTime != nil {
		data.LastUpdatedTime = frameworktypes.StringValue(rule.LastUpdatedTime.Format(time.RFC3339))
	}
	if rule.LastUpdatedBy != nil {
		data.LastUpdatedBy = frameworktypes.StringPointerValue(rule.LastUpdatedBy)
	}

	// Tags
	if len(rule.Tags) > 0 {
		tagsMap, tagDiags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, rule.Tags)
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

// expandRuleActions converts a JSON string to a slice of SDK RuleAction types.
func expandRuleActions(actionsJSON string) ([]types.RuleAction, error) {
	var actionModels []RuleActionModel
	if err := json.Unmarshal([]byte(actionsJSON), &actionModels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal actions JSON: %w", err)
	}

	actions := make([]types.RuleAction, 0, len(actionModels))
	for _, model := range actionModels {
		action := types.RuleAction{
			ActionType: types.ActionType(model.ActionType),
		}

		if model.EventBridgeAction != nil {
			action.EventBridgeAction = &types.EventBridgeActionDefinition{
				Name: aws.String(model.EventBridgeAction.Name),
			}
		}

		if model.TaskAction != nil {
			taskAction := &types.TaskActionDefinition{
				ContactFlowId: aws.String(model.TaskAction.ContactFlowId),
				Name:          aws.String(model.TaskAction.Name),
			}
			if model.TaskAction.Description != "" {
				taskAction.Description = aws.String(model.TaskAction.Description)
			}
			if len(model.TaskAction.References) > 0 {
				refs := make(map[string]types.Reference)
				for key, refModel := range model.TaskAction.References {
					ref := types.Reference{
						Type: types.ReferenceType(refModel.Type),
					}
					if refModel.Value != "" {
						ref.Value = aws.String(refModel.Value)
					}
					if refModel.Arn != "" {
						ref.Arn = aws.String(refModel.Arn)
					}
					if refModel.Status != "" {
						ref.Status = types.ReferenceStatus(refModel.Status)
					}
					if refModel.StatusReason != "" {
						ref.StatusReason = aws.String(refModel.StatusReason)
					}
					refs[key] = ref
				}
				taskAction.References = refs
			}
			action.TaskAction = taskAction
		}

		if model.SendNotificationAction != nil {
			notifAction := &types.SendNotificationActionDefinition{
				Content:        aws.String(model.SendNotificationAction.Content),
				ContentType:    types.NotificationContentType(model.SendNotificationAction.ContentType),
				DeliveryMethod: types.NotificationDeliveryType(model.SendNotificationAction.DeliveryMethod),
			}
			if model.SendNotificationAction.Subject != "" {
				notifAction.Subject = aws.String(model.SendNotificationAction.Subject)
			}
			if model.SendNotificationAction.Recipient != nil {
				recipient := &types.NotificationRecipientType{}
				if len(model.SendNotificationAction.Recipient.UserIds) > 0 {
					recipient.UserIds = model.SendNotificationAction.Recipient.UserIds
				}
				if len(model.SendNotificationAction.Recipient.UserTags) > 0 {
					recipient.UserTags = model.SendNotificationAction.Recipient.UserTags
				}
				notifAction.Recipient = recipient
			}
			action.SendNotificationAction = notifAction
		}

		if model.AssignContactCategoryAction != nil {
			action.AssignContactCategoryAction = &types.AssignContactCategoryActionDefinition{}
		}

		if model.EndAssociatedTasksAction != nil {
			action.EndAssociatedTasksAction = &types.EndAssociatedTasksActionDefinition{}
		}

		if model.SubmitAutoEvaluationAction != nil {
			action.SubmitAutoEvaluationAction = &types.SubmitAutoEvaluationActionDefinition{
				EvaluationFormId: aws.String(model.SubmitAutoEvaluationAction.EvaluationFormId),
			}
		}

		if model.CreateCaseAction != nil {
			createCase := &types.CreateCaseActionDefinition{
				TemplateId: aws.String(model.CreateCaseAction.TemplateId),
			}
			if len(model.CreateCaseAction.Fields) > 0 {
				createCase.Fields = expandFieldValues(model.CreateCaseAction.Fields)
			}
			action.CreateCaseAction = createCase
		}

		if model.UpdateCaseAction != nil {
			updateCase := &types.UpdateCaseActionDefinition{}
			if len(model.UpdateCaseAction.Fields) > 0 {
				updateCase.Fields = expandFieldValues(model.UpdateCaseAction.Fields)
			}
			action.UpdateCaseAction = updateCase
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// expandFieldValues converts field value models to SDK types.
func expandFieldValues(fields []FieldValueModel) []types.FieldValue {
	result := make([]types.FieldValue, 0, len(fields))
	for _, f := range fields {
		fv := types.FieldValue{
			Id: aws.String(f.Id),
		}
		if f.Value != nil {
			union := &types.FieldValueUnion{}
			if f.Value.BooleanValue != nil {
				union.BooleanValue = *f.Value.BooleanValue
			}
			if f.Value.DoubleValue != nil {
				union.DoubleValue = f.Value.DoubleValue
			}
			if f.Value.EmptyValue != nil && *f.Value.EmptyValue {
				union.EmptyValue = &types.EmptyFieldValue{}
			}
			if f.Value.StringValue != nil {
				union.StringValue = f.Value.StringValue
			}
			fv.Value = union
		}
		result = append(result, fv)
	}
	return result
}

// flattenRuleActions converts SDK RuleAction types back to a JSON string.
func flattenRuleActions(actions []types.RuleAction) (string, error) {
	actionModels := make([]RuleActionModel, 0, len(actions))

	for _, action := range actions {
		model := RuleActionModel{
			ActionType: string(action.ActionType),
		}

		if action.EventBridgeAction != nil {
			model.EventBridgeAction = &EventBridgeActionModel{}
			if action.EventBridgeAction.Name != nil {
				model.EventBridgeAction.Name = *action.EventBridgeAction.Name
			}
		}

		if action.TaskAction != nil {
			taskModel := &TaskActionModel{}
			if action.TaskAction.ContactFlowId != nil {
				taskModel.ContactFlowId = *action.TaskAction.ContactFlowId
			}
			if action.TaskAction.Name != nil {
				taskModel.Name = *action.TaskAction.Name
			}
			if action.TaskAction.Description != nil {
				taskModel.Description = *action.TaskAction.Description
			}
			if len(action.TaskAction.References) > 0 {
				taskModel.References = make(map[string]ReferenceModel)
				for key, ref := range action.TaskAction.References {
					refModel := ReferenceModel{
						Type: string(ref.Type),
					}
					if ref.Value != nil {
						refModel.Value = *ref.Value
					}
					if ref.Arn != nil {
						refModel.Arn = *ref.Arn
					}
					if ref.Status != "" {
						refModel.Status = string(ref.Status)
					}
					if ref.StatusReason != nil {
						refModel.StatusReason = *ref.StatusReason
					}
					taskModel.References[key] = refModel
				}
			}
			model.TaskAction = taskModel
		}

		if action.SendNotificationAction != nil {
			notifModel := &SendNotificationActionModel{}
			if action.SendNotificationAction.Content != nil {
				notifModel.Content = *action.SendNotificationAction.Content
			}
			notifModel.ContentType = string(action.SendNotificationAction.ContentType)
			notifModel.DeliveryMethod = string(action.SendNotificationAction.DeliveryMethod)
			if action.SendNotificationAction.Subject != nil {
				notifModel.Subject = *action.SendNotificationAction.Subject
			}
			if action.SendNotificationAction.Recipient != nil {
				recipient := &NotificationRecipientModel{}
				if len(action.SendNotificationAction.Recipient.UserIds) > 0 {
					recipient.UserIds = action.SendNotificationAction.Recipient.UserIds
				}
				if len(action.SendNotificationAction.Recipient.UserTags) > 0 {
					recipient.UserTags = action.SendNotificationAction.Recipient.UserTags
				}
				notifModel.Recipient = recipient
			}
			model.SendNotificationAction = notifModel
		}

		if action.AssignContactCategoryAction != nil {
			model.AssignContactCategoryAction = &struct{}{}
		}

		if action.EndAssociatedTasksAction != nil {
			model.EndAssociatedTasksAction = &struct{}{}
		}

		if action.SubmitAutoEvaluationAction != nil {
			submitModel := &SubmitAutoEvaluationActionModel{}
			if action.SubmitAutoEvaluationAction.EvaluationFormId != nil {
				submitModel.EvaluationFormId = *action.SubmitAutoEvaluationAction.EvaluationFormId
			}
			model.SubmitAutoEvaluationAction = submitModel
		}

		if action.CreateCaseAction != nil {
			createCaseModel := &CreateCaseActionModel{}
			if action.CreateCaseAction.TemplateId != nil {
				createCaseModel.TemplateId = *action.CreateCaseAction.TemplateId
			}
			if len(action.CreateCaseAction.Fields) > 0 {
				createCaseModel.Fields = flattenFieldValues(action.CreateCaseAction.Fields)
			}
			model.CreateCaseAction = createCaseModel
		}

		if action.UpdateCaseAction != nil {
			updateCaseModel := &UpdateCaseActionModel{}
			if len(action.UpdateCaseAction.Fields) > 0 {
				updateCaseModel.Fields = flattenFieldValues(action.UpdateCaseAction.Fields)
			}
			model.UpdateCaseAction = updateCaseModel
		}

		actionModels = append(actionModels, model)
	}

	data, err := json.Marshal(actionModels)
	if err != nil {
		return "", fmt.Errorf("failed to marshal actions to JSON: %w", err)
	}

	return string(data), nil
}

// flattenFieldValues converts SDK FieldValue types to field value models.
func flattenFieldValues(fields []types.FieldValue) []FieldValueModel {
	result := make([]FieldValueModel, 0, len(fields))
	for _, f := range fields {
		fvm := FieldValueModel{}
		if f.Id != nil {
			fvm.Id = *f.Id
		}
		if f.Value != nil {
			union := &FieldValueUnionModel{}
			if f.Value.BooleanValue {
				boolVal := true
				union.BooleanValue = &boolVal
			}
			if f.Value.DoubleValue != nil {
				union.DoubleValue = f.Value.DoubleValue
			}
			if f.Value.EmptyValue != nil {
				emptyVal := true
				union.EmptyValue = &emptyVal
			}
			if f.Value.StringValue != nil {
				union.StringValue = f.Value.StringValue
			}
			fvm.Value = union
		}
		result = append(result, fvm)
	}
	return result
}
