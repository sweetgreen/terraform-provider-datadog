package fwprovider

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkPath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &referenceTableResource{}
	_ resource.ResourceWithImportState = &referenceTableResource{}
)

func NewReferenceTableResource() resource.Resource {
	return &referenceTableResource{}
}

type referenceTableResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TableName     types.String `tfsdk:"table_name"`
	Description   types.String `tfsdk:"description"`
	Source        types.String `tfsdk:"source"`
	Schema        types.Object `tfsdk:"schema"`
	FileMetadata  types.Object `tfsdk:"file_metadata"`
	Tags          types.List   `tfsdk:"tags"`
	CreatedBy     types.String `tfsdk:"created_by"`
	LastUpdatedBy types.String `tfsdk:"last_updated_by"`
	RowCount      types.Int64  `tfsdk:"row_count"`
	Status        types.String `tfsdk:"status"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

type referenceTableSchemaModel struct {
	Fields      types.List `tfsdk:"fields"`
	PrimaryKeys types.List `tfsdk:"primary_keys"`
}

type referenceTableSchemaFieldModel struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type referenceTableFileMetadataModel struct {
	UploadID      types.String `tfsdk:"upload_id"`
	AccessDetails types.Object `tfsdk:"access_details"`
}

type referenceTableAccessDetailsModel struct {
	Type       types.String `tfsdk:"type"`
	Region     types.String `tfsdk:"region"`
	BucketName types.String `tfsdk:"bucket_name"`
	KeyPath    types.String `tfsdk:"key_path"`
}

type referenceTableResource struct {
	Api  *datadogV2.ReferenceTablesApi
	Auth context.Context
}

func (r *referenceTableResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	providerData := request.ProviderData.(*FrameworkProvider)
	r.Api = providerData.DatadogApiInstances.GetReferenceTablesApiV2()
	r.Auth = providerData.Auth
}

func (r *referenceTableResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "reference_table"
}

func (r *referenceTableResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Provides a Datadog Reference Table resource. This can be used to create and manage Datadog Reference Tables for data enrichment.",
		Attributes: map[string]schema.Attribute{
			"table_name": schema.StringAttribute{
				Description: "Unique name to identify this reference table. Used in enrichment processors and API calls.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Optional text describing the purpose or contents of this reference table.",
				Optional:    true,
			},
			"source": schema.StringAttribute{
				Description: "The source type for reference table data. Valid values are `LOCAL_FILE`, `S3`, `GCS`, `AZURE`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema": schema.SingleNestedAttribute{
				Description: "Schema defining the structure and columns of the reference table.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"fields": schema.ListNestedAttribute{
						Description: "List of fields (columns) in the reference table.",
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "Name of the column.",
									Required:    true,
								},
								"type": schema.StringAttribute{
									Description: "Data type of the column. Valid values are `STRING`, `DOUBLE`, `BOOLEAN`.",
									Required:    true,
								},
							},
						},
					},
					"primary_keys": schema.ListAttribute{
						Description: "List of field names that serve as primary keys for the table. Only one primary key is supported.",
						Required:    true,
						ElementType: types.StringType,
					},
				},
			},
			"file_metadata": schema.SingleNestedAttribute{
				Description: "Metadata specifying where and how to access the reference table's data file.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"upload_id": schema.StringAttribute{
						Description: "Upload ID obtained from creating a reference table upload. Use this for LOCAL_FILE source type.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"access_details": schema.SingleNestedAttribute{
						Description: "Details for accessing a file in cloud storage. Use this for S3, GCS, or AZURE source types.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Type of cloud storage. Valid values are `s3`, `gcs`, `azure`.",
								Required:    true,
							},
							"region": schema.StringAttribute{
								Description: "Region where the bucket is located (for S3).",
								Optional:    true,
							},
							"bucket_name": schema.StringAttribute{
								Description: "Name of the storage bucket.",
								Required:    true,
							},
							"key_path": schema.StringAttribute{
								Description: "Path to the CSV file within the bucket.",
								Required:    true,
							},
						},
					},
				},
			},
			"tags": schema.ListAttribute{
				Description: "Tags for organizing and filtering reference tables.",
				Optional:    true,
				ElementType: types.StringType,
			},
			// Computed fields
			"created_by": schema.StringAttribute{
				Description: "UUID of the user who created the reference table.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			"id": utils.ResourceIDAttribute(),
		},
	}
}

func (r *referenceTableResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state referenceTableResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiRequest, diags := r.buildCreateRequest(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	resp, httpResp, err := r.Api.CreateReferenceTable(r.Auth, *apiRequest)
	if err != nil {
		response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error creating reference table"))
		return
	}
	if err := utils.CheckForUnparsed(resp); err != nil {
		response.Diagnostics.AddError("response contains unparsedObject", err.Error())
		return
	}
	if httpResp != nil && httpResp.StatusCode != 200 {
		response.Diagnostics.AddError("unexpected status code", "expected 200, got "+string(rune(httpResp.StatusCode)))
		return
	}

	// Update state with response
	updateDiags := r.updateStateFromResponse(ctx, &state, &resp)
	response.Diagnostics.Append(updateDiags...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *referenceTableResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state referenceTableResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	resp, httpResp, err := r.Api.GetTable(r.Auth, state.ID.ValueString())
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error retrieving reference table"))
		return
	}
	if err := utils.CheckForUnparsed(resp); err != nil {
		response.Diagnostics.AddError("response contains unparsedObject", err.Error())
		return
	}

	updateDiags := r.updateStateFromResponse(ctx, &state, &resp)
	response.Diagnostics.Append(updateDiags...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *referenceTableResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state referenceTableResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiRequest, diags := r.buildUpdateRequest(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.Api.UpdateReferenceTable(r.Auth, state.ID.ValueString(), *apiRequest)
	if err != nil {
		response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error updating reference table"))
		return
	}
	if httpResp != nil && httpResp.StatusCode != 200 {
		response.Diagnostics.AddError("unexpected status code", "expected 200, got "+string(rune(httpResp.StatusCode)))
		return
	}

	// Read back the updated resource
	resp, httpResp, err := r.Api.GetTable(r.Auth, state.ID.ValueString())
	if err != nil {
		response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error retrieving updated reference table"))
		return
	}
	if err := utils.CheckForUnparsed(resp); err != nil {
		response.Diagnostics.AddError("response contains unparsedObject", err.Error())
		return
	}

	updateDiags := r.updateStateFromResponse(ctx, &state, &resp)
	response.Diagnostics.Append(updateDiags...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *referenceTableResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state referenceTableResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.Api.DeleteTable(r.Auth, state.ID.ValueString())
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			// Already deleted
			return
		}
		response.Diagnostics.Append(utils.FrameworkErrorDiag(err, "error deleting reference table"))
	}
}

func (r *referenceTableResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, frameworkPath.Root("id"), request, response)
}

func (r *referenceTableResource) buildCreateRequest(ctx context.Context, state *referenceTableResourceModel) (*datadogV2.CreateTableRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build schema
	var schemaModel referenceTableSchemaModel
	if diagsConvert := state.Schema.As(ctx, &schemaModel, basetypes.ObjectAsOptions{}); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	var fields []referenceTableSchemaFieldModel
	if diagsConvert := schemaModel.Fields.ElementsAs(ctx, &fields, false); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	schemaFields := make([]datadogV2.CreateTableRequestDataAttributesSchemaFieldsItems, len(fields))
	for i, field := range fields {
		fieldType, err := datadogV2.NewReferenceTableSchemaFieldTypeFromValue(field.Type.ValueString())
		if err != nil {
			diags.Append(utils.FrameworkErrorDiag(err, "invalid field type"))
			return nil, diags
		}
		schemaFields[i] = *datadogV2.NewCreateTableRequestDataAttributesSchemaFieldsItems(
			field.Name.ValueString(),
			*fieldType,
		)
	}

	var primaryKeys []string
	if diagsConvert := schemaModel.PrimaryKeys.ElementsAs(ctx, &primaryKeys, false); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	schemaObj := *datadogV2.NewCreateTableRequestDataAttributesSchema(schemaFields, primaryKeys)

	// Build file metadata
	var fileMetadataModel referenceTableFileMetadataModel
	if diagsConvert := state.FileMetadata.As(ctx, &fileMetadataModel, basetypes.ObjectAsOptions{}); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	var fileMetadata datadogV2.CreateTableRequestDataAttributesFileMetadata

	if !fileMetadataModel.UploadID.IsNull() && !fileMetadataModel.UploadID.IsUnknown() {
		// Local file upload
		localFile := datadogV2.NewCreateTableRequestDataAttributesFileMetadataLocalFile(fileMetadataModel.UploadID.ValueString())
		fileMetadata = datadogV2.CreateTableRequestDataAttributesFileMetadataLocalFileAsCreateTableRequestDataAttributesFileMetadata(localFile)
	} else if !fileMetadataModel.AccessDetails.IsNull() && !fileMetadataModel.AccessDetails.IsUnknown() {
		// Cloud storage
		var accessDetailsModel referenceTableAccessDetailsModel
		if diagsConvert := fileMetadataModel.AccessDetails.As(ctx, &accessDetailsModel, basetypes.ObjectAsOptions{}); diagsConvert.HasError() {
			diags.Append(diagsConvert...)
			return nil, diags
		}

		accessDetails := datadogV2.NewCreateTableRequestDataAttributesFileMetadataOneOfAccessDetails(
			accessDetailsModel.BucketName.ValueString(),
			accessDetailsModel.KeyPath.ValueString(),
			accessDetailsModel.Type.ValueString(),
		)

		if !accessDetailsModel.Region.IsNull() && !accessDetailsModel.Region.IsUnknown() {
			accessDetails.SetRegion(accessDetailsModel.Region.ValueString())
		}

		cloudStorage := datadogV2.NewCreateTableRequestDataAttributesFileMetadataCloudStorage(*accessDetails, true)
		fileMetadata = datadogV2.CreateTableRequestDataAttributesFileMetadataCloudStorageAsCreateTableRequestDataAttributesFileMetadata(cloudStorage)
	} else {
		diags.AddError("invalid file metadata", "either upload_id or access_details must be specified")
		return nil, diags
	}

	// Parse source type
	sourceType, err := datadogV2.NewReferenceTableCreateSourceTypeFromValue(state.Source.ValueString())
	if err != nil {
		diags.Append(utils.FrameworkErrorDiag(err, "invalid source type"))
		return nil, diags
	}

	// Build attributes
	attributes := datadogV2.NewCreateTableRequestDataAttributes(
		schemaObj,
		*sourceType,
		state.TableName.ValueString(),
	)

	if !state.Description.IsNull() && !state.Description.IsUnknown() {
		attributes.SetDescription(state.Description.ValueString())
	}

	attributes.SetFileMetadata(fileMetadata)

	if !state.Tags.IsNull() && !state.Tags.IsUnknown() {
		var tags []string
		if diagsConvert := state.Tags.ElementsAs(ctx, &tags, false); diagsConvert.HasError() {
			diags.Append(diagsConvert...)
			return nil, diags
		}
		attributes.SetTags(tags)
	}

	// Build request
	data := datadogV2.NewCreateTableRequestData(datadogV2.CREATETABLEREQUESTDATATYPE_REFERENCE_TABLE)
	data.SetAttributes(*attributes)

	request := datadogV2.NewCreateTableRequest()
	request.SetData(*data)

	return request, diags
}

func (r *referenceTableResource) buildUpdateRequest(ctx context.Context, state *referenceTableResourceModel) (*datadogV2.PatchTableRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build schema
	var schemaModel referenceTableSchemaModel
	if diagsConvert := state.Schema.As(ctx, &schemaModel, basetypes.ObjectAsOptions{}); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	var fields []referenceTableSchemaFieldModel
	if diagsConvert := schemaModel.Fields.ElementsAs(ctx, &fields, false); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	schemaFields := make([]datadogV2.PatchTableRequestDataAttributesSchemaFieldsItems, len(fields))
	for i, field := range fields {
		fieldType, err := datadogV2.NewReferenceTableSchemaFieldTypeFromValue(field.Type.ValueString())
		if err != nil {
			diags.Append(utils.FrameworkErrorDiag(err, "invalid field type"))
			return nil, diags
		}
		schemaFields[i] = *datadogV2.NewPatchTableRequestDataAttributesSchemaFieldsItems(
			field.Name.ValueString(),
			*fieldType,
		)
	}

	var primaryKeys []string
	if diagsConvert := schemaModel.PrimaryKeys.ElementsAs(ctx, &primaryKeys, false); diagsConvert.HasError() {
		diags.Append(diagsConvert...)
		return nil, diags
	}

	schemaObj := datadogV2.NewPatchTableRequestDataAttributesSchema(schemaFields, primaryKeys)

	// Build attributes
	attributes := datadogV2.NewPatchTableRequestDataAttributes()
	attributes.SetSchema(*schemaObj)

	if !state.Description.IsNull() && !state.Description.IsUnknown() {
		attributes.SetDescription(state.Description.ValueString())
	}

	if !state.Tags.IsNull() && !state.Tags.IsUnknown() {
		var tags []string
		if diagsConvert := state.Tags.ElementsAs(ctx, &tags, false); diagsConvert.HasError() {
			diags.Append(diagsConvert...)
			return nil, diags
		}
		attributes.SetTags(tags)
	}

	// Build request
	data := datadogV2.NewPatchTableRequestData(datadogV2.PATCHTABLEREQUESTDATATYPE_REFERENCE_TABLE)
	data.SetAttributes(*attributes)

	request := datadogV2.NewPatchTableRequest()
	request.SetData(*data)

	return request, diags
}

func (r *referenceTableResource) updateStateFromResponse(ctx context.Context, state *referenceTableResourceModel, resp *datadogV2.TableResultV2) diag.Diagnostics {
	var diags diag.Diagnostics

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
				diags.Append(diagsList...)
				return diags
			}
			state.Tags = tagsList
		} else {
			state.Tags = types.ListNull(types.StringType)
		}

		// Update schema from response
		if attrs.HasSchema() {
			respSchema := attrs.GetSchema()
			respFields := respSchema.GetFields()
			fields := make([]referenceTableSchemaFieldModel, len(respFields))
			for i, field := range respFields {
				fields[i] = referenceTableSchemaFieldModel{
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
				diags.Append(diagsList...)
				return diags
			}

			schemaAttrTypes := map[string]attr.Type{
				"fields":       types.ListType{ElemType: schemaFieldType},
				"primary_keys": types.ListType{ElemType: types.StringType},
			}

			primaryKeysList, primaryKeysDiags := types.ListValueFrom(ctx, types.StringType, respSchema.GetPrimaryKeys())
			if primaryKeysDiags.HasError() {
				diags.Append(primaryKeysDiags...)
				return diags
			}

			schemaObj, diagsObj := types.ObjectValueFrom(ctx, schemaAttrTypes, referenceTableSchemaModel{
				Fields:      fieldsList,
				PrimaryKeys: primaryKeysList,
			})
			if diagsObj.HasError() {
				diags.Append(diagsObj...)
				return diags
			}
			state.Schema = schemaObj
		}
	}

	return diags
}
