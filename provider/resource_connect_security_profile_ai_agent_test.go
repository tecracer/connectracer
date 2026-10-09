// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
)

const (
	voiceAgentArn = "arn:aws:app-integrations:eu-central-1:111122223333:application/11111111-2222-3333-4444-555555555555"
	otherAgentArn = "arn:aws:app-integrations:eu-central-1:111122223333:application/66666666-7777-8888-9999-000000000000"
)

func TestSecurityProfileAIAgentIDKeepsTheArnWhole(t *testing.T) {
	t.Parallel()

	got := composeSecurityProfileAIAgentID("instance-1", "sp-1", voiceAgentArn)
	if want := "instance-1/sp-1/" + voiceAgentArn; got != want {
		t.Fatalf("composeSecurityProfileAIAgentID() = %q, want %q", got, want)
	}
}

func TestFindAIAgent(t *testing.T) {
	t.Parallel()

	agents := []types.AIAgent{
		{Arn: aws.String(voiceAgentArn), Type: types.AIAgentTypeThirdParty},
		{Arn: aws.String(otherAgentArn), Type: types.AIAgentTypeThirdParty},
	}

	if found := findAIAgent(agents, otherAgentArn); found == nil || aws.ToString(found.Arn) != otherAgentArn {
		t.Fatalf("findAIAgent() = %v, want the second agent", found)
	}
	if found := findAIAgent(agents, "arn:missing"); found != nil {
		t.Fatalf("findAIAgent() = %v, want nil", found)
	}
}

func TestRemovingTheLastAIAgentSendsAnEmptyList(t *testing.T) {
	t.Parallel()

	kept := withoutAIAgent([]types.AIAgent{{Arn: aws.String(voiceAgentArn)}}, voiceAgentArn)
	if kept == nil || len(kept) != 0 {
		t.Fatalf("withoutAIAgent() = %#v, want an empty, non-nil list", kept)
	}
}
