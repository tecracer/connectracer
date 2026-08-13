// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	qconnecttypes "github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const testAgentID = "97a0c52f-821f-4edf-8ec5-a45e858a5fd8"

func TestValidateOrchestratorUseCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		agents  map[string]string
		useCase map[string]string
		wantErr bool
	}{
		{
			name:    "orchestration requires use case",
			agents:  map[string]string{"ORCHESTRATION": testAgentID + ":$LATEST"},
			useCase: map[string]string{},
			wantErr: true,
		},
		{
			name:    "orchestration with use case ok",
			agents:  map[string]string{"ORCHESTRATION": testAgentID + ":$LATEST"},
			useCase: map[string]string{"ORCHESTRATION": "Connect.SelfService"},
			wantErr: false,
		},
		{
			name:    "non orchestration types skip validation",
			agents:  map[string]string{"SELF_SERVICE": testAgentID + ":1"},
			useCase: map[string]string{},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateOrchestratorUseCases(tc.agents, tc.useCase)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateOrchestratorUseCases() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestUpdateAssistantAIAgentInputOrchestratorUseCase(t *testing.T) {
	t.Parallel()

	input := updateAssistantAIAgentInput(
		"assistant-1",
		string(qconnecttypes.AIAgentTypeOrchestration),
		testAgentID+":$LATEST",
		map[string]string{"ORCHESTRATION": "Connect.SelfService"},
	)

	if aws.ToString(input.OrchestratorUseCase) != "Connect.SelfService" {
		t.Fatalf("OrchestratorUseCase = %q, want Connect.SelfService", aws.ToString(input.OrchestratorUseCase))
	}
	if aws.ToString(input.Configuration.AiAgentId) != testAgentID+":$LATEST" {
		t.Fatalf("AiAgentId = %q", aws.ToString(input.Configuration.AiAgentId))
	}

	withoutUseCase := updateAssistantAIAgentInput(
		"assistant-1",
		string(qconnecttypes.AIAgentTypeSelfService),
		testAgentID+":1",
		map[string]string{"ORCHESTRATION": "Connect.SelfService"},
	)
	if withoutUseCase.OrchestratorUseCase != nil {
		t.Fatalf("OrchestratorUseCase should be nil for SELF_SERVICE, got %q", aws.ToString(withoutUseCase.OrchestratorUseCase))
	}
}

func TestRemoveAssistantAIAgentInputOrchestratorUseCase(t *testing.T) {
	t.Parallel()

	input := removeAssistantAIAgentInput(
		"assistant-1",
		string(qconnecttypes.AIAgentTypeOrchestration),
		map[string]string{"ORCHESTRATION": "Connect.SelfService"},
	)
	if aws.ToString(input.OrchestratorUseCase) != "Connect.SelfService" {
		t.Fatalf("OrchestratorUseCase = %q, want Connect.SelfService", aws.ToString(input.OrchestratorUseCase))
	}
}
