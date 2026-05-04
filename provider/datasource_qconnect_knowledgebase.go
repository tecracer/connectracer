// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &QConnectKnowledgeBaseDataSource{}

func NewQConnectKnowledgeBaseDataSource() datasource.DataSource {
	return &QConnectKnowledgeBaseDataSource{}
}

// QConnectKnowledgeBaseDataSource defines the data source implementation.
type QConnectKnowledgeBaseDataSource struct {
	client *qconnect.Client
}

// QConnectKnowledgeBaseDataSourceModel describes the data source data model.
type QConnectKnowledgeBaseDataSourceModel struct {
	ID                                frameworktypes.String                   `tfsdk:"id"`
	KnowledgeBaseID                   frameworktypes.String                   `tfsdk:"knowledge_base_id"`
	KnowledgeBaseArn                  frameworktypes.String                   `tfsdk:"knowledge_base_arn"`
	Name                              frameworktypes.String                   `tfsdk:"name"`
	KnowledgeBaseType                 frameworktypes.String                   `tfsdk:"knowledge_base_type"`
	Description                       frameworktypes.String                   `tfsdk:"description"`
	Tags                              frameworktypes.Map                      `tfsdk:"tags"`
	RenderingConfiguration            *RenderingConfigurationModel            `tfsdk:"rendering_configuration"`
	ServerSideEncryptionConfiguration *ServerSideEncryptionConfigurationModel `tfsdk:"server_side_encryption_configuration"`
	SourceConfiguration               *SourceConfigurationModel               `tfsdk:"source_configuration"`
	Status                            frameworktypes.String                   `tfsdk:"status"`
}

func (d *QConnectKnowledgeBaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_qconnect_knowledgebase"
}

func (d *QConnectKnowledgeBaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for looking up an AWS Q Connect Knowledge Base by ID",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Terraform identifier (same as knowledge_base_id)",
				Computed:            true,
			},
			"knowledge_base_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the knowledge base to look up",
				Required:            true,
			},
			"knowledge_base_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the knowledge base",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the knowledge base",
				Computed:            true,
			},
			"knowledge_base_type": schema.StringAttribute{
				MarkdownDescription: "The type of knowledge base. Values: EXTERNAL, CUSTOM",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the knowledge base",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags applied to the knowledge base",
				Computed:            true,
				ElementType:         frameworktypes.StringType,
			},
			"rendering_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "Information about how to render the content",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"template_uri": schema.StringAttribute{
						MarkdownDescription: "A URI template containing exactly one variable in ${variableName} format",
						Computed:            true,
					},
				},
			},
			"server_side_encryption_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The KMS key used for encryption",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"kms_key_id": schema.StringAttribute{
						MarkdownDescription: "The KMS key ID or ARN",
						Computed:            true,
					},
				},
			},
			"source_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The source of the knowledge base content. Only set for EXTERNAL knowledge bases",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"app_integration_arn": schema.StringAttribute{
						MarkdownDescription: "The Amazon Resource Name (ARN) of the AppIntegrations DataIntegration",
						Computed:            true,
					},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the knowledge base",
				Computed:            true,
			},
		},
	}
}

func (d *QConnectKnowledgeBaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = clients.QConnect
}

func (d *QConnectKnowledgeBaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data QConnectKnowledgeBaseDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading AWS Q Connect Knowledge Base", map[string]interface{}{
		"knowledge_base_id": data.KnowledgeBaseID.ValueString(),
	})

	// Get the knowledge base
	input := &qconnect.GetKnowledgeBaseInput{
		KnowledgeBaseId: aws.String(data.KnowledgeBaseID.ValueString()),
	}

	output, err := d.client.GetKnowledgeBase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Q Connect Knowledge Base",
			fmt.Sprintf("Unable to read knowledge base %s, got error: %s", data.KnowledgeBaseID.ValueString(), err),
		)
		return
	}

	// Map response to model
	if output.KnowledgeBase != nil {
		kb := output.KnowledgeBase
		
		data.ID = frameworktypes.StringPointerValue(kb.KnowledgeBaseId)
		data.KnowledgeBaseID = frameworktypes.StringPointerValue(kb.KnowledgeBaseId)
		data.KnowledgeBaseArn = frameworktypes.StringPointerValue(kb.KnowledgeBaseArn)
		data.Name = frameworktypes.StringPointerValue(kb.Name)
		data.KnowledgeBaseType = frameworktypes.StringValue(string(kb.KnowledgeBaseType))
		data.Description = frameworktypes.StringPointerValue(kb.Description)
		data.Status = frameworktypes.StringValue(string(kb.Status))

		// Map rendering configuration
		if kb.RenderingConfiguration != nil && kb.RenderingConfiguration.TemplateUri != nil {
			data.RenderingConfiguration = &RenderingConfigurationModel{
				TemplateUri: frameworktypes.StringPointerValue(kb.RenderingConfiguration.TemplateUri),
			}
		}

		// Map server-side encryption configuration
		if kb.ServerSideEncryptionConfiguration != nil && kb.ServerSideEncryptionConfiguration.KmsKeyId != nil {
			data.ServerSideEncryptionConfiguration = &ServerSideEncryptionConfigurationModel{
				KmsKeyId: frameworktypes.StringPointerValue(kb.ServerSideEncryptionConfiguration.KmsKeyId),
			}
		}

		// Map source configuration (for EXTERNAL knowledge bases)
		if kb.SourceConfiguration != nil {
			switch v := kb.SourceConfiguration.(type) {
			case *types.SourceConfigurationMemberAppIntegrations:
				if v.Value.AppIntegrationArn != nil {
					data.SourceConfiguration = &SourceConfigurationModel{
						AppIntegrationArn: frameworktypes.StringPointerValue(v.Value.AppIntegrationArn),
					}
				}
			}
		}

		// Map tags
		if len(kb.Tags) > 0 {
			tagsMap, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, kb.Tags)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				data.Tags = tagsMap
			}
		} else {
			data.Tags = frameworktypes.MapNull(frameworktypes.StringType)
		}
	}

	tflog.Trace(ctx, "Read Q Connect Knowledge Base data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
