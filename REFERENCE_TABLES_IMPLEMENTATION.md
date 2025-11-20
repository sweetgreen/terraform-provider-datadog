# Reference Tables Implementation

This document describes the implementation of Datadog Reference Tables support in the Terraform provider.

## Overview

Reference Tables allow you to enrich your telemetry data with custom metadata stored in CSV files. This implementation provides both a resource to manage reference tables and a data source to query existing tables.

## What Was Implemented

### 1. Resource: `datadog_reference_table`

**Location:** `datadog/fwprovider/resource_datadog_reference_table.go`

A complete Terraform resource that supports:
- ✅ **CRUD Operations**: Create, Read, Update, Delete
- ✅ **Import**: Import existing reference tables
- ✅ **Schema Definition**: Define table structure with fields and primary keys
- ✅ **Multiple Data Sources**:
  - Local CSV file uploads (LOCAL_FILE)
  - AWS S3 buckets (S3)
  - GCP Cloud Storage (GCS)
  - Azure Blob Storage (AZURE)
- ✅ **Tags**: Organize tables with custom tags
- ✅ **Metadata Tracking**: Created by, updated by, row count, status

### 2. Data Source: `datadog_reference_table`

**Location:** `datadog/fwprovider/data_source_datadog_reference_table.go`

Query existing reference tables by:
- Table ID
- Table name

Returns all table attributes including schema, status, and metadata.

### 3. Test Coverage

**Locations:**
- `datadog/tests/resource_datadog_reference_table_test.go`
- `datadog/tests/data_source_datadog_reference_table_test.go`

Comprehensive tests for:
- Basic CRUD operations
- Update operations
- Import functionality
- Data source queries

### 4. Examples

**Locations:**
- `examples/resources/datadog_reference_table/resource.tf`
- `examples/data-sources/datadog_reference_table/data-source.tf`

Complete working examples for all supported cloud storage providers.

## Architecture

### Resource Schema

```hcl
resource "datadog_reference_table" "example" {
  table_name  = "unique_table_name"        # Required, immutable
  description = "Table description"         # Optional
  source      = "S3|GCS|AZURE|LOCAL_FILE"  # Required, immutable

  schema {
    fields {
      name = "column_name"
      type = "STRING|DOUBLE|BOOLEAN"
    }
    primary_keys = ["column_name"]  # Required, used for lookups
  }

  file_metadata {
    # Option 1: Local file upload
    upload_id = "uuid-from-upload-api"

    # Option 2: Cloud storage
    access_details {
      # AWS S3 fields
      aws_account_id  = "..."
      aws_bucket_name = "..."

      # GCP fields
      gcp_project_id           = "..."
      gcp_bucket_name          = "..."
      gcp_service_account_email = "..."

      # Azure fields
      azure_client_id             = "..."
      azure_container_name        = "..."
      azure_storage_account_name  = "..."
      azure_tenant_id             = "..."

      # Common field
      file_path = "path/to/file.csv"
    }
  }

  tags = ["key:value"]  # Optional
}
```

### Cloud Storage Support

The implementation properly handles the provider-specific access details:

#### AWS S3
- `aws_account_id`: AWS account ID
- `aws_bucket_name`: S3 bucket name
- `file_path`: Path to CSV file

#### GCP Cloud Storage
- `gcp_project_id`: GCP project ID
- `gcp_bucket_name`: GCS bucket name
- `gcp_service_account_email`: Service account with read permissions
- `file_path`: Path to CSV file

#### Azure Blob Storage
- `azure_client_id`: Service principal client ID
- `azure_container_name`: Blob container name
- `azure_storage_account_name`: Storage account name
- `azure_tenant_id`: Azure AD tenant ID
- `file_path`: Path to CSV file

### API Integration

The implementation uses the Datadog API v2 Reference Tables endpoints:
- `POST /api/v2/reference-tables` - Create table
- `GET /api/v2/reference-tables/{id}` - Get table
- `PATCH /api/v2/reference-tables/{id}` - Update table
- `DELETE /api/v2/reference-tables/{id}` - Delete table
- `GET /api/v2/reference-tables` - List tables

**API Client:** `GetReferenceTablesApiV2()` in `datadog/internal/utils/api_instances_helper.go`

## Usage Examples

### AWS S3 Example

```hcl
resource "datadog_reference_table" "users" {
  table_name  = "user_enrichment"
  description = "User metadata"
  source      = "S3"

  schema {
    fields {
      name = "user_id"
      type = "STRING"
    }
    fields {
      name = "department"
      type = "STRING"
    }
    primary_keys = ["user_id"]
  }

  file_metadata {
    access_details {
      aws_account_id  = "123456789012"
      aws_bucket_name = "my-bucket"
      file_path       = "users/metadata.csv"
    }
  }

  tags = ["env:prod"]
}
```

### Data Source Query

```hcl
data "datadog_reference_table" "existing" {
  table_name = "user_enrichment"
}

output "table_row_count" {
  value = data.datadog_reference_table.existing.row_count
}
```

## Code Quality

The implementation follows all repository standards:
- ✅ Uses Terraform Plugin Framework (not legacy SDK)
- ✅ Consistent error handling with `utils.FrameworkErrorDiag()`
- ✅ Proper 404 handling (removes from state)
- ✅ Complete schema descriptions for documentation generation
- ✅ Import support with `ImportStatePassthroughID`
- ✅ Plan modifiers for computed fields
- ✅ Comprehensive diagnostics

## Testing

### Unit Tests
Run with: `make test`

### Acceptance Tests
Run with: `DD_TEST_CLIENT_API_KEY=xxx DD_TEST_CLIENT_APP_KEY=yyy make testacc`

**Note:** Acceptance tests require:
- Valid Datadog API and App keys
- Appropriate permissions to create/delete reference tables
- Access to cloud storage (for S3/GCS/Azure tests)

## CSV File Requirements

Reference table CSV files must:
1. Have a header row with column names
2. Match the schema defined in Terraform
3. Include the primary key column(s)
4. Use proper CSV formatting (quoted fields for special characters)
5. Be accessible from Datadog's infrastructure (for cloud storage)

Example CSV:
```csv
user_id,department,team,is_active
user_001,Engineering,Platform,true
user_002,Sales,Enterprise,true
user_003,Marketing,Growth,false
```

## Documentation Generation

Generate documentation with:
```bash
make docs
```

This creates/updates:
- `docs/resources/reference_table.md`
- `docs/data-sources/reference_table.md`

## Known Limitations

1. **Local File Uploads**: The `upload_id` must be obtained through the Datadog API before creating the resource. This is typically a two-step process:
   - Call `CreateReferenceTableUpload` API to get upload URLs
   - Upload CSV chunks to the provided URLs
   - Use the returned `upload_id` in Terraform

2. **Primary Keys**: Only one primary key is supported by the API

3. **File Updates**: Updating the file_metadata (changing the CSV file) requires careful consideration as it may cause the table to be recreated

4. **Sync Schedule**: Cloud storage tables are synced automatically, but sync frequency is controlled by Datadog

## Future Enhancements

Potential improvements:
- [ ] Support for creating upload_id within the resource
- [ ] Validation for CSV file format
- [ ] Support for large file uploads in chunks
- [ ] Integration with other Datadog resources (logs pipelines, etc.)
- [ ] Better handling of sync status and errors

## Related Documentation

- [Datadog Reference Tables Docs](https://docs.datadoghq.com/reference_tables/)
- [Datadog API Reference](https://docs.datadoghq.com/api/latest/reference-tables/)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)

## Support

For issues or questions:
- File an issue in the provider repository
- Check Datadog documentation for API-specific questions
- Review test files for implementation examples
