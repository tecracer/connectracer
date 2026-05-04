// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccWisdomAssistantAssociationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWisdomAssistantAssociationResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant_association.test", "id"),
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant_association.test", "assistant_association_arn"),
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant_association.test", "assistant_id"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant_association.test", "association_type", "KNOWLEDGE_BASE"),
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant_association.test", "association_data.knowledge_base_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "connectracer_wisdom_assistant_association.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccWisdomAssistantAssociationImportStateIdFunc("connectracer_wisdom_assistant_association.test"),
			},
		},
	})
}

func TestAccWisdomAssistantAssociationResource_tags(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with tags
			{
				Config: testAccWisdomAssistantAssociationResourceConfig_tags(rName, "key1", "value1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("connectracer_wisdom_assistant_association.test", "id"),
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant_association.test", "tags.key1", "value1"),
				),
			},
			// Update tags
			{
				Config: testAccWisdomAssistantAssociationResourceConfig_tags(rName, "key1", "value1updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("connectracer_wisdom_assistant_association.test", "tags.key1", "value1updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "connectracer_wisdom_assistant_association.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccWisdomAssistantAssociationImportStateIdFunc("connectracer_wisdom_assistant_association.test"),
			},
		},
	})
}

func testAccWisdomAssistantAssociationImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		assistantID := rs.Primary.Attributes["assistant_id"]
		associationID := rs.Primary.ID

		return fmt.Sprintf("%s/%s", assistantID, associationID), nil
	}
}

func testAccWisdomAssistantAssociationResourceConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "connectracer_wisdom_assistant" "test" {
  name = %[1]q
  type = "AGENT"
}

resource "connectracer_qconnect_knowledgebase" "test" {
  name                = %[1]q
  knowledge_base_type = "CUSTOM"
}

resource "connectracer_wisdom_assistant_association" "test" {
  assistant_id     = connectracer_wisdom_assistant.test.id
  association_type = "KNOWLEDGE_BASE"

  association_data = {
    knowledge_base_id = connectracer_qconnect_knowledgebase.test.id
  }
}
`, rName)
}

func testAccWisdomAssistantAssociationResourceConfig_tags(rName, tagKey, tagValue string) string {
	return fmt.Sprintf(`
resource "connectracer_wisdom_assistant" "test" {
  name = %[1]q
  type = "AGENT"
}

resource "connectracer_qconnect_knowledgebase" "test" {
  name                = %[1]q
  knowledge_base_type = "CUSTOM"
}

resource "connectracer_wisdom_assistant_association" "test" {
  assistant_id     = connectracer_wisdom_assistant.test.id
  association_type = "KNOWLEDGE_BASE"

  association_data = {
    knowledge_base_id = connectracer_qconnect_knowledgebase.test.id
  }

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey, tagValue)
}
