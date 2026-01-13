// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/supabase/cli/pkg/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FunctionBodyDataSource{}

func NewFunctionBodyDataSource() datasource.DataSource {
	return &FunctionBodyDataSource{}
}

// FunctionBodyDataSource defines the data source implementation.
type FunctionBodyDataSource struct {
	client *api.ClientWithResponses
}

// FunctionBodyDataSourceModel describes the data source data model.
type FunctionBodyDataSourceModel struct {
	ProjectRef types.String `tfsdk:"project_ref"`
	Slug       types.String `tfsdk:"slug"`
	Body       types.String `tfsdk:"body"`
}

func (d *FunctionBodyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function_body"
}

func (d *FunctionBodyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve the body/source code of a deployed Edge Function",

		Attributes: map[string]schema.Attribute{
			"project_ref": schema.StringAttribute{
				MarkdownDescription: "Project reference ID",
				Required:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Function slug",
				Required:            true,
			},
			"body": schema.StringAttribute{
				MarkdownDescription: "Function body content",
				Computed:            true,
			},
		},
	}
}

func (d *FunctionBodyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *FunctionBodyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FunctionBodyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.V1GetAFunctionBodyWithResponse(
		ctx,
		data.ProjectRef.ValueString(),
		data.Slug.ValueString(),
	)
	if err != nil {
		msg := fmt.Sprintf("Unable to read function body, got error: %s", err)
		resp.Diagnostics.AddError("Client Error", msg)
		return
	}

	if httpResp.StatusCode() != 200 {
		msg := fmt.Sprintf("Unable to read function body, got status %d: %s", httpResp.StatusCode(), httpResp.Body)
		resp.Diagnostics.AddError("Client Error", msg)
		return
	}

	// The body is returned as raw bytes in httpResp.Body
	data.Body = types.StringValue(string(httpResp.Body))

	tflog.Trace(ctx, "read function body data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
