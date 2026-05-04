// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWisdomAssistantResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWisdomAssistantResourceConfig("test-assistant", "Test Assistant Description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "name", "test-assistant"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "type", "AGENT"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "description", "Test Assistant Description"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "status", "ACTIVE"),
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant.test", "id"),
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant.test", "assistant_arn"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "connectracer_wisdom_assistant.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing (tags only)
			{
				Config: testAccWisdomAssistantResourceConfigWithTags("test-assistant", "Test Assistant Description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "name", "test-assistant"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "tags.Environment", "test"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant.test", "tags.Team", "platform"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccWisdomAssistantResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "connectracer_wisdom_assistant" "test" {
  name        = %[1]q
  type        = "AGENT"
  description = %[2]q
}
`, name, description)
}

func testAccWisdomAssistantResourceConfigWithTags(name, description string) string {
	return fmt.Sprintf(`
resource "connectracer_wisdom_assistant" "test" {
  name        = %[1]q
  type        = "AGENT"
  description = %[2]q
  
  tags = {
    Environment = "test"
    Team        = "platform"
  }
}
`, name, description)
}
