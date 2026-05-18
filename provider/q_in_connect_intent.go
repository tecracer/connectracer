// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &QInConnectIntentResource{}
var _ resource.ResourceWithImportState = &QInConnectIntentResource{}

func NewQInConnectIntentResource() resource.Resource {
	return &QInConnectIntentResource{}
}

// QInConnectIntentResource defines the resource implementation.
type QInConnectIntentResource struct {
	client *lexmodelsv2.Client
}

// QInConnectIntentResourceModel describes the resource data model.
type QInConnectIntentResourceModel struct {
	ID                            types.String                      `tfsdk:"id"`
	BotID                         types.String                      `tfsdk:"bot_id"`
	BotVersion                    types.String                      `tfsdk:"bot_version"`
	LocaleID                      types.String                      `tfsdk:"locale_id"`
	IntentID                      types.String                      `tfsdk:"intent_id"`
	Name                          types.String                      `tfsdk:"name"`
	Description                   types.String                      `tfsdk:"description"`
	ParentIntentSignature         types.String                      `tfsdk:"parent_intent_signature"`
	CreationDateTime              types.String                      `tfsdk:"creation_date_time"`
	LastUpdatedDateTime           types.String                      `tfsdk:"last_updated_date_time"`
	SampleUtterances              []QInConnectSampleUtteranceModel  `tfsdk:"sample_utterance"`
	QInConnectIntentConfiguration []QInConnectIntentConfigModel     `tfsdk:"q_in_connect_intent_configuration"`
	FulfillmentCodeHook           []QInConnectFulfillmentCodeHook   `tfsdk:"fulfillment_code_hook"`
}

type QInConnectSampleUtteranceModel struct {
	Utterance types.String `tfsdk:"utterance"`
}

type QInConnectIntentConfigModel struct {
	QInConnectAssistantConfiguration []QInConnectAssistantConfigModel `tfsdk:"q_in_connect_assistant_configuration"`
}

type QInConnectAssistantConfigModel struct {
	AssistantArn types.String `tfsdk:"assistant_arn"`
}

// Fulfillment code hook — kept minimal: just enabled + active flags.
// The success response message and next-step are hardcoded per the Q-in-Connect
// built-in intent pattern and are not configurable.
type QInConnectFulfillmentCodeHook struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Active  types.Bool `tfsdk:"active"`
}

func (r *QInConnectIntentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_q_in_connect_intent"
}

func (r *QInConnectIntentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Lex V2 Intent configured for Q in Connect (`AMAZON.QInConnectIntent`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite ID: `bot_id/bot_version/locale_id/intent_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bot_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Lex V2 bot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bot_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Version of the bot (e.g. `DRAFT`).",
			},
			"locale_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Locale ID of the bot (e.g. `en_US`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"intent_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier assigned to the intent by Lex.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the intent.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the intent.",
			},
			"parent_intent_signature": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Built-in intent signature this intent is derived from (e.g. `AMAZON.QInConnectIntent`).",
			},
			"creation_date_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp when the intent was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated_date_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp when the intent was last updated.",
			},
		},
		Blocks: map[string]schema.Block{
			"sample_utterance": schema.ListNestedBlock{
				MarkdownDescription: "Sample utterances for the intent.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"utterance": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Sample utterance text.",
						},
					},
				},
			},
			"q_in_connect_intent_configuration": schema.ListNestedBlock{
				MarkdownDescription: "Q in Connect configuration that links this intent to an Amazon Q in Connect assistant.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"q_in_connect_assistant_configuration": schema.ListNestedBlock{
							MarkdownDescription: "Configuration for the Q in Connect assistant.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"assistant_arn": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "ARN of the Amazon Q in Connect assistant.",
										Validators: []validator.String{
											stringvalidator.RegexMatches(
												regexache.MustCompile(`^arn:[a-z-]+?:[a-z-]+?:[a-z0-9-]+:[0-9]{12}:[a-zA-Z0-9_.:/=+\-@]+$`),
												"must be a valid ARN",
											),
										},
									},
								},
							},
						},
					},
				},
			},
			"fulfillment_code_hook": schema.ListNestedBlock{
				MarkdownDescription: "Fulfillment code hook settings. For Q in Connect intents `enabled` is typically `false` and `active` is `true`.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"enabled": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the Lambda fulfillment hook is enabled.",
						},
						"active": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the fulfillment hook is active.",
						},
					},
				},
			},
		},
	}
}

func (r *QInConnectIntentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *QInConnectIntentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data QInConnectIntentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &lexmodelsv2.CreateIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentName: aws.String(data.Name.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}
	if !data.ParentIntentSignature.IsNull() && !data.ParentIntentSignature.IsUnknown() {
		input.ParentIntentSignature = aws.String(data.ParentIntentSignature.ValueString())
	}

	input.SampleUtterances = expandSampleUtterances(data.SampleUtterances)
	input.QInConnectIntentConfiguration = expandQInConnectIntentConfig(data.QInConnectIntentConfiguration)
	input.FulfillmentCodeHook = expandFulfillmentCodeHook(data.FulfillmentCodeHook)

	tflog.Debug(ctx, "Creating Q in Connect Intent", map[string]any{
		"bot_id":     data.BotID.ValueString(),
		"bot_version": data.BotVersion.ValueString(),
		"locale_id":  data.LocaleID.ValueString(),
		"name":       data.Name.ValueString(),
	})

	out, err := r.client.CreateIntent(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Q in Connect Intent",
			fmt.Sprintf("Unable to create intent %q: %s", data.Name.ValueString(), err),
		)
		return
	}

	data.IntentID = types.StringPointerValue(out.IntentId)
	data.ID = types.StringValue(intentCompositeID(
		data.BotID.ValueString(),
		data.BotVersion.ValueString(),
		data.LocaleID.ValueString(),
		aws.ToString(out.IntentId),
	))

	// Read back to populate computed timestamps
	described, err := r.describeIntent(ctx, data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString(), aws.ToString(out.IntentId))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Q in Connect Intent after create", err.Error())
		return
	}
	data.CreationDateTime = flattenTime(described.CreationDateTime)
	data.LastUpdatedDateTime = flattenTime(described.LastUpdatedDateTime)

	tflog.Trace(ctx, "Created Q in Connect Intent", map[string]any{"intent_id": aws.ToString(out.IntentId)})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *QInConnectIntentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data QInConnectIntentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.describeIntent(ctx, data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString(), data.IntentID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Q in Connect Intent", err.Error())
		return
	}

	data.Name = types.StringPointerValue(out.IntentName)
	data.Description = types.StringPointerValue(out.Description)
	data.ParentIntentSignature = types.StringPointerValue(out.ParentIntentSignature)
	data.CreationDateTime = flattenTime(out.CreationDateTime)
	data.LastUpdatedDateTime = flattenTime(out.LastUpdatedDateTime)
	data.SampleUtterances = flattenSampleUtterances(out.SampleUtterances)
	data.QInConnectIntentConfiguration = flattenQInConnectIntentConfig(out.QInConnectIntentConfiguration)
	data.FulfillmentCodeHook = flattenFulfillmentCodeHook(out.FulfillmentCodeHook)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *QInConnectIntentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data QInConnectIntentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &lexmodelsv2.UpdateIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentId:   aws.String(data.IntentID.ValueString()),
		IntentName: aws.String(data.Name.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}
	if !data.ParentIntentSignature.IsNull() && !data.ParentIntentSignature.IsUnknown() {
		input.ParentIntentSignature = aws.String(data.ParentIntentSignature.ValueString())
	}

	input.SampleUtterances = expandSampleUtterances(data.SampleUtterances)
	input.QInConnectIntentConfiguration = expandQInConnectIntentConfig(data.QInConnectIntentConfiguration)
	input.FulfillmentCodeHook = expandFulfillmentCodeHook(data.FulfillmentCodeHook)

	tflog.Debug(ctx, "Updating Q in Connect Intent", map[string]any{"intent_id": data.IntentID.ValueString()})

	_, err := r.client.UpdateIntent(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Q in Connect Intent",
			fmt.Sprintf("Unable to update intent %q: %s", data.IntentID.ValueString(), err),
		)
		return
	}

	// Read back updated timestamps
	out, err := r.describeIntent(ctx, data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString(), data.IntentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Q in Connect Intent after update", err.Error())
		return
	}
	data.LastUpdatedDateTime = flattenTime(out.LastUpdatedDateTime)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *QInConnectIntentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data QInConnectIntentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Q in Connect Intent", map[string]any{"intent_id": data.IntentID.ValueString()})

	_, err := r.client.DeleteIntent(ctx, &lexmodelsv2.DeleteIntentInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
		IntentId:   aws.String(data.IntentID.ValueString()),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Q in Connect Intent",
			fmt.Sprintf("Unable to delete intent %q: %s", data.IntentID.ValueString(), err),
		)
	}
}

func (r *QInConnectIntentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: bot_id/bot_version/locale_id/intent_id
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format bot_id/bot_version/locale_id/intent_id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bot_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bot_version"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("locale_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("intent_id"), parts[3])...)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (r *QInConnectIntentResource) describeIntent(ctx context.Context, botID, botVersion, localeID, intentID string) (*lexmodelsv2.DescribeIntentOutput, error) {
	out, err := r.client.DescribeIntent(ctx, &lexmodelsv2.DescribeIntentInput{
		BotId:      aws.String(botID),
		BotVersion: aws.String(botVersion),
		LocaleId:   aws.String(localeID),
		IntentId:   aws.String(intentID),
	})
	if err != nil {
		return nil, fmt.Errorf("describing intent %q: %w", intentID, err)
	}
	return out, nil
}

func intentCompositeID(botID, botVersion, localeID, intentID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", botID, botVersion, localeID, intentID)
}

func flattenTime(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.UTC().Format(time.RFC3339))
}

// ── expand (Terraform model → AWS SDK) ───────────────────────────────────────

func expandSampleUtterances(in []QInConnectSampleUtteranceModel) []awstypes.SampleUtterance {
	if len(in) == 0 {
		return nil
	}
	out := make([]awstypes.SampleUtterance, len(in))
	for i, u := range in {
		out[i] = awstypes.SampleUtterance{Utterance: aws.String(u.Utterance.ValueString())}
	}
	return out
}

func expandQInConnectIntentConfig(in []QInConnectIntentConfigModel) *awstypes.QInConnectIntentConfiguration {
	if len(in) == 0 {
		return nil
	}
	cfg := in[0]
	out := &awstypes.QInConnectIntentConfiguration{}
	if len(cfg.QInConnectAssistantConfiguration) > 0 {
		out.QInConnectAssistantConfiguration = &awstypes.QInConnectAssistantConfiguration{
			AssistantArn: aws.String(cfg.QInConnectAssistantConfiguration[0].AssistantArn.ValueString()),
		}
	}
	return out
}

func expandFulfillmentCodeHook(in []QInConnectFulfillmentCodeHook) *awstypes.FulfillmentCodeHookSettings {
	if len(in) == 0 {
		return nil
	}
	h := in[0]
	out := &awstypes.FulfillmentCodeHookSettings{
		Enabled: h.Enabled.ValueBool(),
		Active:  aws.Bool(h.Active.ValueBool()),
	}
	return out
}

// ── flatten (AWS SDK → Terraform model) ──────────────────────────────────────

func flattenSampleUtterances(in []awstypes.SampleUtterance) []QInConnectSampleUtteranceModel {
	if len(in) == 0 {
		return nil
	}
	out := make([]QInConnectSampleUtteranceModel, len(in))
	for i, u := range in {
		out[i] = QInConnectSampleUtteranceModel{Utterance: types.StringPointerValue(u.Utterance)}
	}
	return out
}

func flattenQInConnectIntentConfig(in *awstypes.QInConnectIntentConfiguration) []QInConnectIntentConfigModel {
	if in == nil {
		return nil
	}
	cfgModel := QInConnectIntentConfigModel{}
	if in.QInConnectAssistantConfiguration != nil {
		cfgModel.QInConnectAssistantConfiguration = []QInConnectAssistantConfigModel{
			{AssistantArn: types.StringPointerValue(in.QInConnectAssistantConfiguration.AssistantArn)},
		}
	}
	return []QInConnectIntentConfigModel{cfgModel}
}

func flattenFulfillmentCodeHook(in *awstypes.FulfillmentCodeHookSettings) []QInConnectFulfillmentCodeHook {
	if in == nil {
		return nil
	}
	return []QInConnectFulfillmentCodeHook{
		{
			Enabled: types.BoolValue(in.Enabled),
			Active:  types.BoolPointerValue(in.Active),
		},
	}
}
