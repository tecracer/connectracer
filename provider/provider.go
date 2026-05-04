// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appintegrations"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
	"github.com/aws/aws-sdk-go-v2/service/wisdom"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hashicupsProvider is the provider implementation.
type connectracerProvider struct {
    // version is set to the provider version on release, "dev" when the
    // provider is built and ran locally, and "test" when running acceptance
    // testing.
    version string
}




// Ensure the implementation satisfies the expected interfaces.
var (
    _ provider.Provider = &connectracerProvider{}
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &connectracerProvider{}
var _ provider.ProviderWithFunctions = &connectracerProvider{}
var _ provider.ProviderWithEphemeralResources = &connectracerProvider{}
var _ provider.ProviderWithActions = &connectracerProvider{}

// ScaffoldingProvider defines the provider implementation.
type ScaffoldingProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ConnectracerProviderModel describes the provider data model.
type ConnectracerProviderModel struct {
	Region types.String `tfsdk:"region"`
}

// ProviderClients holds the AWS service clients.
type ProviderClients struct {
	Wisdom          *wisdom.Client
	QConnect        *qconnect.Client
	AppIntegrations *appintegrations.Client
	Connect         *connect.Client
}

func (p *connectracerProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "connectracer"
    resp.Version = p.version
}

func (p *connectracerProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for AWS Connect and related services",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region (optional, uses AWS SDK default resolution if not specified)",
				Optional:            true,
			},
		},
	}
}

func (p *connectracerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ConnectracerProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create AWS config",
			err.Error(),
		)
		return
	}

	// Override region if specified in provider configuration
	if !data.Region.IsNull() {
		cfg.Region = data.Region.ValueString()
	}

	// Create AWS Wisdom client
	wisdomClient := wisdom.NewFromConfig(cfg)

	// Create AWS Q Connect client
	qconnectClient := qconnect.NewFromConfig(cfg)

	// Create AWS AppIntegrations client
	appIntegrationsClient := appintegrations.NewFromConfig(cfg)

	// Create AWS Connect client
	connectClient := connect.NewFromConfig(cfg)

	// Store all clients in a struct for resources to access
	clients := &ProviderClients{
		Wisdom:          wisdomClient,
		QConnect:        qconnectClient,
		AppIntegrations: appIntegrationsClient,
		Connect:         connectClient,
	}

	// Make the clients available to data sources and resources
	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *connectracerProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewExampleResource,
		NewWisdomAssistantResource,
		NewQConnectKnowledgeBaseResource,
		NewWisdomAssistantAssociationResource,
		NewAppIntegrationsDataIntegrationResource,
		NewConnectIntegrationAssociationResource,
	}
}

func (p *connectracerProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewExampleEphemeralResource,
	}
}

func (p *connectracerProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
		NewWisdomKnowledgeBasesDataSource,
		NewWisdomAssistantsDataSource,
		NewQConnectKnowledgeBaseDataSource,
	}
}

func (p *connectracerProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func (p *connectracerProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{
		NewExampleAction,
	}
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
    return func() provider.Provider {
        return &connectracerProvider{
            version: version,
        }
    }
}
