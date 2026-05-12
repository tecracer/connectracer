// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BedrockAgentCoreGatewayDataSource{}

func NewBedrockAgentCoreGatewayDataSource() datasource.DataSource {
	return &BedrockAgentCoreGatewayDataSource{}
}

// BedrockAgentCoreGatewayDataSource defines the data source implementation.
type BedrockAgentCoreGatewayDataSource struct {
	client *bedrockagentcorecontrol.Client
}

// BedrockAgentCoreGatewayDataSourceModel describes the data source data model.
type BedrockAgentCoreGatewayDataSourceModel struct {
	Name           types.String `tfsdk:"name"`
	GatewayID      types.String `tfsdk:"gateway_id"`
	GatewayArn     types.String `tfsdk:"gateway_arn"`
	GatewayUrl     types.String `tfsdk:"gateway_url"`
	Description    types.String `tfsdk:"description"`
	Status         types.String `tfsdk:"status"`
	AuthorizerType types.String `tfsdk:"authorizer_type"`
	ProtocolType   types.String `tfsdk:"protocol_type"`
}

func (d *BedrockAgentCoreGatewayDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bedrockagentcore_gateway"
}

func (d *BedrockAgentCoreGatewayDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an Amazon Bedrock AgentCore Gateway by name.\n\nThis data source can be used to retrieve gateway details (ID, URL, ARN) without creating a circular dependency when the gateway needs to reference itself.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the gateway to look up.",
				Required:            true,
			},
			"gateway_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the gateway.",
				Computed:            true,
			},
			"gateway_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the gateway.",
				Computed:            true,
			},
			"gateway_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the gateway.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the gateway.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the gateway.",
				Computed:            true,
			},
			"authorizer_type": schema.StringAttribute{
				MarkdownDescription: "The authorizer type of the gateway.",
				Computed:            true,
			},
			"protocol_type": schema.StringAttribute{
				MarkdownDescription: "The protocol type of the gateway.",
				Computed:            true,
			},
		},
	}
}

func (d *BedrockAgentCoreGatewayDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = clients.BedrockAgentCoreControl
}

func (d *BedrockAgentCoreGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BedrockAgentCoreGatewayDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	tflog.Debug(ctx, "Looking up Bedrock AgentCore Gateway by name", map[string]interface{}{
		"name": name,
	})

	// List gateways and find the one matching the given name
	var foundGatewayId *string

	input := &bedrockagentcorecontrol.ListGatewaysInput{
		MaxResults: aws.Int32(100),
	}

	for {
		output, err := d.client.ListGateways(ctx, input)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing Bedrock AgentCore Gateways",
				fmt.Sprintf("Unable to list gateways, got error: %s", err),
			)
			return
		}

		for _, gateway := range output.Items {
			if gateway.Name != nil && *gateway.Name == name {
				foundGatewayId = gateway.GatewayId
				break
			}
		}

		if foundGatewayId != nil {
			break
		}

		if output.NextToken == nil {
			break
		}

		input.NextToken = output.NextToken
	}

	if foundGatewayId == nil {
		resp.Diagnostics.AddError(
			"Gateway Not Found",
			fmt.Sprintf("No gateway found with name %q", name),
		)
		return
	}

	// Get the full gateway details
	getOutput, err := d.client.GetGateway(ctx, &bedrockagentcorecontrol.GetGatewayInput{
		GatewayIdentifier: foundGatewayId,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Bedrock AgentCore Gateway",
			fmt.Sprintf("Unable to read gateway %s, got error: %s", *foundGatewayId, err),
		)
		return
	}

	// Populate the model from GetGatewayOutput
	data.GatewayID = types.StringValue(*getOutput.GatewayId)
	data.GatewayArn = types.StringValue(*getOutput.GatewayArn)

	if getOutput.GatewayUrl != nil {
		data.GatewayUrl = types.StringValue(*getOutput.GatewayUrl)
	} else {
		data.GatewayUrl = types.StringNull()
	}

	if getOutput.Description != nil {
		data.Description = types.StringValue(*getOutput.Description)
	} else {
		data.Description = types.StringNull()
	}

	data.Status = types.StringValue(string(getOutput.Status))
	data.AuthorizerType = types.StringValue(string(getOutput.AuthorizerType))

	protocolType := string(getOutput.ProtocolType)
	if protocolType != "" {
		data.ProtocolType = types.StringValue(protocolType)
	} else {
		data.ProtocolType = types.StringNull()
	}

	tflog.Trace(ctx, "Read Bedrock AgentCore Gateway data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
