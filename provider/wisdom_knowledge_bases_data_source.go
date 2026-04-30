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
var _ datasource.DataSource = &WisdomKnowledgeBasesDataSource{}

func NewWisdomKnowledgeBasesDataSource() datasource.DataSource {
	return &WisdomKnowledgeBasesDataSource{}
}

// WisdomKnowledgeBasesDataSource defines the data source implementation.
type WisdomKnowledgeBasesDataSource struct {
	client *wisdom.Client
}

// WisdomKnowledgeBasesDataSourceModel describes the data source data model.
type WisdomKnowledgeBasesDataSourceModel struct {
	KnowledgeBases []KnowledgeBaseModel `tfsdk:"knowledge_bases"`
	ID             types.String         `tfsdk:"id"`
}

// KnowledgeBaseModel describes a single knowledge base.
type KnowledgeBaseModel struct {
	KnowledgeBaseArn  types.String `tfsdk:"knowledge_base_arn"`
	KnowledgeBaseID   types.String `tfsdk:"knowledge_base_id"`
	KnowledgeBaseType types.String `tfsdk:"knowledge_base_type"`
	Name              types.String `tfsdk:"name"`
	Status            types.String `tfsdk:"status"`
	Tags              types.Map    `tfsdk:"tags"`
}

func (d *WisdomKnowledgeBasesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wisdom_knowledge_bases"
}

func (d *WisdomKnowledgeBasesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a list of AWS Wisdom knowledge bases",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier",
				Computed:            true,
			},
			"knowledge_bases": schema.ListNestedAttribute{
				MarkdownDescription: "List of knowledge bases",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"knowledge_base_arn": schema.StringAttribute{
							MarkdownDescription: "The ARN of the knowledge base",
							Computed:            true,
						},
						"knowledge_base_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the knowledge base",
							Computed:            true,
						},
						"knowledge_base_type": schema.StringAttribute{
							MarkdownDescription: "The type of the knowledge base",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the knowledge base",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The status of the knowledge base",
							Computed:            true,
						},
						"tags": schema.MapAttribute{
							MarkdownDescription: "Tags associated with the knowledge base",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *WisdomKnowledgeBasesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*wisdom.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *wisdom.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *WisdomKnowledgeBasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WisdomKnowledgeBasesDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call AWS Wisdom API to list knowledge bases
	input := &wisdom.ListKnowledgeBasesInput{}
	
	tflog.Debug(ctx, "Calling AWS Wisdom ListKnowledgeBases API")
	
	result, err := d.client.ListKnowledgeBases(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to list knowledge bases, got error: %s", err),
		)
		return
	}

	// Map response to our model
	knowledgeBases := make([]KnowledgeBaseModel, 0, len(result.KnowledgeBaseSummaries))
	
	for _, kb := range result.KnowledgeBaseSummaries {
		var tags types.Map
		
		if kb.Tags != nil && len(kb.Tags) > 0 {
			tagMap := make(map[string]string)
			for k, v := range kb.Tags {
				tagMap[k] = v
			}
			var diags = resp.Diagnostics
			tags, diags = types.MapValueFrom(ctx, types.StringType, tagMap)
			resp.Diagnostics.Append(diags...)
		} else {
			tags = types.MapNull(types.StringType)
		}

		knowledgeBases = append(knowledgeBases, KnowledgeBaseModel{
			KnowledgeBaseArn:  types.StringPointerValue(kb.KnowledgeBaseArn),
			KnowledgeBaseID:   types.StringPointerValue(kb.KnowledgeBaseId),
			KnowledgeBaseType: types.StringValue(string(kb.KnowledgeBaseType)),
			Name:              types.StringPointerValue(kb.Name),
			Status:            types.StringValue(string(kb.Status)),
			Tags:              tags,
		})
	}

	data.KnowledgeBases = knowledgeBases
	data.ID = types.StringValue("wisdom-knowledge-bases")

	tflog.Trace(ctx, "read wisdom knowledge bases data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
