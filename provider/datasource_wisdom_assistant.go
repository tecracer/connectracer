// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/wisdom"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &WisdomAssistantsDataSource{}

func NewWisdomAssistantsDataSource() datasource.DataSource {
	return &WisdomAssistantsDataSource{}
}

// WisdomAssistantsDataSource defines the data source implementation.
type WisdomAssistantsDataSource struct {
	client *wisdom.Client
}

// WisdomAssistantsDataSourceModel describes the data source data model.
type WisdomAssistantsDataSourceModel struct {
	Assistants []AssistantModel `tfsdk:"assistants"`
	ID         types.String     `tfsdk:"id"`
}

// AssistantModel describes a single assistant.
type AssistantModel struct {
	AssistantArn              types.String                     `tfsdk:"assistant_arn"`
	AssistantID               types.String                     `tfsdk:"assistant_id"`
	IntegrationConfiguration  *IntegrationConfigurationModel   `tfsdk:"integration_configuration"`
	Name                      types.String                     `tfsdk:"name"`
	Status                    types.String                     `tfsdk:"status"`
	Tags                      types.Map                        `tfsdk:"tags"`
	Type                      types.String                     `tfsdk:"type"`
}

// IntegrationConfigurationModel describes the integration configuration.
type IntegrationConfigurationModel struct {
	TopicIntegrationArn types.String `tfsdk:"topic_integration_arn"`
}

func (d *WisdomAssistantsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wisdom_assistants"
}

func (d *WisdomAssistantsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a list of AWS Wisdom assistants",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier",
				Computed:            true,
			},
			"assistants": schema.ListNestedAttribute{
				MarkdownDescription: "List of assistants",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"assistant_arn": schema.StringAttribute{
							MarkdownDescription: "The ARN of the assistant",
							Computed:            true,
						},
						"assistant_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the assistant",
							Computed:            true,
						},
						"integration_configuration": schema.SingleNestedAttribute{
							MarkdownDescription: "The integration configuration for the assistant",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"topic_integration_arn": schema.StringAttribute{
									MarkdownDescription: "The ARN of the topic integration",
									Computed:            true,
								},
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the assistant",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The status of the assistant",
							Computed:            true,
						},
						"tags": schema.MapAttribute{
							MarkdownDescription: "Tags associated with the assistant",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the assistant",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *WisdomAssistantsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = clients.Wisdom
}

func (d *WisdomAssistantsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WisdomAssistantsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call AWS Wisdom API to list assistants
	input := &wisdom.ListAssistantsInput{}
	
	tflog.Debug(ctx, "Calling AWS Wisdom ListAssistants API")
	
	result, err := d.client.ListAssistants(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to list assistants, got error: %s", err),
		)
		return
	}

	// Map response to our model
	assistants := make([]AssistantModel, 0, len(result.AssistantSummaries))
	
	for _, assistant := range result.AssistantSummaries {
		var tags types.Map
		
		if len(assistant.Tags) > 0 {
			tagMap := make(map[string]string)
			for k, v := range assistant.Tags {
				tagMap[k] = v
			}
			var diags = resp.Diagnostics
			tags, diags = types.MapValueFrom(ctx, types.StringType, tagMap)
			resp.Diagnostics.Append(diags...)
		} else {
			tags = types.MapNull(types.StringType)
		}

		// Handle integration configuration
		var integrationConfig *IntegrationConfigurationModel
		if assistant.IntegrationConfiguration != nil && assistant.IntegrationConfiguration.TopicIntegrationArn != nil {
			integrationConfig = &IntegrationConfigurationModel{
				TopicIntegrationArn: types.StringPointerValue(assistant.IntegrationConfiguration.TopicIntegrationArn),
			}
		}

		assistants = append(assistants, AssistantModel{
			AssistantArn:             types.StringPointerValue(assistant.AssistantArn),
			AssistantID:              types.StringPointerValue(assistant.AssistantId),
			IntegrationConfiguration: integrationConfig,
			Name:                     types.StringPointerValue(assistant.Name),
			Status:                   types.StringValue(string(assistant.Status)),
			Tags:                     tags,
			Type:                     types.StringValue(string(assistant.Type)),
		})
	}

	data.Assistants = assistants
	data.ID = types.StringValue("wisdom-assistants")

	tflog.Trace(ctx, "read wisdom assistants data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
