// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"

	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseAIAgentSecurityProfileImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		id                    string
		wantInstanceID        string
		wantAIAgentArn        string
		wantSecurityProfileID string
		wantErr               bool
	}{
		{
			name:                  "well formed ID with ARN containing slashes",
			id:                    "12345678-1234-1234-1234-123456789012/arn:aws:wisdom:eu-central-1:123456789012:ai-agent/97a0c52f-aaaa-bbbb-cccc-dddddddddddd/134bb661-64ba-4069-856a-d9b086034cf6",
			wantInstanceID:        "12345678-1234-1234-1234-123456789012",
			wantAIAgentArn:        "arn:aws:wisdom:eu-central-1:123456789012:ai-agent/97a0c52f-aaaa-bbbb-cccc-dddddddddddd",
			wantSecurityProfileID: "134bb661-64ba-4069-856a-d9b086034cf6",
			wantErr:               false,
		},
		{
			name:    "missing security profile id",
			id:      "instance-1/arn:aws:wisdom:eu-central-1:123456789012:ai-agent/agent-1/",
			wantErr: true,
		},
		{
			name:    "missing instance id",
			id:      "/arn:aws:wisdom:eu-central-1:123456789012:ai-agent/agent-1/sp-1",
			wantErr: true,
		},
		{
			name:    "no slashes at all",
			id:      "not-a-valid-id",
			wantErr: true,
		},
		{
			name:    "only two segments (arn without an internal slash, so first == last)",
			id:      "instance-1/sp-1",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			instanceID, aiAgentArn, securityProfileID, err := parseAIAgentSecurityProfileImportID(tc.id)

			if (err != nil) != tc.wantErr {
				t.Fatalf("parseAIAgentSecurityProfileImportID() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if instanceID != tc.wantInstanceID {
				t.Errorf("instanceID = %q, want %q", instanceID, tc.wantInstanceID)
			}
			if aiAgentArn != tc.wantAIAgentArn {
				t.Errorf("aiAgentArn = %q, want %q", aiAgentArn, tc.wantAIAgentArn)
			}
			if securityProfileID != tc.wantSecurityProfileID {
				t.Errorf("securityProfileID = %q, want %q", securityProfileID, tc.wantSecurityProfileID)
			}
		})
	}
}

const testAgentArn = "arn:aws:wisdom:eu-central-1:123456789012:ai-agent/97a0c52f-aaaa-bbbb-cccc-dddddddddddd/11111111-2222-3333-4444-555555555555"

func TestAssociatedAIAgentArnsWithoutAVersionIsTheAgentAlone(t *testing.T) {
	t.Parallel()

	got := associatedAIAgentArns(testAgentArn, frameworktypes.Int64Null())
	if len(got) != 1 || got[0] != testAgentArn {
		t.Fatalf("associatedAIAgentArns() = %v, want only the agent ARN", got)
	}
}

func TestAssociatedAIAgentArnsWithAVersionCoverEveryQualifier(t *testing.T) {
	t.Parallel()

	got := associatedAIAgentArns(testAgentArn, frameworktypes.Int64Value(3))
	want := []string{testAgentArn, testAgentArn + ":$LATEST", testAgentArn + ":$SAVED", testAgentArn + ":3"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("associatedAIAgentArns() = %v, want %v", got, want)
	}

	checked := versionArnsToCheck(testAgentArn, frameworktypes.Int64Value(3))
	if strings.Join(checked, ",") != testAgentArn+":$LATEST,"+testAgentArn+":3" {
		t.Errorf("versionArnsToCheck() = %v, want :$LATEST and :3 without :$SAVED", checked)
	}
}

func TestANewVersionMovesOnlyTheNumberedAssociation(t *testing.T) {
	t.Parallel()

	previous := associatedAIAgentArns(testAgentArn, frameworktypes.Int64Value(3))
	planned := associatedAIAgentArns(testAgentArn, frameworktypes.Int64Value(4))

	gone := arnsMissingFrom(previous, planned)
	if len(gone) != 1 || gone[0] != testAgentArn+":3" {
		t.Fatalf("arnsMissingFrom() = %v, want only :3", gone)
	}
}

func TestDroppingTheVersionRemovesEveryVersionAssociation(t *testing.T) {
	t.Parallel()

	gone := arnsMissingFrom(
		associatedAIAgentArns(testAgentArn, frameworktypes.Int64Value(3)),
		associatedAIAgentArns(testAgentArn, frameworktypes.Int64Null()),
	)
	if len(gone) != 3 {
		t.Fatalf("arnsMissingFrom() = %v, want :$LATEST, :$SAVED and :3", gone)
	}
}
