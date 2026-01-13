// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/supabase/cli/pkg/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FunctionResource{}

func NewFunctionResource() resource.Resource {
	return &FunctionResource{}
}

// FunctionResource defines the resource implementation.
type FunctionResource struct {
	client *api.ClientWithResponses
}

// FunctionResourceModel describes the resource data model.
type FunctionResourceModel struct {
	ProjectRef     types.String `tfsdk:"project_ref"`
	Slug           types.String `tfsdk:"slug"`
	EntrypointPath types.String `tfsdk:"entrypoint_path"`
	SourceDir      types.String `tfsdk:"source_dir"`
	Name           types.String `tfsdk:"name"`
	VerifyJwt      types.Bool   `tfsdk:"verify_jwt"`
	ImportMapPath  types.String `tfsdk:"import_map_path"`
	Id             types.String `tfsdk:"id"`
	Status         types.String `tfsdk:"status"`
	Version        types.Int64  `tfsdk:"version"`
	CreatedAt      types.Int64  `tfsdk:"created_at"`
	UpdatedAt      types.Int64  `tfsdk:"updated_at"`
}

// FunctionDeployMetadata matches the API's expected metadata format.
type FunctionDeployMetadata struct {
	EntrypointPath string   `json:"entrypoint_path"`
	ImportMapPath  *string  `json:"import_map_path,omitempty"`
	Name           *string  `json:"name,omitempty"`
	StaticPatterns []string `json:"static_patterns,omitempty"`
	VerifyJwt      *bool    `json:"verify_jwt,omitempty"`
}

func (r *FunctionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (r *FunctionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Edge Function resource",

		Attributes: map[string]schema.Attribute{
			// Required inputs
			"project_ref": schema.StringAttribute{
				MarkdownDescription: "Project reference ID",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Function slug (must start with a letter and contain only letters, numbers, underscores, and hyphens)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`),
						"must start with a letter and contain only letters, numbers, underscores, and hyphens",
					),
				},
			},
			"entrypoint_path": schema.StringAttribute{
				MarkdownDescription: "Path to the entrypoint file relative to source_dir (e.g., index.ts)",
				Required:            true,
			},
			"source_dir": schema.StringAttribute{
				MarkdownDescription: "Directory containing function source files",
				Required:            true,
			},

			// Optional inputs
			"name": schema.StringAttribute{
				MarkdownDescription: "Function name (defaults to slug if not specified)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"verify_jwt": schema.BoolAttribute{
				MarkdownDescription: "Whether to verify JWT tokens (default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"import_map_path": schema.StringAttribute{
				MarkdownDescription: "Path to the import map file relative to source_dir",
				Optional:            true,
			},

			// Computed outputs
			"id": schema.StringAttribute{
				MarkdownDescription: "Function identifier (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Function status (ACTIVE, REMOVED, THROTTLED)",
				Computed:            true,
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Function deployment version",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				MarkdownDescription: "Unix timestamp when function was created",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				MarkdownDescription: "Unix timestamp when function was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *FunctionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *FunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FunctionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(deployFunction(ctx, &data, r.client)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created edge function resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FunctionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(readFunction(ctx, &data, r.client)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read function")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FunctionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(deployFunction(ctx, &data, r.client)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated function")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FunctionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(deleteFunction(ctx, &data, r.client)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "deleted function")
}

func deployFunction(ctx context.Context, data *FunctionResourceModel, client *api.ClientWithResponses) diag.Diagnostics {
	// Build multipart form body
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Build metadata struct
	metadata := FunctionDeployMetadata{
		EntrypointPath: filepath.ToSlash(data.EntrypointPath.ValueString()),
	}
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		metadata.Name = &name
	}
	if !data.VerifyJwt.IsNull() && !data.VerifyJwt.IsUnknown() {
		verifyJwt := data.VerifyJwt.ValueBool()
		metadata.VerifyJwt = &verifyJwt
	}
	if !data.ImportMapPath.IsNull() && !data.ImportMapPath.IsUnknown() {
		importMapPath := filepath.ToSlash(data.ImportMapPath.ValueString())
		metadata.ImportMapPath = &importMapPath
	}

	// Write metadata as a single JSON-encoded form field
	metadataField, err := writer.CreateFormField("metadata")
	if err != nil {
		msg := fmt.Sprintf("Unable to create metadata field, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}
	if err := json.NewEncoder(metadataField).Encode(metadata); err != nil {
		msg := fmt.Sprintf("Unable to encode metadata, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	// Add files from source_dir
	sourceDir := data.SourceDir.ValueString()
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get relative path for the form field name
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Convert to forward slashes for cross-platform compatibility
		relPath = filepath.ToSlash(relPath)

		// Create form file with relative path as filename
		part, err := writer.CreateFormFile("file", relPath)
		if err != nil {
			return fmt.Errorf("failed to create form file: %w", err)
		}

		// Read and write file content
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})
	if err != nil {
		msg := fmt.Sprintf("Unable to read source directory, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	if err := writer.Close(); err != nil {
		msg := fmt.Sprintf("Unable to close multipart writer, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	// Call API
	params := &api.V1DeployAFunctionParams{
		Slug: data.Slug.ValueStringPointer(),
	}

	httpResp, err := client.V1DeployAFunctionWithBodyWithResponse(
		ctx,
		data.ProjectRef.ValueString(),
		params,
		writer.FormDataContentType(),
		&buf,
	)
	if err != nil {
		msg := fmt.Sprintf("Unable to deploy function, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	if httpResp.JSON201 == nil {
		msg := fmt.Sprintf("Unable to deploy function, got status %d: %s", httpResp.StatusCode(), httpResp.Body)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	// Map response to data model
	result := httpResp.JSON201
	data.Id = types.StringValue(result.Id)
	data.Slug = types.StringValue(result.Slug)
	data.Name = types.StringValue(result.Name)
	data.Status = types.StringValue(string(result.Status))
	data.Version = types.Int64Value(int64(result.Version))

	if result.CreatedAt != nil {
		data.CreatedAt = types.Int64Value(*result.CreatedAt)
	}
	if result.UpdatedAt != nil {
		data.UpdatedAt = types.Int64Value(*result.UpdatedAt)
	}

	return nil
}

func readFunction(ctx context.Context, data *FunctionResourceModel, client *api.ClientWithResponses) diag.Diagnostics {
	httpResp, err := client.V1GetAFunctionWithResponse(
		ctx,
		data.ProjectRef.ValueString(),
		data.Slug.ValueString(),
	)
	if err != nil {
		msg := fmt.Sprintf("Unable to read function, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		return nil
	}

	if httpResp.JSON200 == nil {
		msg := fmt.Sprintf("Unable to read function, got status %d: %s", httpResp.StatusCode(), httpResp.Body)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	result := httpResp.JSON200
	data.Id = types.StringValue(result.Id)
	data.Slug = types.StringValue(result.Slug)
	data.Name = types.StringValue(result.Name)
	data.Status = types.StringValue(string(result.Status))
	data.Version = types.Int64Value(int64(result.Version))
	data.CreatedAt = types.Int64Value(result.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.UpdatedAt)

	if result.VerifyJwt != nil {
		data.VerifyJwt = types.BoolValue(*result.VerifyJwt)
	}

	return nil
}

func deleteFunction(ctx context.Context, data *FunctionResourceModel, client *api.ClientWithResponses) diag.Diagnostics {
	httpResp, err := client.V1DeleteAFunctionWithResponse(
		ctx,
		data.ProjectRef.ValueString(),
		data.Slug.ValueString(),
	)
	if err != nil {
		msg := fmt.Sprintf("Unable to delete function, got error: %s", err)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		return nil
	}

	if httpResp.StatusCode() != http.StatusOK {
		msg := fmt.Sprintf("Unable to delete function, got status %d: %s", httpResp.StatusCode(), httpResp.Body)
		return diag.Diagnostics{diag.NewErrorDiagnostic("Client Error", msg)}
	}

	return nil
}
