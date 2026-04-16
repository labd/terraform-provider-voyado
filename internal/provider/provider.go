package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/labd/terraform-provider-voyado/internal/engage"
)

var _ provider.Provider = (*VoyadoProvider)(nil)

type VoyadoProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VoyadoProvider{version: version}
	}
}

func (p *VoyadoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "voyado"
	resp.Version = p.version
}

func (p *VoyadoProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for the Engage HTTP API (for example `https://{tenant}.voyado.com` or `https://{tenant}.staging.voyado.com`). Paths such as `api/v3/...` are appended to this URL.",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Engage API key (`apikey` header).",
				Required:            true,
				Sensitive:           true,
			},
			"user_agent": schema.StringAttribute{
				MarkdownDescription: "Optional `User-Agent` header; Voyado recommends setting this for supportability.",
				Optional:            true,
			},
		},
	}
}

type providerModel struct {
	APIURL    types.String `tfsdk:"api_url"`
	APIKey    types.String `tfsdk:"api_key"`
	UserAgent types.String `tfsdk:"user_agent"`
}

func (p *VoyadoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := engage.NewClient(config.APIURL.ValueString(), config.APIKey.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	if !config.UserAgent.IsNull() && config.UserAgent.ValueString() != "" {
		client.WithUserAgent(config.UserAgent.ValueString())
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *VoyadoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInteractionSchemaResource,
	}
}

func (p *VoyadoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
