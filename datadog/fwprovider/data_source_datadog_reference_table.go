package fwprovider

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
)

var _ datasource.DataSourceWithConfigure = &referenceTableDataSource{}

func NewReferenceTableDataSource() datasource.DataSource {
	return &referenceTableDataSource{}
}

type referenceTableDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	TableName     types.String `tfsdk:"table_name"`
	Description   types.String `tfsdk:"description"`
	Source        types.String `tfsdk:"source"`
	Schema        types.Object `tfsdk:"schema"`
	Tags          types.List   `tfsdk:"tags"`
	CreatedBy     types.String `tfsdk:"created_by"`
	LastUpdatedBy types.String `tfsdk:"last_updated_by"`
	RowCount      types.Int64  `tfsdk:"row_count"`
	Status        types.String `tfsdk:"status"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

// Reuse the same model types from the resource for consistency
type referenceTableSchemaModelDataSource struct {
	Fields      types.List `tfsdk:"fields"`
	PrimaryKeys types.List `tfsdk:"primary_keys"`
}

type referenceTableSchemaFieldModelDataSource struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type referenceTableDataSource struct {
	Api  *datadogV2.ReferenceTablesApi
	Auth context.Context
}

func (d *referenceTableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "reference_table"
}

func (d *referenceTableDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	providerData := request.ProviderData.(*FrameworkProvider)
	d.Api = providerData.DatadogApiInstances.GetReferenceTablesApiV2()
	d.Auth = providerData.Auth
}

func (d *referenceTableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Use this data source to retrieve information about an existing Datadog Reference Table.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the reference table.",
				Optional:    true,
				Computed:    true,
			},
			"table_name": schema.StringAttribute{
				Description: "Unique name to identify this reference table. Required if `id` is not specified.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional text describing the purpose or contents of this reference table.",
				Computed:    true,
			},
			"source": schema.StringAttribute{
				Description: "The source type for reference table data.",
				Computed:    true,
			},
			"schema": schema.SingleNestedAttribute{
				Description: "Schema defining the structure and columns of the reference table.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"fields": schema.ListNestedAttribute{
						Description: "List of fields (columns) in the reference table.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "Name of the column.",
									Computed:    true,
								},
								"type": schema.StringAttribute{
									Description: "Data type of the column.",
									Computed:    true,
								},
							},
						},
					},
					"primary_keys": schema.ListAttribute{
						Description: "List of field names that serve as primary keys for the table.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"tags": schema.ListAttribute{
				Description: "Tags for organizing and filtering reference tables.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_by": schema.StringAttribute{
				Description: "UUID of the user who created the reference table.",
				Computed:    true,
			},
			"last_updated_by": schema.StringAttribute{
				Description: "UUID of the user who last updated the reference table.",
				Computed:    true,
			},
			"row_count": schema.Int64Attribute{
				Description: "The number of successfully processed rows in the reference table.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The processing status of the table.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "When the reference table was last updated, in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (d *referenceTableDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state referenceTableDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var tableID string

	// If ID is provided, use it directly
	if !state.ID.IsNull() && !state.ID.IsUnknown() {
		tableID = state.ID.ValueString()
	} else if !state.TableName.IsNull() && !state.TableName.IsUnknown() {
		// Otherwise, search by table name
		tableName := state.TableName.ValueString()

		// List all tables and find the one with matching name
		listResp, _, err := d.Api.ListTables(d.Auth)
		if err != nil {
			response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error listing reference tables"))
			return
		}
		if err := utils.CheckForUnparsed(listResp); err != nil {
			response.Diagnostics.AddError("response contains unparsedObject", err.Error())
			return
		}

		tables := listResp.GetData()
		found := false
		for _, table := range tables {
			if table.HasAttributes() {
				attrs := table.GetAttributes()
				if attrs.HasTableName() && attrs.GetTableName() == tableName {
					tableID = table.GetId()
					found = true
					break
				}
			}
		}

		if !found {
			response.Diagnostics.AddError(
				"Reference table not found",
				"Could not find a reference table with table_name: "+tableName,
			)
			return
		}
	} else {
		response.Diagnostics.AddError(
			"Missing required field",
			"Either 'id' or 'table_name' must be specified",
		)
		return
	}

	// Get the table by ID
	resp, httpResp, err := d.Api.GetTable(d.Auth, tableID)
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			response.Diagnostics.AddError(
				"Reference table not found",
				"The reference table with ID "+tableID+" was not found",
			)
			return
		}
		response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error retrieving reference table"))
		return
	}
	if err := utils.CheckForUnparsed(resp); err != nil {
		response.Diagnostics.AddError("response contains unparsedObject", err.Error())
		return
	}

	// Update state
	data := resp.GetData()
	state.ID = types.StringValue(data.GetId())

	if data.HasAttributes() {
		attrs := data.GetAttributes()

		if attrs.HasTableName() {
			state.TableName = types.StringValue(attrs.GetTableName())
		}

		if attrs.HasDescription() {
			state.Description = types.StringValue(attrs.GetDescription())
		} else {
			state.Description = types.StringNull()
		}

		if attrs.HasSource() {
			state.Source = types.StringValue(string(attrs.GetSource()))
		}

		if attrs.HasCreatedBy() {
			state.CreatedBy = types.StringValue(attrs.GetCreatedBy())
		}

		if attrs.HasLastUpdatedBy() {
			state.LastUpdatedBy = types.StringValue(attrs.GetLastUpdatedBy())
		}

		if attrs.HasRowCount() {
			state.RowCount = types.Int64Value(attrs.GetRowCount())
		}

		if attrs.HasStatus() {
			state.Status = types.StringValue(attrs.GetStatus())
		}

		if attrs.HasUpdatedAt() {
			state.UpdatedAt = types.StringValue(attrs.GetUpdatedAt())
		}

		if attrs.HasTags() {
			tags := attrs.GetTags()
			tagsList, diagsList := types.ListValueFrom(ctx, types.StringType, tags)
			if diagsList.HasError() {
				response.Diagnostics.Append(diagsList...)
				return
			}
			state.Tags = tagsList
		} else {
			state.Tags = types.ListNull(types.StringType)
		}

		// Update schema from response
		if attrs.HasSchema() {
			respSchema := attrs.GetSchema()
			respFields := respSchema.GetFields()
			fields := make([]referenceTableSchemaFieldModelDataSource, len(respFields))
			for i, field := range respFields {
				fields[i] = referenceTableSchemaFieldModelDataSource{
					Name: types.StringValue(field.GetName()),
					Type: types.StringValue(string(field.GetType())),
				}
			}

			// Build the schema object types
			schemaFieldType := types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name": types.StringType,
					"type": types.StringType,
				},
			}

			fieldsList, diagsList := types.ListValueFrom(ctx, schemaFieldType, fields)
			if diagsList.HasError() {
				response.Diagnostics.Append(diagsList...)
				return
			}

			primaryKeysList, primaryKeysDiags := types.ListValueFrom(ctx, types.StringType, respSchema.GetPrimaryKeys())
			if primaryKeysDiags.HasError() {
				response.Diagnostics.Append(primaryKeysDiags...)
				return
			}

			schemaAttrTypes := map[string]attr.Type{
				"fields":       types.ListType{ElemType: schemaFieldType},
				"primary_keys": types.ListType{ElemType: types.StringType},
			}

			schemaObj, diagsObj := types.ObjectValueFrom(ctx, schemaAttrTypes, referenceTableSchemaModelDataSource{
				Fields:      fieldsList,
				PrimaryKeys: primaryKeysList,
			})
			if diagsObj.HasError() {
				response.Diagnostics.Append(diagsObj...)
				return
			}
			state.Schema = schemaObj
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
