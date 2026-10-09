// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMultiAgentConfigurationsRoundTrip(t *testing.T) {
	handoffs := []AgentHandoffModel{{
		ApplicationID:         frameworktypes.StringValue("11111111-2222-3333-4444-555555555555"),
		AIAgentID:             frameworktypes.StringNull(),
		Instruction:           frameworktypes.StringValue("Transfer every voice call."),
		AudioStreamingEnabled: frameworktypes.BoolValue(true),
		ImmediateHandoff:      frameworktypes.BoolValue(true),
	}}
	delegates := []AgentDelegateModel{{
		ApplicationID: frameworktypes.StringNull(),
		AIAgentID:     frameworktypes.StringValue("66666666-7777-8888-9999-000000000000"),
		Instruction:   frameworktypes.StringValue("Ask for a credit check."),
		Examples:      []frameworktypes.String{frameworktypes.StringValue("Can I pay later?")},
	}}

	expanded := expandMultiAgentConfigurations(handoffs, delegates)
	if len(expanded) != 2 {
		t.Fatalf("expandMultiAgentConfigurations() returned %d configurations, want 2", len(expanded))
	}
	handoff, ok := expanded[0].(*types.MultiAgentConfigurationMemberHandoffAgentConfiguration)
	if !ok {
		t.Fatalf("first configuration is %T, want a handoff", expanded[0])
	}
	if target, ok := handoff.Value.AgentTarget.(*types.AgentTargetMemberApplicationId); !ok || target.Value != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("handoff target = %#v, want the application id", handoff.Value.AgentTarget)
	}
	if !aws.ToBool(handoff.Value.AudioStreamingEnabled) || !aws.ToBool(handoff.Value.ImmediateHandoff) {
		t.Errorf("handoff = %+v, want audio streaming and an immediate handoff", handoff.Value)
	}

	gotHandoffs, gotDelegates := flattenMultiAgentConfigurations(expanded)
	if !handoffsEqual(gotHandoffs, handoffs) {
		t.Errorf("handoffs read back as %+v, want %+v", gotHandoffs, handoffs)
	}
	if !delegatesEqual(gotDelegates, delegates) {
		t.Errorf("delegates read back as %+v, want %+v", gotDelegates, delegates)
	}
}

func TestAnAgentWithoutCollaboratorsSendsNone(t *testing.T) {
	if expandMultiAgentConfigurations(nil, nil) != nil {
		t.Error("an agent without collaborators must not send multiAgentConfigurations")
	}
}

func TestAChangedCollaboratorRequiresANewVersion(t *testing.T) {
	state := []OrchestrationConfigModel{{
		ConnectInstanceArn: frameworktypes.StringValue("arn:aws:connect:eu-central-1:111122223333:instance/x"),
		Handoffs: []AgentHandoffModel{{
			ApplicationID:         frameworktypes.StringValue("app"),
			AudioStreamingEnabled: frameworktypes.BoolValue(false),
			ImmediateHandoff:      frameworktypes.BoolValue(true),
		}},
	}}
	plan := []OrchestrationConfigModel{state[0]}
	plan[0].Handoffs = []AgentHandoffModel{state[0].Handoffs[0]}
	plan[0].Handoffs[0].AudioStreamingEnabled = frameworktypes.BoolValue(true)

	if orchestrationConfigEqual(plan, state) {
		t.Error("turning on audio streaming must count as a change")
	}
}
