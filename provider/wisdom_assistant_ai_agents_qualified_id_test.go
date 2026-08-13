// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"
)

func TestParseQualifiedAIAgentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		agentID   string
		qualifier string
	}{
		{
			input:     "97a0c52f-821f-4edf-8ec5-a45e858a5fd8:$LATEST",
			agentID:   "97a0c52f-821f-4edf-8ec5-a45e858a5fd8",
			qualifier: "$LATEST",
		},
		{
			input:     "97a0c52f-821f-4edf-8ec5-a45e858a5fd8:7",
			agentID:   "97a0c52f-821f-4edf-8ec5-a45e858a5fd8",
			qualifier: "7",
		},
		{
			input:     "97a0c52f-821f-4edf-8ec5-a45e858a5fd8",
			agentID:   "97a0c52f-821f-4edf-8ec5-a45e858a5fd8",
			qualifier: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := parseQualifiedAIAgentID(tc.input)
			if got.agentID != tc.agentID || got.qualifier != tc.qualifier {
				t.Fatalf("parseQualifiedAIAgentID(%q) = %+v, want agentID=%q qualifier=%q", tc.input, got, tc.agentID, tc.qualifier)
			}
		})
	}
}

func TestIsLatestQualifier(t *testing.T) {
	t.Parallel()

	for _, q := range []string{"", "$LATEST", "LATEST", "latest", " $LATEST "} {
		if !isLatestQualifier(q) {
			t.Fatalf("isLatestQualifier(%q) = false, want true", q)
		}
	}
	if isLatestQualifier("7") {
		t.Fatal("isLatestQualifier(\"7\") = true, want false")
	}
}

func TestResolveQualifiedAIAgentVersionInvalidQualifier(t *testing.T) {
	t.Parallel()

	_, err := resolveQualifiedAIAgentVersion(
		context.Background(),
		nil,
		"assistant-1",
		parseQualifiedAIAgentID("97a0c52f-821f-4edf-8ec5-a45e858a5fd8:not-a-version"),
	)
	if err == nil {
		t.Fatal("expected error for invalid numeric qualifier")
	}
}
