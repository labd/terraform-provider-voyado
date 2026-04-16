package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/labd/terraform-provider-voyado/internal/engage"
)

var _ resource.Resource = (*interactionSchemaResource)(nil)
var _ resource.ResourceWithImportState = (*interactionSchemaResource)(nil)

type interactionSchemaResource struct {
	client *engage.Client
}

func NewInteractionSchemaResource() resource.Resource {
	return &interactionSchemaResource{}
}

func (r *interactionSchemaResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "voyado_interaction_schema"
}

func (r *interactionSchemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := stringplanmodifier.RequiresReplace()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Voyado Engage [interaction schema](https://developer.voyado.com/docs/loyalty/interactions#the-interactionschemas-endpoint). " +
			"The Engage API does not support in-place updates; changing `schema_id`, `display_name`, or `json_schema` replaces the resource (delete then create). " +
			"Deleting a schema removes all interactions for that schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Same as `schema_id` (Engage interaction schema identifier).",
				Computed:            true,
			},
			"schema_id": schema.StringAttribute{
				MarkdownDescription: "Unique schema id (`id` in the Engage API). Allowed characters: letters, digits, `_`, and `-`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{replace},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name in the Engage UI (`displayName` in the API).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{replace},
			},
			"json_schema": schema.StringAttribute{
				CustomType:          jsontypes.NormalizedType{},
				MarkdownDescription: "JSON-encoded object sent as `jsonSchema` (see Voyado documentation for required property metadata). Semantically equal JSON (whitespace, key order) is accepted.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{replace},
			},
		},
	}
}

func (r *interactionSchemaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*engage.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider configure type",
			fmt.Sprintf("expected *engage.Client, got %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

type interactionSchemaModel struct {
	ID          types.String         `tfsdk:"id"`
	SchemaID    types.String         `tfsdk:"schema_id"`
	DisplayName types.String         `tfsdk:"display_name"`
	JSONSchema  jsontypes.Normalized `tfsdk:"json_schema"`
}

func (r *interactionSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured client", "provider is not configured")
		return
	}

	var plan interactionSchemaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, d := parseJSONObject(plan.JSONSchema)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := json.Marshal(map[string]any{
		"id":          plan.SchemaID.ValueString(),
		"displayName": plan.DisplayName.ValueString(),
		"jsonSchema":  obj,
	})
	if err != nil {
		resp.Diagnostics.AddError("Marshal request body", err.Error())
		return
	}

	_, err = r.client.CreateInteractionSchema(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Engage API error creating interaction schema", err.Error())
		return
	}

	read, err := r.client.GetInteractionSchema(ctx, plan.SchemaID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Engage API error reading interaction schema after create", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applyReadResponse(&plan, read)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *interactionSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured client", "provider is not configured")
		return
	}

	var state interactionSchemaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		id = state.SchemaID.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Read error", "missing id and schema_id in state")
		return
	}

	raw, err := r.client.GetInteractionSchema(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Engage API error reading interaction schema", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applyReadResponse(&state, raw)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *interactionSchemaResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// RequiresReplace on all writable attributes; Terraform replaces instead of calling Update.
}

func (r *interactionSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured client", "provider is not configured")
		return
	}

	var state interactionSchemaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		id = state.SchemaID.ValueString()
	}
	if err := r.client.DeleteInteractionSchema(ctx, id); err != nil {
		resp.Diagnostics.AddError("Engage API error deleting interaction schema", err.Error())
		return
	}
}

func (r *interactionSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("schema_id"), req, resp)
}

func parseJSONObject(v jsontypes.Normalized) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	s := strings.TrimSpace(v.ValueString())
	if s == "" {
		diags.AddError("Invalid json_schema", "must be a non-empty JSON object")
		return nil, diags
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		diags.AddError("Invalid json_schema", err.Error())
		return nil, diags
	}
	return obj, diags
}

func (r *interactionSchemaResource) applyReadResponse(m *interactionSchemaModel, raw []byte) diag.Diagnostics {
	var diags diag.Diagnostics
	var parsed struct {
		ID          string          `json:"id"`
		DisplayName string          `json:"displayName"`
		JSONSchema  json.RawMessage `json:"jsonSchema"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		diags.AddError("Parse interaction schema response", err.Error())
		return diags
	}

	var js any
	if err := json.Unmarshal(parsed.JSONSchema, &js); err != nil {
		diags.AddError("Parse jsonSchema from response", err.Error())
		return diags
	}
	norm, err := json.Marshal(js)
	if err != nil {
		diags.AddError("Normalize jsonSchema", err.Error())
		return diags
	}

	m.ID = types.StringValue(parsed.ID)
	m.SchemaID = types.StringValue(parsed.ID)
	m.DisplayName = types.StringValue(parsed.DisplayName)
	m.JSONSchema = jsontypes.NewNormalizedValue(string(norm))
	return diags
}
