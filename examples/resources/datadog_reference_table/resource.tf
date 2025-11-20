# Example: Reference Table with AWS S3 source
resource "datadog_reference_table" "users_s3" {
  table_name  = "user_enrichment"
  description = "User metadata for log enrichment"
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
    fields {
      name = "team"
      type = "STRING"
    }
    fields {
      name = "is_active"
      type = "BOOLEAN"
    }
    primary_keys = ["user_id"]
  }

  file_metadata {
    access_details {
      aws_account_id  = "123456789012"
      aws_bucket_name = "my-datadog-reference-tables"
      file_path       = "users/user_metadata.csv"
    }
  }

  tags = ["env:prod", "team:data", "managed_by:terraform"]
}

# Example: Reference Table with GCP Cloud Storage source
resource "datadog_reference_table" "products_gcp" {
  table_name  = "product_catalog"
  description = "Product information for trace enrichment"
  source      = "GCS"

  schema {
    fields {
      name = "product_id"
      type = "STRING"
    }
    fields {
      name = "product_name"
      type = "STRING"
    }
    fields {
      name = "price"
      type = "DOUBLE"
    }
    fields {
      name = "in_stock"
      type = "BOOLEAN"
    }
    primary_keys = ["product_id"]
  }

  file_metadata {
    access_details {
      gcp_project_id           = "my-gcp-project"
      gcp_bucket_name          = "datadog-reference-data"
      gcp_service_account_email = "datadog-reader@my-project.iam.gserviceaccount.com"
      file_path                = "products/catalog.csv"
    }
  }

  tags = ["env:prod", "source:gcp"]
}

# Example: Reference Table with Azure Blob Storage source
resource "datadog_reference_table" "customers_azure" {
  table_name  = "customer_segments"
  description = "Customer segmentation data"
  source      = "AZURE"

  schema {
    fields {
      name = "customer_id"
      type = "STRING"
    }
    fields {
      name = "segment"
      type = "STRING"
    }
    fields {
      name = "lifetime_value"
      type = "DOUBLE"
    }
    primary_keys = ["customer_id"]
  }

  file_metadata {
    access_details {
      azure_client_id             = "00000000-0000-0000-0000-000000000000"
      azure_container_name        = "reference-tables"
      azure_storage_account_name  = "mydatadogstorage"
      azure_tenant_id             = "00000000-0000-0000-0000-000000000000"
      file_path                   = "customers/segments.csv"
    }
  }

  tags = ["env:prod", "source:azure"]
}

# Example: Reference Table with local file upload (requires separate upload process)
resource "datadog_reference_table" "regions_local" {
  table_name  = "service_regions"
  description = "Service region mapping"
  source      = "LOCAL_FILE"

  schema {
    fields {
      name = "region_code"
      type = "STRING"
    }
    fields {
      name = "region_name"
      type = "STRING"
    }
    fields {
      name = "enabled"
      type = "BOOLEAN"
    }
    primary_keys = ["region_code"]
  }

  file_metadata {
    # The upload_id should be obtained from the CreateReferenceTableUpload API
    # This is typically done outside of Terraform
    upload_id = "abcd1234-5678-90ef-ghij-klmnopqrstuv"
  }

  tags = ["env:staging"]
}
