# Query reference table by table name
data "datadog_reference_table" "by_name" {
  table_name = "user_enrichment"
}

# Query reference table by ID
data "datadog_reference_table" "by_id" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

# Use the data source outputs
output "table_info" {
  value = {
    id          = data.datadog_reference_table.by_name.id
    table_name  = data.datadog_reference_table.by_name.table_name
    description = data.datadog_reference_table.by_name.description
    source      = data.datadog_reference_table.by_name.source
    row_count   = data.datadog_reference_table.by_name.row_count
    status      = data.datadog_reference_table.by_name.status
    created_by  = data.datadog_reference_table.by_name.created_by
    updated_at  = data.datadog_reference_table.by_name.updated_at
  }
}

# Use with another resource that references the table
resource "datadog_logs_custom_pipeline" "example" {
  name = "Example Pipeline with Reference Table"

  processor {
    reference_table_lookup_processor {
      name   = "Enrich with user data"
      source = "user_id"
      target = "user"
      lookup_enrichment_table = data.datadog_reference_table.by_name.table_name
    }
  }
}
