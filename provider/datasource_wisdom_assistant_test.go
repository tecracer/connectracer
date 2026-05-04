// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWisdomAssistantsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccWisdomAssistantsDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.connectracer_wisdom_assistants.test", "id", "wisdom-assistants"),
					resource.TestCheckResourceAttrSet("data.connectracer_wisdom_assistants.test", "assistants.#"),
				),
			},
		},
	})
}

const testAccWisdomAssistantsDataSourceConfig = `
data "connectracer_wisdom_assistants" "test" {
}
`
