// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// ensureRequiredTags ensures the AmazonConnectEnabled tag is set
func ensureRequiredTags(ctx context.Context, userTags frameworktypes.Map) (map[string]string, error) {
	tags := make(map[string]string)

	// Copy user-provided tags
	if !userTags.IsNull() && !userTags.IsUnknown() {
		elements := make(map[string]string)
		diags := userTags.ElementsAs(ctx, &elements, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to read tags")
		}
		for k, v := range elements {
			tags[k] = v
		}
	}

	// Ensure required tag is present
	if _, exists := tags["AmazonConnectEnabled"]; !exists {
		tags["AmazonConnectEnabled"] = "True"
	}

	return tags, nil
}

// Note: Plan modifiers that change the plan value cause validation errors in Terraform.
// Instead, we add required tags in the Create/Update methods and mark tags as Computed.
// This allows the provider to add tags without causing plan validation failures.
