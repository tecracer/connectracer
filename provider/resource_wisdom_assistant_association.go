// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wisdom"
	"github.com/aws/aws-sdk-go-v2/service/wisdom/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WisdomAssistantAssociationResource{}
var _ resource.ResourceWithImportState = &WisdomAssistantAssociationResource{}

func NewWisdomAssistantAssociationResource() resource.Resource {
	return &WisdomAssistantAssociationResource{}
}

// WisdomAssistantAssociationResource defines the resource implementation.
type WisdomAssistantAssociationResource struct {
	client *wisdom.Client
}

// WisdomAssistantAssociationResourceModel describes the resource data model.
type WisdomAssistantAssociationResourceModel struct {
	ID                       frameworktypes.String                    `tfsdk:"id"`
	AssistantAssociationArn  frameworktypes.String                    `tfsdk:"assistant_association_arn"`
	AssistantID              frameworktypes.String                    `tfsdk:"assistant_id"`
	AssociationType          frameworktypes.String                    `tfsdk:"association_type"`
	AssociationData          *AssociationDataModel                    `tfsdk:"association_data"`
	Tags                     frameworktypes.Map                       `tfsdk:"tags"`
}

// AssociationDataModel describes the association data nested block.
type AssociationDataModel struct {
	KnowledgeBaseID frameworktypes.String `tfsdk:"knowledge_base_id"`
}

func (r *WisdomAssistantAssociationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wisdom_assistant_association"
}

func (r *WisdomAssistantAssociationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AWS Wisdom Assistant Association resource. Automatically sets the required `AmazonConnectEnabled = \"True\"` tag.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the assistant association",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"assistant_association_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the assistant association",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"assistant_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Wisdom assistant",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"association_type": schema.StringAttribute{
				MarkdownDescription: "The type of association. Valid values: KNOWLEDGE_BASE",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"association_data": schema.SingleNestedAttribute{
				MarkdownDescription: "The data for the association",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"knowledge_base_id": schema.StringAttribute{
						MarkdownDescription: "The ID of the knowledge base",
						Required:            true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to apply to the assistant association. The `AmazonConnectEnabled = \"True\"` tag is automatically added if not present.",
				Optional:            true,
				Computed:            true,
				ElementType:         frameworktypes.StringType,
				PlanModifiers: []planmodifier.Map{
					&ensureConnectEnabledTagModifier{},
				},
			},
		},
	}
}

func (r *WisdomAssistantAssociationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.Wisdom
}

func (r *WisdomAssistantAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WisdomAssistantAssociationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure required tags
	tags, err := ensureRequiredTags(ctx, data.Tags)
	if err != nil {
		resp.Diagnostics.AddError("Tag Error", fmt.Sprintf("Unable to process tags: %s", err))
		return
	}

	// Build CreateAssistantAssociation input
	input := &wisdom.CreateAssistantAssociationInput{
		AssistantId:     aws.String(data.AssistantID.ValueString()),
		AssociationType: types.AssociationType(data.AssociationType.ValueString()),
		Tags:            tags,
	}

	// Add association data
	if data.AssociationData != nil {
		input.Association = &types.AssistantAssociationInputDataMemberKnowledgeBaseId{
			Value: data.AssociationData.KnowledgeBaseID.ValueString(),
		}
	}

	tflog.Debug(ctx, "Creating AWS Wisdom Assistant Association", map[string]interface{}{
		"assistant_id":     data.AssistantID.ValueString(),
		"association_type": data.AssociationType.ValueString(),
	})

	// Create the assistant association
	output, err := r.client.CreateAssistantAssociation(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Wisdom Assistant Association",
			fmt.Sprintf("Unable to create assistant association, got error: %s", err),
		)
		return
	}

	// Map response to model
	if output.AssistantAssociation != nil {
		data.ID = frameworktypes.StringPointerValue(output.AssistantAssociation.AssistantAssociationId)
		data.AssistantAssociationArn = frameworktypes.StringPointerValue(output.AssistantAssociation.AssistantAssociationArn)
		data.AssistantID = frameworktypes.StringPointerValue(output.AssistantAssociation.AssistantId)
		data.AssociationType = frameworktypes.StringValue(string(output.AssistantAssociation.AssociationType))

		// Map association data
		if output.AssistantAssociation.AssociationData != nil {
			switch v := output.AssistantAssociation.AssociationData.(type) {
			case *types.AssistantAssociationOutputDataMemberKnowledgeBaseAssociation:
				data.AssociationData = &AssociationDataModel{
					KnowledgeBaseID: frameworktypes.StringPointerValue(v.Value.KnowledgeBaseId),
				}
			}
		}

		// Store tags back in state
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	} else {
		// If API didn't return tags, still store what we sent
		tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, tags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Tags = tagsMap
		}
	}

	tflog.Trace(ctx, "Created Wisdom Assistant Association resource", map[string]interface{}{
		"association_id": data.ID.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WisdomAssistantAssociationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS Wisdom Assistant Association", map[string]interface{}{
		"assistant_id":   data.AssistantID.ValueString(),
		"association_id": data.ID.ValueString(),
	})

	// Get the assistant association
	input := &wisdom.GetAssistantAssociationInput{
		AssistantId:            aws.String(data.AssistantID.ValueString()),
		AssistantAssociationId: aws.String(data.ID.ValueString()),
	}

	output, err := r.client.GetAssistantAssociation(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Wisdom Assistant Association",
			fmt.Sprintf("Unable to read assistant association %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Map response to model
	if output.AssistantAssociation != nil {
		data.AssistantAssociationArn = frameworktypes.StringPointerValue(output.AssistantAssociation.AssistantAssociationArn)
		data.AssistantID = frameworktypes.StringPointerValue(output.AssistantAssociation.AssistantId)
		data.AssociationType = frameworktypes.StringValue(string(output.AssistantAssociation.AssociationType))

		// Map association data
		if output.AssistantAssociation.AssociationData != nil {
			switch v := output.AssistantAssociation.AssociationData.(type) {
			case *types.AssistantAssociationOutputDataMemberKnowledgeBaseAssociation:
				data.AssociationData = &AssociationDataModel{
					KnowledgeBaseID: frameworktypes.StringPointerValue(v.Value.KnowledgeBaseId),
				}
			}
		} else {
			data.AssociationData = nil
		}

		// Map tags
		if len(output.AssistantAssociation.Tags) > 0 {
			tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.AssistantAssociation.Tags)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				data.Tags = tagsMap
			}
		} else {
			data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WisdomAssistantAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WisdomAssistantAssociationResourceModel
	var state WisdomAssistantAssociationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating AWS Wisdom Assistant Association", map[string]interface{}{
		"association_id": data.ID.ValueString(),
	})

	// Note: AWS Wisdom doesn't have an UpdateAssistantAssociation API
	// The only field that can be updated is tags using TagResource/UntagResource

	// Handle tag updates
	if !data.Tags.Equal(state.Tags) {
		associationArn := state.AssistantAssociationArn.ValueString()

		// Ensure required tags for new tags
		newTags, err := ensureRequiredTags(ctx, data.Tags)
		if err != nil {
			resp.Diagnostics.AddError("Tag Error", fmt.Sprintf("Unable to process tags: %s", err))
			return
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
			untagInput := &wisdom.UntagResourceInput{
				ResourceArn: aws.String(associationArn),
				TagKeys:     tagsToRemove,
			}
			_, err := r.client.UntagResource(ctx, untagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing tags from Wisdom Assistant Association",
					fmt.Sprintf("Unable to remove tags, got error: %s", err),
				)
				return
			}
		}

		// Add new/updated tags
		if len(newTags) > 0 {
			tagInput := &wisdom.TagResourceInput{
				ResourceArn: aws.String(associationArn),
				Tags:        newTags,
			}
			_, err := r.client.TagResource(ctx, tagInput)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error tagging Wisdom Assistant Association",
					fmt.Sprintf("Unable to add tags, got error: %s", err),
				)
				return
			}
		}
	}

	// Read the resource again to get the current state
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: req.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics

	tflog.Trace(ctx, "Updated Wisdom Assistant Association resource")
}

func (r *WisdomAssistantAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WisdomAssistantAssociationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting AWS Wisdom Assistant Association", map[string]interface{}{
		"assistant_id":   data.AssistantID.ValueString(),
		"association_id": data.ID.ValueString(),
	})

	// Delete the assistant association
	input := &wisdom.DeleteAssistantAssociationInput{
		AssistantId:            aws.String(data.AssistantID.ValueString()),
		AssistantAssociationId: aws.String(data.ID.ValueString()),
	}

	_, err := r.client.DeleteAssistantAssociation(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Wisdom Assistant Association",
			fmt.Sprintf("Unable to delete assistant association %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "Deleted Wisdom Assistant Association resource")
}

func (r *WisdomAssistantAssociationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by composite ID: assistant_id/association_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: assistant_id/association_id, got: %s", req.ID),
		)
		return
	}

	assistantID := parts[0]
	associationID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("assistant_id"), assistantID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), associationID)...)
}
