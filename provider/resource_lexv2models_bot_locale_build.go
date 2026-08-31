// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LexV2ModelsBotLocaleBuildResource{}

func NewLexV2ModelsBotLocaleBuildResource() resource.Resource {
	return &LexV2ModelsBotLocaleBuildResource{}
}

// LexV2ModelsBotLocaleBuildResource builds a bot locale as a step of its own, so the build
// can be ordered after the intents and slots exist. The bot locale resource's own
// build_on_apply runs at locale create, which is too early for anything declared against
// that locale.
type LexV2ModelsBotLocaleBuildResource struct {
	client *lexmodelsv2.Client
}

type LexV2ModelsBotLocaleBuildResourceModel struct {
	ID         frameworktypes.String `tfsdk:"id"`
	BotID      frameworktypes.String `tfsdk:"bot_id"`
	BotVersion frameworktypes.String `tfsdk:"bot_version"`
	LocaleID   frameworktypes.String `tfsdk:"locale_id"`
	Triggers   frameworktypes.Map    `tfsdk:"triggers"`
}

func (r *LexV2ModelsBotLocaleBuildResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lexv2models_bot_locale_build"
}

func (r *LexV2ModelsBotLocaleBuildResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Builds an Amazon Lex V2 bot locale and waits until it reaches `Built`.\n\n" +
			"A locale has to be rebuilt after its intents, slots and slot types change, and " +
			"`connectracer_lexv2models_bot_locale`'s `build_on_apply` cannot do it: that one runs when the " +
			"locale itself is created, before anything is declared against it, and making the locale depend " +
			"on its own slots is a cycle. Put this resource after them instead.\n\n" +
			"Put whatever should force a rebuild into `triggers`. Any change there replaces the resource, " +
			"which runs the build again. Destroying it does nothing: a built locale cannot be unbuilt.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (bot_id/bot_version/locale_id)",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bot_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bot",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bot_version": schema.StringAttribute{
				MarkdownDescription: "Version of the bot, normally `DRAFT`",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"locale_id": schema.StringAttribute{
				MarkdownDescription: "Locale to build, e.g. `de_DE`",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values that force a rebuild when they change. Reference whatever the build has to reflect, such as slot type ids or the utterances they were built from.",
				ElementType:         frameworktypes.StringType,
				Optional:            true,
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *LexV2ModelsBotLocaleBuildResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LexV2ModelsBotLocaleBuildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LexV2ModelsBotLocaleBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := buildBotLocaleAndWait(ctx, r.client,
		data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to build bot locale", err.Error())
		return
	}

	data.ID = frameworktypes.StringValue(strings.Join([]string{
		data.BotID.ValueString(), data.BotVersion.ValueString(), data.LocaleID.ValueString(),
	}, "/"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read keeps the state as it is. The locale's live status says whether it is built now, not
// whether it was built from what this resource last saw, so reconciling against it would
// only produce diffs nobody can act on. Use triggers to force a rebuild.
func (r *LexV2ModelsBotLocaleBuildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LexV2ModelsBotLocaleBuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is unreachable: every attribute forces replacement.
func (r *LexV2ModelsBotLocaleBuildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LexV2ModelsBotLocaleBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete does nothing. A build is an event, not a thing that can be removed.
func (r *LexV2ModelsBotLocaleBuildResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
