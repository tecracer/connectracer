// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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

var _ resource.Resource = &LexV2ModelsIntentResource{}
var _ resource.ResourceWithImportState = &LexV2ModelsIntentResource{}

func NewLexV2ModelsIntentResource() resource.Resource {
	return &LexV2ModelsIntentResource{}
}

// LexV2ModelsIntentResource manages a Lex V2 intent without owning the parts of it that
// other resources set. `UpdateIntent` replaces the whole intent, so a resource that models
// a field implicitly deletes it when the config leaves it out. The official
// aws_lexv2models_intent models slot_priority that way, which makes it delete on every
// refresh whatever set the priorities, and the two never converge.
//
// This resource models only what it is given and carries everything else back unchanged.
type LexV2ModelsIntentResource struct {
	client *lexmodelsv2.Client
}

type LexV2ModelsIntentResourceModel struct {
	ID                    frameworktypes.String             `tfsdk:"id"`
	IntentID              frameworktypes.String             `tfsdk:"intent_id"`
	BotID                 frameworktypes.String             `tfsdk:"bot_id"`
	BotVersion            frameworktypes.String             `tfsdk:"bot_version"`
	LocaleID              frameworktypes.String             `tfsdk:"locale_id"`
	Name                  frameworktypes.String             `tfsdk:"name"`
	Description           frameworktypes.String             `tfsdk:"description"`
	ParentIntentSignature frameworktypes.String             `tfsdk:"parent_intent_signature"`
	SampleUtterances      []LexV2ModelsSampleUtteranceModel `tfsdk:"sample_utterance"`
}

type LexV2ModelsSampleUtteranceModel struct {
	Utterance frameworktypes.String `tfsdk:"utterance"`
}

func (r *LexV2ModelsIntentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lexv2models_intent"
}

func (r *LexV2ModelsIntentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Lex V2 intent, without deleting the parts of it that other " +
			"resources own.\n\n" +
			"`UpdateIntent` replaces the entire intent, so a resource that models a field deletes that " +
			"field whenever the configuration omits it. The official `aws_lexv2models_intent` models " +
			"`slot_priority` this way, which makes it plan the priorities away on every refresh while " +
			"`connectracer_lexv2models_intent_slot_priorities` plans them back. The two never converge, " +
			"and the usual advice of adding `lifecycle { ignore_changes = [slot_priority] }` is knowledge " +
			"a user has to have before the configuration is correct.\n\n" +
			"This resource models name, description, parent signature and sample utterances. Everything " +
			"else on the intent, including slot priorities, code hooks, confirmation and closing settings, " +
			"contexts, Kendra and QnA configuration, is read back and sent again unchanged, so whatever " +
			"set it keeps it.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (bot_id/bot_version/locale_id/intent_id)",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"intent_id": schema.StringAttribute{
				MarkdownDescription: "Identifier Lex assigned to the intent",
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
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the intent. `YesIntent` and `NoIntent` are reserved and fail the locale build.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the intent",
				Optional:            true,
			},
			"parent_intent_signature": schema.StringAttribute{
				MarkdownDescription: "Signature of the built-in intent this one derives from, e.g. `AMAZON.FallbackIntent`",
				Optional:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"sample_utterance": schema.ListNestedBlock{
				MarkdownDescription: "Utterances that match this intent. A locale needs at least one custom " +
					"intent with an utterance, otherwise the build fails with \"A locale must have at least " +
					"one custom intent with a valid utterance\". For an intent whose whole answer is one slot " +
					"value, the utterance is the slot itself, e.g. `{answer}`.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"utterance": schema.StringAttribute{
							MarkdownDescription: "The utterance text",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *LexV2ModelsIntentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LexV2ModelsIntentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LexV2ModelsIntentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Lex V2 intent", map[string]interface{}{"name": data.Name.ValueString()})

	out, err := r.client.CreateIntent(ctx, &lexmodelsv2.CreateIntentInput{
		BotId:                 aws.String(data.BotID.ValueString()),
		BotVersion:            aws.String(data.BotVersion.ValueString()),
		LocaleId:              aws.String(data.LocaleID.ValueString()),
		IntentName:            aws.String(data.Name.ValueString()),
		Description:           optionalString(data.Description),
		ParentIntentSignature: optionalString(data.ParentIntentSignature),
		SampleUtterances:      sampleUtterancesFromModel(data.SampleUtterances),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create intent", err.Error())
		return
	}

	data.IntentID = frameworktypes.StringValue(aws.ToString(out.IntentId))
	data.ID = frameworktypes.StringValue(r.composeID(&data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsIntentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LexV2ModelsIntentResourceModel
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

	data.Name = frameworktypes.StringValue(aws.ToString(out.IntentName))
	data.Description = optionalStringValue(out.Description)
	data.ParentIntentSignature = optionalStringValue(out.ParentIntentSignature)
	data.SampleUtterances = sampleUtterancesToModel(out.SampleUtterances)
	data.ID = frameworktypes.StringValue(r.composeID(&data))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update reads the intent back first and re-sends every field this resource does not model.
// That is what keeps slot priorities, code hooks and the rest alive across an update here.
func (r *LexV2ModelsIntentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LexV2ModelsIntentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.describe(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read intent before update", err.Error())
		return
	}

	_, err = r.client.UpdateIntent(ctx, &lexmodelsv2.UpdateIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentId:   aws.String(data.IntentID.ValueString()),

		IntentName:            aws.String(data.Name.ValueString()),
		Description:           optionalString(data.Description),
		ParentIntentSignature: optionalString(data.ParentIntentSignature),
		SampleUtterances:      sampleUtterancesFromModel(data.SampleUtterances),

		// Not modeled here, so carried over verbatim.
		SlotPriorities:            current.SlotPriorities,
		DialogCodeHook:            current.DialogCodeHook,
		FulfillmentCodeHook:       current.FulfillmentCodeHook,
		IntentConfirmationSetting: current.IntentConfirmationSetting,
		IntentClosingSetting:      current.IntentClosingSetting,
		InputContexts:             current.InputContexts,
		OutputContexts:            current.OutputContexts,
		KendraConfiguration:       current.KendraConfiguration,
		InitialResponseSetting:    current.InitialResponseSetting,
		QnAIntentConfiguration:    current.QnAIntentConfiguration,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update intent", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(r.composeID(&data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsIntentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LexV2ModelsIntentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteIntent(ctx, &lexmodelsv2.DeleteIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentId:   aws.String(data.IntentID.ValueString()),
	})
	if err != nil && !isLexNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete intent", err.Error())
	}
}

func (r *LexV2ModelsIntentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *LexV2ModelsIntentResource) composeID(data *LexV2ModelsIntentResourceModel) string {
	return strings.Join([]string{
		data.BotID.ValueString(),
		data.BotVersion.ValueString(),
		data.LocaleID.ValueString(),
		data.IntentID.ValueString(),
	}, "/")
}

func (r *LexV2ModelsIntentResource) describe(ctx context.Context, data *LexV2ModelsIntentResourceModel) (*lexmodelsv2.DescribeIntentOutput, error) {
	return r.client.DescribeIntent(ctx, &lexmodelsv2.DescribeIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentId:   aws.String(data.IntentID.ValueString()),
	})
}

func optionalString(v frameworktypes.String) *string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	return aws.String(v.ValueString())
}

func optionalStringValue(v *string) frameworktypes.String {
	if v == nil || *v == "" {
		return frameworktypes.StringNull()
	}
	return frameworktypes.StringValue(*v)
}

func sampleUtterancesFromModel(in []LexV2ModelsSampleUtteranceModel) []lexv2types.SampleUtterance {
	if len(in) == 0 {
		return nil
	}
	out := make([]lexv2types.SampleUtterance, 0, len(in))
	for _, u := range in {
		out = append(out, lexv2types.SampleUtterance{Utterance: aws.String(u.Utterance.ValueString())})
	}
	return out
}

func sampleUtterancesToModel(in []lexv2types.SampleUtterance) []LexV2ModelsSampleUtteranceModel {
	if len(in) == 0 {
		return nil
	}
	out := make([]LexV2ModelsSampleUtteranceModel, 0, len(in))
	for _, u := range in {
		out = append(out, LexV2ModelsSampleUtteranceModel{Utterance: frameworktypes.StringValue(aws.ToString(u.Utterance))})
	}
	return out
}
