package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/terraform-providers/terraform-provider-datadog/datadog/fwprovider"
	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
)

func TestAccDatadogReferenceTable_Basic(t *testing.T) {
	t.Parallel()
	ctx, providers, accProviders := testAccFrameworkMuxProviders(context.Background(), t)
	tableName := uniqueEntityName(ctx, t)
	resourceName := "datadog_reference_table.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: accProviders,
		CheckDestroy:             testAccCheckDatadogReferenceTableDestroy(providers.frameworkProvider),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDatadogReferenceTableConfig(tableName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogReferenceTableExists(providers.frameworkProvider, resourceName),
					resource.TestCheckResourceAttr(resourceName, "table_name", tableName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test reference table"),
					resource.TestCheckResourceAttr(resourceName, "source", "S3"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.0.name", "id"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.0.type", "STRING"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.1.name", "name"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.1.type", "STRING"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.2.name", "value"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.2.type", "DOUBLE"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
				),
			},
		},
	})
}

func TestAccDatadogReferenceTable_Update(t *testing.T) {
	t.Parallel()
	ctx, providers, accProviders := testAccFrameworkMuxProviders(context.Background(), t)
	tableName := uniqueEntityName(ctx, t)
	resourceName := "datadog_reference_table.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: accProviders,
		CheckDestroy:             testAccCheckDatadogReferenceTableDestroy(providers.frameworkProvider),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDatadogReferenceTableConfig(tableName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogReferenceTableExists(providers.frameworkProvider, resourceName),
					resource.TestCheckResourceAttr(resourceName, "table_name", tableName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test reference table"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			{
				Config: testAccCheckDatadogReferenceTableConfigUpdated(tableName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogReferenceTableExists(providers.frameworkProvider, resourceName),
					resource.TestCheckResourceAttr(resourceName, "table_name", tableName),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated reference table description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "schema.fields.#", "4"),
				),
			},
		},
	})
}

func TestAccDatadogReferenceTable_Import(t *testing.T) {
	t.Parallel()
	ctx, providers, accProviders := testAccFrameworkMuxProviders(context.Background(), t)
	tableName := uniqueEntityName(ctx, t)
	resourceName := "datadog_reference_table.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: accProviders,
		CheckDestroy:             testAccCheckDatadogReferenceTableDestroy(providers.frameworkProvider),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDatadogReferenceTableConfig(tableName),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"file_metadata"},
			},
		},
	})
}

func testAccCheckDatadogReferenceTableConfig(tableName string) string {
	return fmt.Sprintf(`
resource "datadog_reference_table" "test" {
  table_name  = "%s"
  description = "Test reference table"
  source      = "S3"

  schema {
    fields {
      name = "id"
      type = "STRING"
    }
    fields {
      name = "name"
      type = "STRING"
    }
    fields {
      name = "value"
      type = "DOUBLE"
    }
  }

  file_metadata {
    access_details {
      type        = "s3"
      bucket_name = "my-test-bucket"
      key_path    = "reference-tables/test-data.csv"
      region      = "us-east-1"
    }
  }

  tags = ["env:test", "team:platform"]
}`, tableName)
}

func testAccCheckDatadogReferenceTableConfigUpdated(tableName string) string {
	return fmt.Sprintf(`
resource "datadog_reference_table" "test" {
  table_name  = "%s"
  description = "Updated reference table description"
  source      = "S3"

  schema {
    fields {
      name = "id"
      type = "STRING"
    }
    fields {
      name = "name"
      type = "STRING"
    }
    fields {
      name = "value"
      type = "DOUBLE"
    }
    fields {
      name = "enabled"
      type = "BOOLEAN"
    }
  }

  file_metadata {
    access_details {
      type        = "s3"
      bucket_name = "my-test-bucket"
      key_path    = "reference-tables/test-data.csv"
      region      = "us-east-1"
    }
  }

  tags = ["env:test", "team:platform", "version:v2"]
}`, tableName)
}

func testAccCheckDatadogReferenceTableExists(accProvider *fwprovider.FrameworkProvider, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		apiInstances := accProvider.DatadogApiInstances
		auth := accProvider.Auth

		if err := datadogReferenceTableExistsHelper(auth, s, apiInstances, resourceName); err != nil {
			return err
		}
		return nil
	}
}

func datadogReferenceTableExistsHelper(ctx context.Context, s *terraform.State, apiInstances *utils.ApiInstances, resourceName string) error {
	resource, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return fmt.Errorf("resource not found: %s", resourceName)
	}

	id := resource.Primary.ID
	if _, _, err := apiInstances.GetReferenceTablesApiV2().GetTable(ctx, id); err != nil {
		return fmt.Errorf("received an error retrieving reference table: %s", err)
	}
	return nil
}

func testAccCheckDatadogReferenceTableDestroy(accProvider *fwprovider.FrameworkProvider) func(*terraform.State) error {
	return func(s *terraform.State) error {
		apiInstances := accProvider.DatadogApiInstances
		auth := accProvider.Auth

		if err := datadogReferenceTableDestroyHelper(auth, s, apiInstances); err != nil {
			return err
		}
		return nil
	}
}

func datadogReferenceTableDestroyHelper(ctx context.Context, s *terraform.State, apiInstances *utils.ApiInstances) error {
	for _, r := range s.RootModule().Resources {
		if r.Type != "datadog_reference_table" {
			continue
		}

		id := r.Primary.ID
		if _, httpResp, err := apiInstances.GetReferenceTablesApiV2().GetTable(ctx, id); err != nil {
			if httpResp != nil && httpResp.StatusCode == 404 {
				continue
			}
			return fmt.Errorf("received an error retrieving reference table: %s", err)
		}
		return fmt.Errorf("reference table still exists")
	}
	return nil
}
