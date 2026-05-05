// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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

// ensureConnectEnabledTagModifier is a plan modifier that ensures AmazonConnectEnabled tag is present
type ensureConnectEnabledTagModifier struct{}

func (m *ensureConnectEnabledTagModifier) Description(ctx context.Context) string {
	return "Ensures the AmazonConnectEnabled tag is set to True"
}

func (m *ensureConnectEnabledTagModifier) MarkdownDescription(ctx context.Context) string {
	return "Ensures the `AmazonConnectEnabled` tag is set to `True`"
}

func (m *ensureConnectEnabledTagModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	// If the resource is being destroyed, don't modify the plan
	if req.Plan.Raw.IsNull() {
		return
	}

	// Get the planned tags
	var plannedTags map[string]string
	if !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() {
		diags := req.PlanValue.ElementsAs(ctx, &plannedTags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		plannedTags = make(map[string]string)
	}

	// Ensure the required tag is present
	if _, exists := plannedTags["AmazonConnectEnabled"]; !exists {
		plannedTags["AmazonConnectEnabled"] = "True"
		
		// Convert back to framework type
		modifiedTags, diags := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, plannedTags)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			resp.PlanValue = modifiedTags
		}
	}
}
