// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccQConnectKnowledgeBaseDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccQConnectKnowledgeBaseDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.connectracer_qconnect_knowledgebase.test", "id"),
					resource.TestCheckResourceAttrSet("data.connectracer_qconnect_knowledgebase.test", "knowledge_base_id"),
					resource.TestCheckResourceAttrSet("data.connectracer_qconnect_knowledgebase.test", "knowledge_base_arn"),
					resource.TestCheckResourceAttrSet("data.connectracer_qconnect_knowledgebase.test", "name"),
					resource.TestCheckResourceAttrSet("data.connectracer_qconnect_knowledgebase.test", "knowledge_base_type"),
					resource.TestCheckResourceAttrSet("data.connectracer_qconnect_knowledgebase.test", "status"),
				),
			},
		},
	})
}

const testAccQConnectKnowledgeBaseDataSourceConfig = `
# Create a knowledge base
resource "connectracer_qconnect_knowledgebase" "test" {
  name                = "test-kb-datasource"
  knowledge_base_type = "CUSTOM"
  description         = "Test KB for datasource"

  tags = {
    Environment = "test"
    Purpose     = "datasource-test"
  }
}

# Look up the knowledge base using the datasource
data "connectracer_qconnect_knowledgebase" "test" {
  knowledge_base_id = connectracer_qconnect_knowledgebase.test.id
}
`
