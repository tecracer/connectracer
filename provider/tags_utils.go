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

// Note: We don't use plan modifiers for adding default tags because they cause
// "Provider produced invalid plan" errors in Terraform. Instead, we:
// 1. Add required tags during Create/Update operations via ensureRequiredTags()
// 2. Mark the tags attribute as both Optional and Computed
// 3. Let the Read operation populate the final tag values from AWS into state
//
// This approach ensures the tag is added without causing plan validation errors.
