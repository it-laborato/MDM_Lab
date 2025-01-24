package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"terraform-provider-mdmlabdm/mdmlabdm_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This Terraform provider is based on the example provider from the Terraform
// documentation. It is a simple provider that interacts with the MDMlabDM API.

// Ensure MDMlabDMProvider satisfies various provider interfaces.
var _ provider.Provider = &MDMlabDMProvider{}
var _ provider.ProviderWithFunctions = &MDMlabDMProvider{}

// MDMlabDMProvider defines the provider implementation.
type MDMlabDMProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// MDMlabDMProviderModel describes the provider data model. It requires a URL
// and api key to communicate with MDMlabDM.
type MDMlabDMProviderModel struct {
	Url    types.String `tfsdk:"url"`
	ApiKey types.String `tfsdk:"apikey"`
}

func (p *MDMlabDMProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mdmlabdm"
	resp.Version = p.version
}

func (p *MDMlabDMProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "URL of your MDMlabDM server",
				Optional:            true,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: "API Key for authentication",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *MDMlabDMProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config MDMlabDMProviderModel

	tflog.Info(ctx, "Configuring MDMlabDM client")

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if config.Url.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown MDMlabDM url",
			"Url is unknown")
	}

	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikey"),
			"Unknown MDMlabDM apikey",
			"api key is unknown")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	url := os.Getenv("FLEETDM_URL")
	apikey := os.Getenv("FLEETDM_APIKEY")

	if !config.Url.IsNull() {
		url = config.Url.ValueString()
	}

	if !config.ApiKey.IsNull() {
		apikey = config.ApiKey.ValueString()
	}

	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing url",
			"Really, the url is required")
	}

	if apikey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikey"),
			"Missing apikey",
			"Really, the apikey is required")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Example client configuration for data sources and resources
	client := mdmlabdm_client.NewMDMlabDMClient(url, apikey)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *MDMlabDMProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTeamsResource,
	}
}

func (p *MDMlabDMProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *MDMlabDMProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MDMlabDMProvider{
			version: version,
		}
	}
}
