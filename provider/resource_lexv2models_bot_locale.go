// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	lexv2types "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &LexV2ModelsBotLocaleResource{}
var _ resource.ResourceWithImportState = &LexV2ModelsBotLocaleResource{}

func NewLexV2ModelsBotLocaleResource() resource.Resource {
	return &LexV2ModelsBotLocaleResource{}
}

// LexV2ModelsBotLocaleResource defines the resource implementation.
type LexV2ModelsBotLocaleResource struct {
	client *lexmodelsv2.Client
}

// LexV2ModelsBotLocaleResourceModel describes the resource data model.
type LexV2ModelsBotLocaleResourceModel struct {
	ID                           frameworktypes.String                     `tfsdk:"id"`
	BotID                        frameworktypes.String                     `tfsdk:"bot_id"`
	BotVersion                   frameworktypes.String                     `tfsdk:"bot_version"`
	LocaleID                     frameworktypes.String                     `tfsdk:"locale_id"`
	LocaleName                   frameworktypes.String                     `tfsdk:"locale_name"`
	NluIntentConfidenceThreshold frameworktypes.Float64                    `tfsdk:"nlu_intent_confidence_threshold"`
	Description                  frameworktypes.String                     `tfsdk:"description"`
	SpeechDetectionSensitivity   frameworktypes.String                     `tfsdk:"speech_detection_sensitivity"`
	BotLocaleStatus              frameworktypes.String                     `tfsdk:"bot_locale_status"`
	CreationDateTime             frameworktypes.String                     `tfsdk:"creation_date_time"`
	LastUpdatedDateTime          frameworktypes.String                     `tfsdk:"last_updated_date_time"`
	FailureReasons               frameworktypes.List                       `tfsdk:"failure_reasons"`
	VoiceSettings                []BotLocaleVoiceSettingsModel             `tfsdk:"voice_settings"`
	SpeechRecognitionSettings    []BotLocaleSpeechRecognitionSettingsModel `tfsdk:"speech_recognition_settings"`
	UnifiedSpeechSettings        []BotLocaleUnifiedSpeechSettingsModel     `tfsdk:"unified_speech_settings"`
	GenerativeAISettings         []BotLocaleGenerativeAISettingsModel      `tfsdk:"generative_ai_settings"`
	AudioFillerSettings          []BotLocaleAudioFillerSettingsModel       `tfsdk:"audio_filler_settings"`
	BuildOnApply                 frameworktypes.Bool                       `tfsdk:"build_on_apply"`
}

type BotLocaleAudioFillerSettingsModel struct {
	Enabled                             frameworktypes.Bool   `tfsdk:"enabled"`
	AudioType                           frameworktypes.String `tfsdk:"audio_type"`
	StartDelayInMilliseconds            frameworktypes.Int64  `tfsdk:"start_delay_in_milliseconds"`
	MinimumPlayDurationInMilliseconds   frameworktypes.Int64  `tfsdk:"minimum_play_duration_in_milliseconds"`
	ResponseDeliveryDelayInMilliseconds frameworktypes.Int64  `tfsdk:"response_delivery_delay_in_milliseconds"`
}

type BotLocaleVoiceSettingsModel struct {
	VoiceID frameworktypes.String `tfsdk:"voice_id"`
	Engine  frameworktypes.String `tfsdk:"engine"`
}

type BotLocaleSpeechRecognitionSettingsModel struct {
	SpeechModelPreference frameworktypes.String             `tfsdk:"speech_model_preference"`
	SpeechModelConfig     []BotLocaleSpeechModelConfigModel `tfsdk:"speech_model_config"`
}

type BotLocaleSpeechModelConfigModel struct {
	DeepgramConfig []BotLocaleDeepgramConfigModel `tfsdk:"deepgram_config"`
}

type BotLocaleDeepgramConfigModel struct {
	APITokenSecretARN frameworktypes.String `tfsdk:"api_token_secret_arn"`
	ModelID           frameworktypes.String `tfsdk:"model_id"`
}

type BotLocaleUnifiedSpeechSettingsModel struct {
	SpeechFoundationModel []BotLocaleSpeechFoundationModelModel `tfsdk:"speech_foundation_model"`
}

type BotLocaleSpeechFoundationModelModel struct {
	ModelARN frameworktypes.String `tfsdk:"model_arn"`
	VoiceID  frameworktypes.String `tfsdk:"voice_id"`
}

type BotLocaleGenerativeAISettingsModel struct {
	BuildtimeSettings []BotLocaleBuildtimeSettingsModel `tfsdk:"buildtime_settings"`
	RuntimeSettings   []BotLocaleRuntimeSettingsModel   `tfsdk:"runtime_settings"`
}

type BotLocaleBuildtimeSettingsModel struct {
	DescriptiveBotBuilder     []BotLocaleBedrockGenSettingModel `tfsdk:"descriptive_bot_builder"`
	SampleUtteranceGeneration []BotLocaleBedrockGenSettingModel `tfsdk:"sample_utterance_generation"`
}

type BotLocaleBedrockGenSettingModel struct {
	Enabled                   frameworktypes.Bool              `tfsdk:"enabled"`
	BedrockModelSpecification []BotLocaleBedrockModelSpecModel `tfsdk:"bedrock_model_specification"`
}

type BotLocaleBedrockModelSpecModel struct {
	ModelARN     frameworktypes.String     `tfsdk:"model_arn"`
	CustomPrompt frameworktypes.String     `tfsdk:"custom_prompt"`
	TraceStatus  frameworktypes.String     `tfsdk:"trace_status"`
	Guardrail    []BotLocaleGuardrailModel `tfsdk:"guardrail"`
}

type BotLocaleGuardrailModel struct {
	Identifier frameworktypes.String `tfsdk:"identifier"`
	Version    frameworktypes.String `tfsdk:"version"`
}

type BotLocaleRuntimeSettingsModel struct {
	NluImprovement            []BotLocaleNluImprovementModel `tfsdk:"nlu_improvement"`
	SlotResolutionImprovement []BotLocaleSlotResolutionModel `tfsdk:"slot_resolution_improvement"`
}

type BotLocaleNluImprovementModel struct {
	Enabled                      frameworktypes.Bool                  `tfsdk:"enabled"`
	AssistedNluMode              frameworktypes.String                `tfsdk:"assisted_nlu_mode"`
	IntentDisambiguationSettings []BotLocaleIntentDisambiguationModel `tfsdk:"intent_disambiguation_settings"`
}

type BotLocaleIntentDisambiguationModel struct {
	Enabled                     frameworktypes.Bool   `tfsdk:"enabled"`
	CustomDisambiguationMessage frameworktypes.String `tfsdk:"custom_disambiguation_message"`
	MaxDisambiguationIntents    frameworktypes.Int64  `tfsdk:"max_disambiguation_intents"`
}

type BotLocaleSlotResolutionModel struct {
	Enabled                   frameworktypes.Bool              `tfsdk:"enabled"`
	BedrockModelSpecification []BotLocaleBedrockModelSpecModel `tfsdk:"bedrock_model_specification"`
}

func (r *LexV2ModelsBotLocaleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lexv2models_bot_locale"
}

func (r *LexV2ModelsBotLocaleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	guardrailBlock := schema.ListNestedBlock{
		MarkdownDescription: "Amazon Bedrock guardrail configuration",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"identifier": schema.StringAttribute{
					MarkdownDescription: "The identifier of the Bedrock guardrail",
					Required:            true,
				},
				"version": schema.StringAttribute{
					MarkdownDescription: "The version of the Bedrock guardrail",
					Required:            true,
				},
			},
		},
	}

	bedrockModelSpecBlock := schema.ListNestedBlock{
		MarkdownDescription: "Amazon Bedrock model specification for a generative AI feature",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"model_arn": schema.StringAttribute{
					MarkdownDescription: "The ARN of the Amazon Bedrock foundation model",
					Required:            true,
				},
				"custom_prompt": schema.StringAttribute{
					MarkdownDescription: "A custom system prompt to prepend when calling this model",
					Optional:            true,
				},
				"trace_status": schema.StringAttribute{
					MarkdownDescription: "Whether to enable tracing for this model call. Valid values: `ENABLED`, `DISABLED`",
					Optional:            true,
					Computed:            true,
				},
			},
			Blocks: map[string]schema.Block{
				"guardrail": guardrailBlock,
			},
		},
	}

	bedrockGenSettingBlock := schema.ListNestedBlock{
		MarkdownDescription: "Generative AI feature toggle with optional Bedrock model override",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{
					MarkdownDescription: "Whether this generative AI feature is enabled",
					Required:            true,
				},
			},
			Blocks: map[string]schema.Block{
				"bedrock_model_specification": bedrockModelSpecBlock,
			},
		},
	}

	intentDisambiguationBlock := schema.ListNestedBlock{
		MarkdownDescription: "Intent disambiguation settings for assisted NLU",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{
					MarkdownDescription: "Whether intent disambiguation is enabled",
					Required:            true,
				},
				"custom_disambiguation_message": schema.StringAttribute{
					MarkdownDescription: "Custom message to use when asking the user to disambiguate intents",
					Optional:            true,
				},
				"max_disambiguation_intents": schema.Int64Attribute{
					MarkdownDescription: "Maximum number of intents to present for disambiguation (1–5)",
					Optional:            true,
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Amazon Lex V2 bot locale.\n\n" +
			"Feature-complete alternative to the official `aws_lexv2models_bot_locale` resource.\n" +
			"Adds support for `unified_speech_settings` (including `speech_foundation_model`), " +
			"`speech_detection_sensitivity`, `speech_recognition_settings` with Deepgram support, " +
			"and all generative AI settings available in the Lex V2 CreateBotLocale / UpdateBotLocale API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier in the format `bot_id/bot_version/locale_id`",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bot_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Lex V2 bot (10 alphanumeric characters)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bot_version": schema.StringAttribute{
				MarkdownDescription: "The bot version to associate this locale with. Must be `DRAFT`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locale_id": schema.StringAttribute{
				MarkdownDescription: "The locale identifier (e.g. `en_US`, `de_DE`). See [supported languages](https://docs.aws.amazon.com/lexv2/latest/dg/how-languages.html)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locale_name": schema.StringAttribute{
				MarkdownDescription: "The human-readable name of the locale (computed by AWS)",
				Computed:            true,
			},
			"nlu_intent_confidence_threshold": schema.Float64Attribute{
				MarkdownDescription: "Confidence threshold (0.0–1.0) below which Amazon Lex inserts `AMAZON.FallbackIntent` into the result list",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the bot locale (max 2000 characters)",
				Optional:            true,
			},
			"speech_detection_sensitivity": schema.StringAttribute{
				MarkdownDescription: "Voice activity detection (VAD) sensitivity. Valid values: `Default`, `HighNoiseTolerance`, `MaximumNoiseTolerance`",
				Optional:            true,
				Computed:            true,
			},
			"bot_locale_status": schema.StringAttribute{
				MarkdownDescription: "Current status of the bot locale (e.g. `NotBuilt`, `Built`, `Failed`)",
				Computed:            true,
			},
			"creation_date_time": schema.StringAttribute{
				MarkdownDescription: "RFC3339 timestamp of when the locale was created",
				Computed:            true,
			},
			"last_updated_date_time": schema.StringAttribute{
				MarkdownDescription: "RFC3339 timestamp of when the locale was last updated",
				Computed:            true,
			},
			"failure_reasons": schema.ListAttribute{
				MarkdownDescription: "Failure reasons when `bot_locale_status` is `Failed`",
				Computed:            true,
				ElementType:         frameworktypes.StringType,
			},
			"build_on_apply": schema.BoolAttribute{
				MarkdownDescription: "When `true`, runs `BuildBotLocale` after create and update and waits until the locale reaches `Built`.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},

		Blocks: map[string]schema.Block{
			"voice_settings": schema.ListNestedBlock{
				MarkdownDescription: "Amazon Polly voice settings for voice interaction",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"voice_id": schema.StringAttribute{
							MarkdownDescription: "Amazon Polly voice identifier (e.g. `Joanna`)",
							Required:            true,
						},
						"engine": schema.StringAttribute{
							MarkdownDescription: "Polly engine. Valid values: `standard`, `neural`, `long-form`, `generative`",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},

			"speech_recognition_settings": schema.ListNestedBlock{
				MarkdownDescription: "Speech-to-text settings for the bot locale",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"speech_model_preference": schema.StringAttribute{
							MarkdownDescription: "Preferred speech model. Valid values: `Standard`, `Neural`, `Deepgram`",
							Optional:            true,
							Computed:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"speech_model_config": schema.ListNestedBlock{
							MarkdownDescription: "Provider-specific speech model configuration",
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"deepgram_config": schema.ListNestedBlock{
										MarkdownDescription: "Configuration for the Deepgram speech-to-text provider",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"api_token_secret_arn": schema.StringAttribute{
													MarkdownDescription: "ARN of the Secrets Manager secret containing the Deepgram API token",
													Required:            true,
												},
												"model_id": schema.StringAttribute{
													MarkdownDescription: "Deepgram model identifier",
													Optional:            true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},

			"unified_speech_settings": schema.ListNestedBlock{
				MarkdownDescription: "Unified speech settings combining recognition and synthesis via a foundation model. Not supported by the official `aws_lexv2models_bot_locale` resource.",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"speech_foundation_model": schema.ListNestedBlock{
							MarkdownDescription: "Foundation model for unified speech processing",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"model_arn": schema.StringAttribute{
										MarkdownDescription: "ARN of the Amazon Bedrock foundation model for speech",
										Required:            true,
									},
									"voice_id": schema.StringAttribute{
										MarkdownDescription: "Voice identifier for speech synthesis with the foundation model",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},

			"generative_ai_settings": schema.ListNestedBlock{
				MarkdownDescription: "Generative AI capabilities powered by Amazon Bedrock",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"buildtime_settings": schema.ListNestedBlock{
							MarkdownDescription: "Generative AI features that run at bot build time",
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"descriptive_bot_builder":     bedrockGenSettingBlock,
									"sample_utterance_generation": bedrockGenSettingBlock,
								},
							},
						},
						"runtime_settings": schema.ListNestedBlock{
							MarkdownDescription: "Generative AI features that run at bot runtime",
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"nlu_improvement": schema.ListNestedBlock{
										MarkdownDescription: "Assisted NLU improvement using generative AI",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"enabled": schema.BoolAttribute{
													MarkdownDescription: "Whether NLU improvement is enabled",
													Required:            true,
												},
												"assisted_nlu_mode": schema.StringAttribute{
													MarkdownDescription: "Assisted NLU mode. Valid values: `Primary`, `Fallback`",
													Optional:            true,
													Computed:            true,
												},
											},
											Blocks: map[string]schema.Block{
												"intent_disambiguation_settings": intentDisambiguationBlock,
											},
										},
									},
									"slot_resolution_improvement": schema.ListNestedBlock{
										MarkdownDescription: "Slot resolution improvement using generative AI",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"enabled": schema.BoolAttribute{
													MarkdownDescription: "Whether slot resolution improvement is enabled",
													Required:            true,
												},
											},
											Blocks: map[string]schema.Block{
												"bedrock_model_specification": bedrockModelSpecBlock,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"audio_filler_settings": schema.ListNestedBlock{
				MarkdownDescription: "Audio filler played while Amazon Lex processes the user's input, reducing perceived latency.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether audio filler playback is enabled",
							Required:            true,
						},
						"audio_type": schema.StringAttribute{
							MarkdownDescription: "The audio filler sound to play. Valid values: " +
								"`MELODY_CHIPPER_CHIME`, `MELODY_CURIOUS_CRAWL`, `MELODY_RISING_RIPPLE`, " +
								"`MELODY_PATIENT_PING`, `MELODY_PONDERING_PONG`, " +
								"`TYPING_KINETIC_KEYS`, `TYPING_QUIET_QWERTY`",
							Required: true,
						},
						"start_delay_in_milliseconds": schema.Int64Attribute{
							MarkdownDescription: "Time (ms) to wait after end of user utterance before starting audio filler. Valid range: 500–5000. Default: 2500",
							Optional:            true,
							Computed:            true,
						},
						"minimum_play_duration_in_milliseconds": schema.Int64Attribute{
							MarkdownDescription: "Minimum time (ms) the filler plays once started, even if the bot response is ready sooner. Valid range: 1000–5000. Default: 3000",
							Optional:            true,
							Computed:            true,
						},
						"response_delivery_delay_in_milliseconds": schema.Int64Attribute{
							MarkdownDescription: "Silent gap (ms) inserted between end of audio filler and start of bot response. Valid range: 200–1000. Default: 500",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *LexV2ModelsBotLocaleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LexV2ModelsBotLocaleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LexV2ModelsBotLocaleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &lexmodelsv2.CreateBotLocaleInput{
		BotId:                        aws.String(data.BotID.ValueString()),
		BotVersion:                   aws.String(data.BotVersion.ValueString()),
		LocaleId:                     aws.String(data.LocaleID.ValueString()),
		NluIntentConfidenceThreshold: aws.Float64(data.NluIntentConfidenceThreshold.ValueFloat64()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}
	if !data.SpeechDetectionSensitivity.IsNull() && !data.SpeechDetectionSensitivity.IsUnknown() {
		input.SpeechDetectionSensitivity = lexv2types.SpeechDetectionSensitivity(data.SpeechDetectionSensitivity.ValueString())
	}
	if len(data.VoiceSettings) > 0 {
		input.VoiceSettings = expandBotLocaleVoiceSettings(data.VoiceSettings)
	}
	if len(data.SpeechRecognitionSettings) > 0 {
		input.SpeechRecognitionSettings = expandBotLocaleSpeechRecognitionSettings(data.SpeechRecognitionSettings)
	}
	if len(data.UnifiedSpeechSettings) > 0 {
		input.UnifiedSpeechSettings = expandBotLocaleUnifiedSpeechSettings(data.UnifiedSpeechSettings)
	}
	if len(data.GenerativeAISettings) > 0 {
		input.GenerativeAISettings = expandBotLocaleGenerativeAISettings(data.GenerativeAISettings)
	}
	if len(data.AudioFillerSettings) > 0 {
		input.AudioFillerSettings = expandBotLocaleAudioFillerSettings(data.AudioFillerSettings)
	}

	tflog.Debug(ctx, "Creating Lex V2 bot locale", map[string]interface{}{
		"bot_id":      data.BotID.ValueString(),
		"bot_version": data.BotVersion.ValueString(),
		"locale_id":   data.LocaleID.ValueString(),
	})

	output, err := r.client.CreateBotLocale(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Lex V2 bot locale",
			fmt.Sprintf("Unable to create bot locale %q for bot %q: %s",
				data.LocaleID.ValueString(), data.BotID.ValueString(), err),
		)
		return
	}

	data.ID = frameworktypes.StringValue(fmt.Sprintf("%s/%s/%s",
		aws.ToString(output.BotId),
		aws.ToString(output.BotVersion),
		aws.ToString(output.LocaleId),
	))

	tflog.Trace(ctx, "Created Lex V2 bot locale", map[string]interface{}{"id": data.ID.ValueString()})

	// CreateBotLocale returns HTTP 202; wait until the locale leaves Creating state.
	if err := r.waitForBotLocaleNotCreating(ctx, data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString()); err != nil {
		resp.Diagnostics.AddWarning(
			"Lex V2 bot locale status wait timed out",
			fmt.Sprintf("Timed out waiting for bot locale to leave Creating state: %s", err),
		)
	}

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.BuildOnApply.IsNull() && data.BuildOnApply.ValueBool() {
		if err := buildBotLocaleAndWait(ctx, r.client, data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error building Lex V2 bot locale", err.Error())
			return
		}
		diags = r.readAndPopulateModel(ctx, &data)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsBotLocaleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LexV2ModelsBotLocaleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Lex V2 bot locale", map[string]interface{}{"id": data.ID.ValueString()})

	diags := r.readAndPopulateModel(ctx, &data)
	if diags.HasError() {
		for _, d := range diags {
			if d.Severity() == diag.SeverityError && strings.Contains(d.Detail(), "ResourceNotFoundException") {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsBotLocaleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LexV2ModelsBotLocaleResourceModel
	var state LexV2ModelsBotLocaleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	tflog.Debug(ctx, "Updating Lex V2 bot locale", map[string]interface{}{"id": data.ID.ValueString()})

	input := &lexmodelsv2.UpdateBotLocaleInput{
		BotId:                        aws.String(data.BotID.ValueString()),
		BotVersion:                   aws.String(data.BotVersion.ValueString()),
		LocaleId:                     aws.String(data.LocaleID.ValueString()),
		NluIntentConfidenceThreshold: aws.Float64(data.NluIntentConfidenceThreshold.ValueFloat64()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		input.Description = aws.String(data.Description.ValueString())
	}
	if !data.SpeechDetectionSensitivity.IsNull() && !data.SpeechDetectionSensitivity.IsUnknown() {
		input.SpeechDetectionSensitivity = lexv2types.SpeechDetectionSensitivity(data.SpeechDetectionSensitivity.ValueString())
	}
	if len(data.VoiceSettings) > 0 {
		input.VoiceSettings = expandBotLocaleVoiceSettings(data.VoiceSettings)
	}
	if len(data.SpeechRecognitionSettings) > 0 {
		input.SpeechRecognitionSettings = expandBotLocaleSpeechRecognitionSettings(data.SpeechRecognitionSettings)
	}
	if len(data.UnifiedSpeechSettings) > 0 {
		input.UnifiedSpeechSettings = expandBotLocaleUnifiedSpeechSettings(data.UnifiedSpeechSettings)
	}
	if len(data.GenerativeAISettings) > 0 {
		input.GenerativeAISettings = expandBotLocaleGenerativeAISettings(data.GenerativeAISettings)
	}
	if len(data.AudioFillerSettings) > 0 {
		input.AudioFillerSettings = expandBotLocaleAudioFillerSettings(data.AudioFillerSettings)
	}

	_, err := r.client.UpdateBotLocale(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Lex V2 bot locale",
			fmt.Sprintf("Unable to update bot locale %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	diags := r.readAndPopulateModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.BuildOnApply.IsNull() && data.BuildOnApply.ValueBool() {
		if err := buildBotLocaleAndWait(ctx, r.client, data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error building Lex V2 bot locale", err.Error())
			return
		}
		diags = r.readAndPopulateModel(ctx, &data)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Trace(ctx, "Updated Lex V2 bot locale")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LexV2ModelsBotLocaleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LexV2ModelsBotLocaleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Lex V2 bot locale", map[string]interface{}{"id": data.ID.ValueString()})

	_, err := r.client.DeleteBotLocale(ctx, &lexmodelsv2.DeleteBotLocaleInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Lex V2 bot locale",
			fmt.Sprintf("Unable to delete bot locale %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Lex V2 bot locale")
}

func (r *LexV2ModelsBotLocaleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: bot_id/bot_version/locale_id  (e.g. ABCDE12345/DRAFT/en_US)
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format bot_id/bot_version/locale_id, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bot_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bot_version"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("locale_id"), parts[2])...)
}

// readAndPopulateModel calls DescribeBotLocale and maps all response fields into data.
func (r *LexV2ModelsBotLocaleResource) readAndPopulateModel(ctx context.Context, data *LexV2ModelsBotLocaleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	output, err := r.client.DescribeBotLocale(ctx, &lexmodelsv2.DescribeBotLocaleInput{
		BotId:      aws.String(data.BotID.ValueString()),
		BotVersion: aws.String(data.BotVersion.ValueString()),
		LocaleId:   aws.String(data.LocaleID.ValueString()),
	})
	if err != nil {
		diags.AddError(
			"Error reading Lex V2 bot locale",
			fmt.Sprintf("Unable to describe bot locale %q (bot %q): %s",
				data.LocaleID.ValueString(), data.BotID.ValueString(), err),
		)
		return diags
	}

	data.BotID = frameworktypes.StringPointerValue(output.BotId)
	data.BotVersion = frameworktypes.StringPointerValue(output.BotVersion)
	data.LocaleID = frameworktypes.StringPointerValue(output.LocaleId)
	data.LocaleName = frameworktypes.StringPointerValue(output.LocaleName)
	data.BotLocaleStatus = frameworktypes.StringValue(string(output.BotLocaleStatus))

	if output.NluIntentConfidenceThreshold != nil {
		data.NluIntentConfidenceThreshold = frameworktypes.Float64Value(*output.NluIntentConfidenceThreshold)
	}

	if output.Description != nil {
		data.Description = frameworktypes.StringPointerValue(output.Description)
	} else {
		data.Description = frameworktypes.StringNull()
	}

	if output.SpeechDetectionSensitivity != "" {
		data.SpeechDetectionSensitivity = frameworktypes.StringValue(string(output.SpeechDetectionSensitivity))
	} else {
		data.SpeechDetectionSensitivity = frameworktypes.StringNull()
	}

	if output.CreationDateTime != nil {
		data.CreationDateTime = frameworktypes.StringValue(output.CreationDateTime.Format(time.RFC3339))
	} else {
		data.CreationDateTime = frameworktypes.StringNull()
	}

	if output.LastUpdatedDateTime != nil {
		data.LastUpdatedDateTime = frameworktypes.StringValue(output.LastUpdatedDateTime.Format(time.RFC3339))
	} else {
		data.LastUpdatedDateTime = frameworktypes.StringNull()
	}

	if len(output.FailureReasons) > 0 {
		listVal, listDiags := frameworktypes.ListValueFrom(ctx, frameworktypes.StringType, output.FailureReasons)
		diags.Append(listDiags...)
		data.FailureReasons = listVal
	} else {
		data.FailureReasons = frameworktypes.ListValueMust(frameworktypes.StringType, nil)
	}

	data.VoiceSettings = flattenBotLocaleVoiceSettings(output.VoiceSettings)
	data.SpeechRecognitionSettings = flattenBotLocaleSpeechRecognitionSettings(output.SpeechRecognitionSettings)
	data.UnifiedSpeechSettings = flattenBotLocaleUnifiedSpeechSettings(output.UnifiedSpeechSettings)
	data.GenerativeAISettings = flattenBotLocaleGenerativeAISettings(output.GenerativeAISettings)
	data.AudioFillerSettings = flattenBotLocaleAudioFillerSettings(output.AudioFillerSettings)

	return diags
}

// waitForBotLocaleNotCreating polls DescribeBotLocale until the status leaves Creating.
// CreateBotLocale is async (HTTP 202), so this is needed before reading back state.
func (r *LexV2ModelsBotLocaleResource) waitForBotLocaleNotCreating(ctx context.Context, botID, botVersion, localeID string) error {
	const (
		maxAttempts  = 30
		pollInterval = 3 * time.Second
	)
	for i := range maxAttempts {
		output, err := r.client.DescribeBotLocale(ctx, &lexmodelsv2.DescribeBotLocaleInput{
			BotId:      aws.String(botID),
			BotVersion: aws.String(botVersion),
			LocaleId:   aws.String(localeID),
		})
		if err != nil {
			var nfe *lexv2types.ResourceNotFoundException
			if errors.As(err, &nfe) {
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("polling bot locale status: %w", err)
		}
		if output.BotLocaleStatus != lexv2types.BotLocaleStatusCreating {
			return nil
		}
		tflog.Debug(ctx, "Waiting for bot locale to leave Creating state",
			map[string]interface{}{"attempt": i + 1})
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("bot locale %q did not leave Creating state after %d attempts", localeID, maxAttempts)
}

// ── Expand helpers (Terraform model → AWS SDK input types) ───────────────────

func expandBotLocaleVoiceSettings(m []BotLocaleVoiceSettingsModel) *lexv2types.VoiceSettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.VoiceSettings{VoiceId: aws.String(v.VoiceID.ValueString())}
	if !v.Engine.IsNull() && !v.Engine.IsUnknown() {
		out.Engine = lexv2types.VoiceEngine(v.Engine.ValueString())
	}
	return out
}

func expandBotLocaleSpeechRecognitionSettings(m []BotLocaleSpeechRecognitionSettingsModel) *lexv2types.SpeechRecognitionSettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.SpeechRecognitionSettings{}
	if !v.SpeechModelPreference.IsNull() && !v.SpeechModelPreference.IsUnknown() {
		out.SpeechModelPreference = lexv2types.SpeechModelPreference(v.SpeechModelPreference.ValueString())
	}
	if len(v.SpeechModelConfig) > 0 {
		out.SpeechModelConfig = expandBotLocaleSpeechModelConfig(v.SpeechModelConfig)
	}
	return out
}

func expandBotLocaleSpeechModelConfig(m []BotLocaleSpeechModelConfigModel) *lexv2types.SpeechModelConfig {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.SpeechModelConfig{}
	if len(v.DeepgramConfig) > 0 {
		dg := v.DeepgramConfig[0]
		out.DeepgramConfig = &lexv2types.DeepgramSpeechModelConfig{
			ApiTokenSecretArn: aws.String(dg.APITokenSecretARN.ValueString()),
		}
		if !dg.ModelID.IsNull() && !dg.ModelID.IsUnknown() {
			out.DeepgramConfig.ModelId = aws.String(dg.ModelID.ValueString())
		}
	}
	return out
}

func expandBotLocaleUnifiedSpeechSettings(m []BotLocaleUnifiedSpeechSettingsModel) *lexv2types.UnifiedSpeechSettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	if len(v.SpeechFoundationModel) == 0 {
		return nil
	}
	sfm := v.SpeechFoundationModel[0]
	out := &lexv2types.UnifiedSpeechSettings{
		SpeechFoundationModel: &lexv2types.SpeechFoundationModel{
			ModelArn: aws.String(sfm.ModelARN.ValueString()),
		},
	}
	if !sfm.VoiceID.IsNull() && !sfm.VoiceID.IsUnknown() {
		out.SpeechFoundationModel.VoiceId = aws.String(sfm.VoiceID.ValueString())
	}
	return out
}

func expandBotLocaleGenerativeAISettings(m []BotLocaleGenerativeAISettingsModel) *lexv2types.GenerativeAISettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.GenerativeAISettings{}
	if len(v.BuildtimeSettings) > 0 {
		out.BuildtimeSettings = expandBotLocaleBuildtimeSettings(v.BuildtimeSettings)
	}
	if len(v.RuntimeSettings) > 0 {
		out.RuntimeSettings = expandBotLocaleRuntimeSettings(v.RuntimeSettings)
	}
	return out
}

func expandBotLocaleBuildtimeSettings(m []BotLocaleBuildtimeSettingsModel) *lexv2types.BuildtimeSettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.BuildtimeSettings{}
	if len(v.DescriptiveBotBuilder) > 0 {
		s := v.DescriptiveBotBuilder[0]
		out.DescriptiveBotBuilder = &lexv2types.DescriptiveBotBuilderSpecification{
			Enabled:                   s.Enabled.ValueBool(),
			BedrockModelSpecification: expandBotLocaleBedrockModelSpec(s.BedrockModelSpecification),
		}
	}
	if len(v.SampleUtteranceGeneration) > 0 {
		s := v.SampleUtteranceGeneration[0]
		out.SampleUtteranceGeneration = &lexv2types.SampleUtteranceGenerationSpecification{
			Enabled:                   s.Enabled.ValueBool(),
			BedrockModelSpecification: expandBotLocaleBedrockModelSpec(s.BedrockModelSpecification),
		}
	}
	return out
}

func expandBotLocaleRuntimeSettings(m []BotLocaleRuntimeSettingsModel) *lexv2types.RuntimeSettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.RuntimeSettings{}
	if len(v.NluImprovement) > 0 {
		n := v.NluImprovement[0]
		nlu := &lexv2types.NluImprovementSpecification{Enabled: n.Enabled.ValueBool()}
		if !n.AssistedNluMode.IsNull() && !n.AssistedNluMode.IsUnknown() {
			nlu.AssistedNluMode = lexv2types.AssistedNluMode(n.AssistedNluMode.ValueString())
		}
		if len(n.IntentDisambiguationSettings) > 0 {
			ids := n.IntentDisambiguationSettings[0]
			idSetting := &lexv2types.IntentDisambiguationSettings{Enabled: ids.Enabled.ValueBool()}
			if !ids.CustomDisambiguationMessage.IsNull() && !ids.CustomDisambiguationMessage.IsUnknown() {
				idSetting.CustomDisambiguationMessage = aws.String(ids.CustomDisambiguationMessage.ValueString())
			}
			if !ids.MaxDisambiguationIntents.IsNull() && !ids.MaxDisambiguationIntents.IsUnknown() {
				v32 := int32(ids.MaxDisambiguationIntents.ValueInt64())
				idSetting.MaxDisambiguationIntents = &v32
			}
			nlu.IntentDisambiguationSettings = idSetting
		}
		out.NluImprovement = nlu
	}
	if len(v.SlotResolutionImprovement) > 0 {
		s := v.SlotResolutionImprovement[0]
		out.SlotResolutionImprovement = &lexv2types.SlotResolutionImprovementSpecification{
			Enabled:                   s.Enabled.ValueBool(),
			BedrockModelSpecification: expandBotLocaleBedrockModelSpec(s.BedrockModelSpecification),
		}
	}
	return out
}

func expandBotLocaleBedrockModelSpec(m []BotLocaleBedrockModelSpecModel) *lexv2types.BedrockModelSpecification {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.BedrockModelSpecification{ModelArn: aws.String(v.ModelARN.ValueString())}
	if !v.CustomPrompt.IsNull() && !v.CustomPrompt.IsUnknown() {
		out.CustomPrompt = aws.String(v.CustomPrompt.ValueString())
	}
	if !v.TraceStatus.IsNull() && !v.TraceStatus.IsUnknown() {
		out.TraceStatus = lexv2types.BedrockTraceStatus(v.TraceStatus.ValueString())
	}
	if len(v.Guardrail) > 0 {
		g := v.Guardrail[0]
		out.Guardrail = &lexv2types.BedrockGuardrailConfiguration{
			Identifier: aws.String(g.Identifier.ValueString()),
			Version:    aws.String(g.Version.ValueString()),
		}
	}
	return out
}

// ── Flatten helpers (AWS SDK response types → Terraform model) ────────────────

func flattenBotLocaleVoiceSettings(v *lexv2types.VoiceSettings) []BotLocaleVoiceSettingsModel {
	if v == nil {
		return nil
	}
	return []BotLocaleVoiceSettingsModel{{
		VoiceID: frameworktypes.StringPointerValue(v.VoiceId),
		Engine:  frameworktypes.StringValue(string(v.Engine)),
	}}
}

func flattenBotLocaleSpeechRecognitionSettings(v *lexv2types.SpeechRecognitionSettings) []BotLocaleSpeechRecognitionSettingsModel {
	if v == nil {
		return nil
	}
	m := BotLocaleSpeechRecognitionSettingsModel{
		SpeechModelConfig: flattenBotLocaleSpeechModelConfig(v.SpeechModelConfig),
	}
	if v.SpeechModelPreference != "" {
		m.SpeechModelPreference = frameworktypes.StringValue(string(v.SpeechModelPreference))
	} else {
		m.SpeechModelPreference = frameworktypes.StringNull()
	}
	return []BotLocaleSpeechRecognitionSettingsModel{m}
}

func flattenBotLocaleSpeechModelConfig(v *lexv2types.SpeechModelConfig) []BotLocaleSpeechModelConfigModel {
	if v == nil {
		return nil
	}
	m := BotLocaleSpeechModelConfigModel{}
	if v.DeepgramConfig != nil {
		dg := BotLocaleDeepgramConfigModel{
			APITokenSecretARN: frameworktypes.StringPointerValue(v.DeepgramConfig.ApiTokenSecretArn),
		}
		if v.DeepgramConfig.ModelId != nil {
			dg.ModelID = frameworktypes.StringPointerValue(v.DeepgramConfig.ModelId)
		} else {
			dg.ModelID = frameworktypes.StringNull()
		}
		m.DeepgramConfig = []BotLocaleDeepgramConfigModel{dg}
	}
	return []BotLocaleSpeechModelConfigModel{m}
}

func flattenBotLocaleUnifiedSpeechSettings(v *lexv2types.UnifiedSpeechSettings) []BotLocaleUnifiedSpeechSettingsModel {
	if v == nil || v.SpeechFoundationModel == nil {
		return nil
	}
	sfm := BotLocaleSpeechFoundationModelModel{
		ModelARN: frameworktypes.StringPointerValue(v.SpeechFoundationModel.ModelArn),
	}
	if v.SpeechFoundationModel.VoiceId != nil {
		sfm.VoiceID = frameworktypes.StringPointerValue(v.SpeechFoundationModel.VoiceId)
	} else {
		sfm.VoiceID = frameworktypes.StringNull()
	}
	return []BotLocaleUnifiedSpeechSettingsModel{{
		SpeechFoundationModel: []BotLocaleSpeechFoundationModelModel{sfm},
	}}
}

func flattenBotLocaleGenerativeAISettings(v *lexv2types.GenerativeAISettings) []BotLocaleGenerativeAISettingsModel {
	if v == nil {
		return nil
	}
	bt := flattenBotLocaleBuildtimeSettings(v.BuildtimeSettings)
	rt := flattenBotLocaleRuntimeSettings(v.RuntimeSettings)
	if bt == nil && rt == nil {
		return nil
	}
	return []BotLocaleGenerativeAISettingsModel{{BuildtimeSettings: bt, RuntimeSettings: rt}}
}

func flattenBotLocaleBuildtimeSettings(v *lexv2types.BuildtimeSettings) []BotLocaleBuildtimeSettingsModel {
	if v == nil {
		return nil
	}
	m := BotLocaleBuildtimeSettingsModel{}
	if v.DescriptiveBotBuilder != nil {
		m.DescriptiveBotBuilder = []BotLocaleBedrockGenSettingModel{{
			Enabled:                   frameworktypes.BoolValue(v.DescriptiveBotBuilder.Enabled),
			BedrockModelSpecification: flattenBotLocaleBedrockModelSpec(v.DescriptiveBotBuilder.BedrockModelSpecification),
		}}
	}
	if v.SampleUtteranceGeneration != nil {
		m.SampleUtteranceGeneration = []BotLocaleBedrockGenSettingModel{{
			Enabled:                   frameworktypes.BoolValue(v.SampleUtteranceGeneration.Enabled),
			BedrockModelSpecification: flattenBotLocaleBedrockModelSpec(v.SampleUtteranceGeneration.BedrockModelSpecification),
		}}
	}
	return []BotLocaleBuildtimeSettingsModel{m}
}

func flattenBotLocaleRuntimeSettings(v *lexv2types.RuntimeSettings) []BotLocaleRuntimeSettingsModel {
	if v == nil {
		return nil
	}
	m := BotLocaleRuntimeSettingsModel{}
	if v.NluImprovement != nil {
		n := v.NluImprovement
		nlu := BotLocaleNluImprovementModel{Enabled: frameworktypes.BoolValue(n.Enabled)}
		if n.AssistedNluMode != "" {
			nlu.AssistedNluMode = frameworktypes.StringValue(string(n.AssistedNluMode))
		} else {
			nlu.AssistedNluMode = frameworktypes.StringNull()
		}
		if n.IntentDisambiguationSettings != nil {
			ids := n.IntentDisambiguationSettings
			idm := BotLocaleIntentDisambiguationModel{Enabled: frameworktypes.BoolValue(ids.Enabled)}
			if ids.CustomDisambiguationMessage != nil {
				idm.CustomDisambiguationMessage = frameworktypes.StringPointerValue(ids.CustomDisambiguationMessage)
			} else {
				idm.CustomDisambiguationMessage = frameworktypes.StringNull()
			}
			if ids.MaxDisambiguationIntents != nil {
				idm.MaxDisambiguationIntents = frameworktypes.Int64Value(int64(*ids.MaxDisambiguationIntents))
			} else {
				idm.MaxDisambiguationIntents = frameworktypes.Int64Null()
			}
			nlu.IntentDisambiguationSettings = []BotLocaleIntentDisambiguationModel{idm}
		}
		m.NluImprovement = []BotLocaleNluImprovementModel{nlu}
	}
	if v.SlotResolutionImprovement != nil {
		s := v.SlotResolutionImprovement
		m.SlotResolutionImprovement = []BotLocaleSlotResolutionModel{{
			Enabled:                   frameworktypes.BoolValue(s.Enabled),
			BedrockModelSpecification: flattenBotLocaleBedrockModelSpec(s.BedrockModelSpecification),
		}}
	}
	return []BotLocaleRuntimeSettingsModel{m}
}

func flattenBotLocaleBedrockModelSpec(v *lexv2types.BedrockModelSpecification) []BotLocaleBedrockModelSpecModel {
	if v == nil {
		return nil
	}
	m := BotLocaleBedrockModelSpecModel{ModelARN: frameworktypes.StringPointerValue(v.ModelArn)}
	if v.CustomPrompt != nil {
		m.CustomPrompt = frameworktypes.StringPointerValue(v.CustomPrompt)
	} else {
		m.CustomPrompt = frameworktypes.StringNull()
	}
	if v.TraceStatus != "" {
		m.TraceStatus = frameworktypes.StringValue(string(v.TraceStatus))
	} else {
		m.TraceStatus = frameworktypes.StringNull()
	}
	if v.Guardrail != nil {
		m.Guardrail = []BotLocaleGuardrailModel{{
			Identifier: frameworktypes.StringPointerValue(v.Guardrail.Identifier),
			Version:    frameworktypes.StringPointerValue(v.Guardrail.Version),
		}}
	}
	return []BotLocaleBedrockModelSpecModel{m}
}

func expandBotLocaleAudioFillerSettings(m []BotLocaleAudioFillerSettingsModel) *lexv2types.AudioFillerSettings {
	if len(m) == 0 {
		return nil
	}
	v := m[0]
	out := &lexv2types.AudioFillerSettings{
		Enabled:   v.Enabled.ValueBool(),
		AudioType: lexv2types.AudioFillerType(v.AudioType.ValueString()),
	}
	if !v.StartDelayInMilliseconds.IsNull() && !v.StartDelayInMilliseconds.IsUnknown() {
		v32 := int32(v.StartDelayInMilliseconds.ValueInt64())
		out.StartDelayInMilliseconds = &v32
	}
	if !v.MinimumPlayDurationInMilliseconds.IsNull() && !v.MinimumPlayDurationInMilliseconds.IsUnknown() {
		v32 := int32(v.MinimumPlayDurationInMilliseconds.ValueInt64())
		out.MinimumPlayDurationInMilliseconds = &v32
	}
	if !v.ResponseDeliveryDelayInMilliseconds.IsNull() && !v.ResponseDeliveryDelayInMilliseconds.IsUnknown() {
		v32 := int32(v.ResponseDeliveryDelayInMilliseconds.ValueInt64())
		out.ResponseDeliveryDelayInMilliseconds = &v32
	}
	return out
}

func flattenBotLocaleAudioFillerSettings(v *lexv2types.AudioFillerSettings) []BotLocaleAudioFillerSettingsModel {
	if v == nil {
		return nil
	}
	m := BotLocaleAudioFillerSettingsModel{
		Enabled:   frameworktypes.BoolValue(v.Enabled),
		AudioType: frameworktypes.StringValue(string(v.AudioType)),
	}
	if v.StartDelayInMilliseconds != nil {
		m.StartDelayInMilliseconds = frameworktypes.Int64Value(int64(*v.StartDelayInMilliseconds))
	} else {
		m.StartDelayInMilliseconds = frameworktypes.Int64Null()
	}
	if v.MinimumPlayDurationInMilliseconds != nil {
		m.MinimumPlayDurationInMilliseconds = frameworktypes.Int64Value(int64(*v.MinimumPlayDurationInMilliseconds))
	} else {
		m.MinimumPlayDurationInMilliseconds = frameworktypes.Int64Null()
	}
	if v.ResponseDeliveryDelayInMilliseconds != nil {
		m.ResponseDeliveryDelayInMilliseconds = frameworktypes.Int64Value(int64(*v.ResponseDeliveryDelayInMilliseconds))
	} else {
		m.ResponseDeliveryDelayInMilliseconds = frameworktypes.Int64Null()
	}
	return []BotLocaleAudioFillerSettingsModel{m}
}
