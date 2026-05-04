// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccQConnectKnowledgeBaseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccQConnectKnowledgeBaseResourceConfig("test-kb", "CUSTOM", "Test Knowledge Base Description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "name", "test-kb"),
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "knowledge_base_type", "CUSTOM"),
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "description", "Test Knowledge Base Description"),
					resource.TestCheckResourceAttrSet("connectracer_qconnect_knowledgebase.test", "id"),
					resource.TestCheckResourceAttrSet("connectracer_qconnect_knowledgebase.test", "knowledge_base_arn"),
					resource.TestCheckResourceAttrSet("connectracer_qconnect_knowledgebase.test", "status"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "connectracer_qconnect_knowledgebase.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing (tags only)
			{
				Config: testAccQConnectKnowledgeBaseResourceConfigWithTags("test-kb", "CUSTOM", "Test Knowledge Base Description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "name", "test-kb"),
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "tags.Environment", "test"),
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "tags.Team", "platform"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccQConnectKnowledgeBaseResourceWithEncryption(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with encryption
			{
				Config: testAccQConnectKnowledgeBaseResourceConfigWithEncryption("test-kb-encrypted", "CUSTOM"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "name", "test-kb-encrypted"),
					resource.TestCheckResourceAttr("connectracer_qconnect_knowledgebase.test", "knowledge_base_type", "CUSTOM"),
					resource.TestCheckResourceAttrSet("connectracer_qconnect_knowledgebase.test", "server_side_encryption_configuration.kms_key_id"),
				),
			},
		},
	})
}

func testAccQConnectKnowledgeBaseResourceConfig(name, kbType, description string) string {
	return fmt.Sprintf(`
resource "connectracer_qconnect_knowledgebase" "test" {
  name                = %[1]q
  knowledge_base_type = %[2]q
  description         = %[3]q
}
`, name, kbType, description)
}

func testAccQConnectKnowledgeBaseResourceConfigWithTags(name, kbType, description string) string {
	return fmt.Sprintf(`
resource "connectracer_qconnect_knowledgebase" "test" {
  name                = %[1]q
  knowledge_base_type = %[2]q
  description         = %[3]q
  
  tags = {
    Environment = "test"
    Team        = "platform"
  }
}
`, name, kbType, description)
}

func testAccQConnectKnowledgeBaseResourceConfigWithEncryption(name, kbType string) string {
	return fmt.Sprintf(`
resource "connectracer_qconnect_knowledgebase" "test" {
  name                = %[1]q
  knowledge_base_type = %[2]q
  description         = "Knowledge Base with encryption"

  server_side_encryption_configuration {
    kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  }
}
`, name, kbType)
}
