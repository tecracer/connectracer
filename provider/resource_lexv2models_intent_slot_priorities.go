// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	lexv2types "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &LexV2ModelsIntentSlotPrioritiesResource{}
var _ resource.ResourceWithImportState = &LexV2ModelsIntentSlotPrioritiesResource{}

func NewLexV2ModelsIntentSlotPrioritiesResource() resource.Resource {
	return &LexV2ModelsIntentSlotPrioritiesResource{}
}

// LexV2ModelsIntentSlotPrioritiesResource sets the slot elicitation order of an intent
// from a resource of its own, which is the only way to express it in Terraform: a slot
// is created with the id of its intent, so an intent that referenced its slots would
// close the loop.
type LexV2ModelsIntentSlotPrioritiesResource struct {
	client *lexmodelsv2.Client
}

type LexV2ModelsIntentSlotPrioritiesResourceModel struct {
	ID             frameworktypes.String          `tfsdk:"id"`
	BotID          frameworktypes.String          `tfsdk:"bot_id"`
	BotVersion     frameworktypes.String          `tfsdk:"bot_version"`
	LocaleID       frameworktypes.String          `tfsdk:"locale_id"`
	IntentID       frameworktypes.String          `tfsdk:"intent_id"`
	SlotPriorities []LexV2ModelsSlotPriorityModel `tfsdk:"slot_priority"`
}

type LexV2ModelsSlotPriorityModel struct {
	Priority frameworktypes.Int64  `tfsdk:"priority"`
	SlotID   frameworktypes.String `tfsdk:"slot_id"`
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lexv2models_intent_slot_priorities"
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sets the slot elicitation order on an existing Amazon Lex V2 intent.\n\n" +
			"Lex refuses to build a locale whose intent has slots without priorities, but an " +
			"`aws_lexv2models_slot` is created with its intent's id, so declaring the priorities on the " +
			"intent itself is a dependency cycle. This resource breaks it: create the intent, then its " +
			"slots, then point this at both.\n\n" +
			"Every other field of the intent is read back and sent again unchanged, since `UpdateIntent` " +
			"replaces the whole intent.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (bot_id/bot_version/locale_id/intent_id)",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bot_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bot the intent belongs to",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bot_version": schema.StringAttribute{
				MarkdownDescription: "Version of the bot, normally `DRAFT`",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"locale_id": schema.StringAttribute{
				MarkdownDescription: "Locale of the intent, e.g. `de_DE`",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"intent_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the intent whose slots are being ordered",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},

		Blocks: map[string]schema.Block{
			"slot_priority": schema.ListNestedBlock{
				MarkdownDescription: "One entry per slot of the intent. Lex elicits slots in ascending priority.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Elicitation order, lowest first",
							Required:            true,
						},
						"slot_id": schema.StringAttribute{
							MarkdownDescription: "Identifier of the slot",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.LexModelsV2
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LexV2ModelsIntentSlotPrioritiesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Unable to set slot priorities", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(r.composeID(&data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LexV2ModelsIntentSlotPrioritiesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.describe(ctx, &data)
	if err != nil {
		if isLexNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read intent", err.Error())
		return
	}

	priorities := make([]LexV2ModelsSlotPriorityModel, 0, len(out.SlotPriorities))
	for _, p := range out.SlotPriorities {
		priorities = append(priorities, LexV2ModelsSlotPriorityModel{
			Priority: frameworktypes.Int64Value(int64(aws.ToInt32(p.Priority))),
			SlotID:   frameworktypes.StringValue(aws.ToString(p.SlotId)),
		})
	}
	data.SlotPriorities = priorities
	data.ID = frameworktypes.StringValue(r.composeID(&data))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LexV2ModelsIntentSlotPrioritiesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Unable to update slot priorities", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(r.composeID(&data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete clears the priorities again. It is best effort: the intent is usually destroyed
// in the same apply, and an intent that is already gone is not an error worth failing on.
func (r *LexV2ModelsIntentSlotPrioritiesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LexV2ModelsIntentSlotPrioritiesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.SlotPriorities = nil
	if err := r.apply(ctx, &data); err != nil && !isLexNotFound(err) {
		tflog.Warn(ctx, "Unable to clear slot priorities, continuing", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected bot_id/bot_version/locale_id/intent_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bot_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bot_version"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("locale_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("intent_id"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) composeID(data *LexV2ModelsIntentSlotPrioritiesResourceModel) string {
	return strings.Join([]string{
		data.BotID.ValueString(),
		data.BotVersion.ValueString(),
		data.LocaleID.ValueString(),
		data.IntentID.ValueString(),
	}, "/")
}

func (r *LexV2ModelsIntentSlotPrioritiesResource) describe(ctx context.Context, data *LexV2ModelsIntentSlotPrioritiesResourceModel) (*lexmodelsv2.DescribeIntentOutput, error) {
	return r.client.DescribeIntent(ctx, &lexmodelsv2.DescribeIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentId:   aws.String(data.IntentID.ValueString()),
	})
}

// apply reads the intent back and sends it again with the configured priorities.
// UpdateIntent replaces the whole intent, so every field has to be carried over: anything
// omitted here is silently dropped from the live object.
func (r *LexV2ModelsIntentSlotPrioritiesResource) apply(ctx context.Context, data *LexV2ModelsIntentSlotPrioritiesResourceModel) error {
	current, err := r.describe(ctx, data)
	if err != nil {
		return fmt.Errorf("reading intent: %w", err)
	}

	priorities := make([]lexv2types.SlotPriority, 0, len(data.SlotPriorities))
	for _, p := range data.SlotPriorities {
		priorities = append(priorities, lexv2types.SlotPriority{
			Priority: aws.Int32(int32(p.Priority.ValueInt64())),
			SlotId:   aws.String(p.SlotID.ValueString()),
		})
	}

	tflog.Debug(ctx, "Setting Lex V2 intent slot priorities", map[string]interface{}{
		"intent_id": data.IntentID.ValueString(),
		"count":     len(priorities),
	})

	_, err = r.client.UpdateIntent(ctx, &lexmodelsv2.UpdateIntentInput{
		BotId:                     aws.String(data.BotID.ValueString()),
		BotVersion:                aws.String(data.BotVersion.ValueString()),
		LocaleId:                  aws.String(data.LocaleID.ValueString()),
		IntentId:                  aws.String(data.IntentID.ValueString()),
		IntentName:                current.IntentName,
		Description:               current.Description,
		ParentIntentSignature:     current.ParentIntentSignature,
		SampleUtterances:          current.SampleUtterances,
		DialogCodeHook:            current.DialogCodeHook,
		FulfillmentCodeHook:       current.FulfillmentCodeHook,
		IntentConfirmationSetting: current.IntentConfirmationSetting,
		IntentClosingSetting:      current.IntentClosingSetting,
		InputContexts:             current.InputContexts,
		OutputContexts:            current.OutputContexts,
		KendraConfiguration:       current.KendraConfiguration,
		InitialResponseSetting:    current.InitialResponseSetting,
		QnAIntentConfiguration:    current.QnAIntentConfiguration,
		SlotPriorities:            priorities,
	})
	if err != nil {
		return fmt.Errorf("updating intent: %w", err)
	}

	return nil
}

func isLexNotFound(err error) bool {
	var nfe *lexv2types.ResourceNotFoundException
	return errors.As(err, &nfe)
}
