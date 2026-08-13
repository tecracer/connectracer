// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAiAgentPlanRequiresNewVersion(t *testing.T) {
	t.Parallel()

	base := ConnectAIAgentResourceModel{
		Description:      frameworktypes.StringValue("desc"),
		VisibilityStatus: frameworktypes.StringValue("PUBLISHED"),
		Tags:             frameworktypes.MapNull(frameworktypes.StringType),
		OrchestrationConfiguration: []OrchestrationConfigModel{
			{
				OrchestrationAIPromptId: frameworktypes.StringValue("prompt-1:$LATEST"),
				ConnectInstanceArn:      frameworktypes.StringValue("arn:aws:connect:eu-central-1:123:instance/abc"),
				Locale:                  frameworktypes.StringValue("en_US"),
			},
		},
	}

	t.Run("unchanged plan does not require version", func(t *testing.T) {
		t.Parallel()
		if aiAgentPlanRequiresNewVersion(&base, &base) {
			t.Fatal("expected no new version for identical plan/state")
		}
	})

	t.Run("prompt change requires version", func(t *testing.T) {
		t.Parallel()
		plan := base
		plan.OrchestrationConfiguration = []OrchestrationConfigModel{
			{
				OrchestrationAIPromptId: frameworktypes.StringValue("prompt-2:$LATEST"),
				ConnectInstanceArn:      base.OrchestrationConfiguration[0].ConnectInstanceArn,
				Locale:                  base.OrchestrationConfiguration[0].Locale,
			},
		}
		if !aiAgentPlanRequiresNewVersion(&plan, &base) {
			t.Fatal("expected new version when orchestration prompt changes")
		}
	})

	t.Run("visibility change requires version", func(t *testing.T) {
		t.Parallel()
		plan := base
		plan.VisibilityStatus = frameworktypes.StringValue("SAVED")
		if !aiAgentPlanRequiresNewVersion(&plan, &base) {
			t.Fatal("expected new version when visibility changes")
		}
	})

	t.Run("description change requires version", func(t *testing.T) {
		t.Parallel()
		plan := base
		plan.Description = frameworktypes.StringValue("new desc")
		if !aiAgentPlanRequiresNewVersion(&plan, &base) {
			t.Fatal("expected new version when description changes")
		}
	})
}

func TestOrchestrationConfigEqual(t *testing.T) {
	t.Parallel()

	a := []OrchestrationConfigModel{
		{
			OrchestrationAIPromptId: frameworktypes.StringValue("prompt-1"),
			ConnectInstanceArn:      frameworktypes.StringValue("arn:instance"),
			Locale:                  frameworktypes.StringValue("de_DE"),
		},
	}
	b := []OrchestrationConfigModel{
		{
			OrchestrationAIPromptId: frameworktypes.StringValue("prompt-1"),
			ConnectInstanceArn:      frameworktypes.StringValue("arn:instance"),
			Locale:                  frameworktypes.StringValue("de_DE"),
		},
	}
	if !orchestrationConfigEqual(a, b) {
		t.Fatal("expected equal orchestration configs")
	}

	b[0].Locale = frameworktypes.StringValue("en_US")
	if orchestrationConfigEqual(a, b) {
		t.Fatal("expected different locale to be unequal")
	}
}
