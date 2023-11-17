package provider

import (
	"context"
	_ "embed"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/wttech/terraform-provider-aem/internal/client"
)

// Ensure AEMProvider satisfies various provider interfaces.
var _ provider.Provider = &AEMProvider{}

//go:embed description.md
var DescriptionMD string

// AEMProvider defines the provider implementation.
type AEMProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type AEMProviderModel struct{}

func (p *AEMProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "aem"
	resp.Version = p.version
}

func (p *AEMProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DescriptionMD,
		Attributes:          map[string]schema.Attribute{},
	}
}

func (p *AEMProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AEMProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientManager := client.ClientManagerDefault
	resp.DataSourceData = clientManager
	resp.ResourceData = clientManager
}

func (p *AEMProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewInstanceResource}
}

func (p *AEMProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AEMProvider{version: version}
	}
}
