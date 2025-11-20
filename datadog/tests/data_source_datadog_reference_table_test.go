package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatadogReferenceTableDataSource_Basic(t *testing.T) {
	t.Parallel()
	ctx, _, accProviders := testAccFrameworkMuxProviders(context.Background(), t)
	tableName := uniqueEntityName(ctx, t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatadogReferenceTableConfig(tableName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "table_name", tableName),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "description", "Test reference table for data source"),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "source", "S3"),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "schema.fields.#", "2"),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "schema.fields.0.name", "key"),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "schema.fields.0.type", "STRING"),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "schema.fields.1.name", "value"),
					resource.TestCheckResourceAttr("data.datadog_reference_table.test_by_name", "schema.fields.1.type", "STRING"),
					resource.TestCheckResourceAttrSet("data.datadog_reference_table.test_by_name", "id"),
					resource.TestCheckResourceAttrSet("data.datadog_reference_table.test_by_name", "created_by"),
					// Test data source by ID
					resource.TestCheckResourceAttrPair("data.datadog_reference_table.test_by_id", "id", "datadog_reference_table.test", "id"),
					resource.TestCheckResourceAttrPair("data.datadog_reference_table.test_by_id", "table_name", "datadog_reference_table.test", "table_name"),
					resource.TestCheckResourceAttrPair("data.datadog_reference_table.test_by_id", "description", "datadog_reference_table.test", "description"),
				),
			},
		},
	})
}

func testAccDataSourceDatadogReferenceTableConfig(tableName string) string {
	return fmt.Sprintf(`
resource "datadog_reference_table" "test" {
  table_name  = "%s"
  description = "Test reference table for data source"
  source      = "S3"

  schema {
    fields {
      name = "key"
      type = "STRING"
    }
    fields {
      name = "value"
      type = "STRING"
    }
    primary_keys = ["key"]
  }

  file_metadata {
    access_details {
      aws_account_id  = "123456789012"
      aws_bucket_name = "my-test-bucket"
      file_path       = "reference-tables/datasource-test.csv"
    }
  }

  tags = ["datasource:test"]
}

data "datadog_reference_table" "test_by_name" {
  table_name = datadog_reference_table.test.table_name
}

data "datadog_reference_table" "test_by_id" {
  id = datadog_reference_table.test.id
}`, tableName)
}
