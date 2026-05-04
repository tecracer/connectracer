// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccConnectIntegrationAssociationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConnectIntegrationAssociationResourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("connectracer_connect_integration_association.test", "id"),
					resource.TestCheckResourceAttrSet("connectracer_connect_integration_association.test", "integration_association_arn"),
					resource.TestCheckResourceAttr("connectracer_connect_integration_association.test", "integration_type", "WISDOM_KNOWLEDGE_BASE"),
					resource.TestCheckResourceAttrSet("connectracer_connect_integration_association.test", "instance_id"),
					resource.TestCheckResourceAttrSet("connectracer_connect_integration_association.test", "integration_arn"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "connectracer_connect_integration_association.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["connectracer_connect_integration_association.test"]
					instanceID := rs.Primary.Attributes["instance_id"]
					id := rs.Primary.ID
					return fmt.Sprintf("%s/%s", instanceID, id), nil
				},
			},
			// Update and Read testing (tags only)
			{
				Config: testAccConnectIntegrationAssociationResourceConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("connectracer_connect_integration_association.test", "id"),
					resource.TestCheckResourceAttr("connectracer_connect_integration_association.test", "tags.Environment", "staging"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

const testAccConnectIntegrationAssociationResourceConfig = `
# Note: In a real test, you would need to provide actual Connect instance and Wisdom KB ARNs
# This is a placeholder that shows the resource structure

# Create a Wisdom assistant for testing
resource "connectracer_wisdom_assistant" "test" {
  name        = "test-assistant"
  type        = "AGENT"
  description = "Test assistant for integration association"
}

# Create a QConnect knowledge base for testing
resource "connectracer_qconnect_knowledgebase" "test" {
  name                = "test-kb"
  knowledge_base_type = "CUSTOM"
  description         = "Test KB for integration association"
}

# Note: You need a real Connect instance ID
# This is a placeholder - replace with actual instance ID in acceptance tests
variable "connect_instance_id" {
  type    = string
  default = "12345678-1234-1234-1234-123456789012"
}

resource "connectracer_connect_integration_association" "test" {
  instance_id      = var.connect_instance_id
  integration_type = "WISDOM_KNOWLEDGE_BASE"
  integration_arn  = connectracer_qconnect_knowledgebase.test.knowledge_base_arn

  tags = {
    Environment = "test"
    Terraform   = "true"
  }
}
`

const testAccConnectIntegrationAssociationResourceConfigUpdated = `
# Create a Wisdom assistant for testing
resource "connectracer_wisdom_assistant" "test" {
  name        = "test-assistant"
  type        = "AGENT"
  description = "Test assistant for integration association"
}

# Create a QConnect knowledge base for testing
resource "connectracer_qconnect_knowledgebase" "test" {
  name                = "test-kb"
  knowledge_base_type = "CUSTOM"
  description         = "Test KB for integration association"
}

variable "connect_instance_id" {
  type    = string
  default = "12345678-1234-1234-1234-123456789012"
}

resource "connectracer_connect_integration_association" "test" {
  instance_id      = var.connect_instance_id
  integration_type = "WISDOM_KNOWLEDGE_BASE"
  integration_arn  = connectracer_qconnect_knowledgebase.test.knowledge_base_arn

  tags = {
    Environment = "staging"
    Terraform   = "true"
  }
}
`
