// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// Handoffs come first, then delegates, each in configuration order. The API keeps
// both in one list, so a configuration mixing the two reads back in this order.
func expandMultiAgentConfigurations(handoffs []AgentHandoffModel, delegates []AgentDelegateModel) []types.MultiAgentConfiguration {
	if len(handoffs) == 0 && len(delegates) == 0 {
		return nil
	}

	configurations := make([]types.MultiAgentConfiguration, 0, len(handoffs)+len(delegates))
	for _, handoff := range handoffs {
		configurations = append(configurations, &types.MultiAgentConfigurationMemberHandoffAgentConfiguration{
			Value: types.HandoffAgentConfiguration{
				AgentTarget:           expandAgentTarget(handoff.ApplicationID, handoff.AIAgentID),
				Instruction:           expandMultiAgentInstruction(handoff.Instruction, handoff.Examples),
				AudioStreamingEnabled: aws.Bool(handoff.AudioStreamingEnabled.ValueBool()),
				ImmediateHandoff:      aws.Bool(handoff.ImmediateHandoff.ValueBool()),
			},
		})
	}
	for _, delegate := range delegates {
		configurations = append(configurations, &types.MultiAgentConfigurationMemberDelegateAgentConfiguration{
			Value: types.DelegateAgentConfiguration{
				AgentTarget: expandAgentTarget(delegate.ApplicationID, delegate.AIAgentID),
				Instruction: expandMultiAgentInstruction(delegate.Instruction, delegate.Examples),
			},
		})
	}
	return configurations
}

func flattenMultiAgentConfigurations(configurations []types.MultiAgentConfiguration) ([]AgentHandoffModel, []AgentDelegateModel) {
	var handoffs []AgentHandoffModel
	var delegates []AgentDelegateModel

	for _, configuration := range configurations {
		switch c := configuration.(type) {
		case *types.MultiAgentConfigurationMemberHandoffAgentConfiguration:
			applicationID, aiAgentID := flattenAgentTarget(c.Value.AgentTarget)
			instruction, examples := flattenMultiAgentInstruction(c.Value.Instruction)
			handoffs = append(handoffs, AgentHandoffModel{
				ApplicationID:         applicationID,
				AIAgentID:             aiAgentID,
				Instruction:           instruction,
				Examples:              examples,
				AudioStreamingEnabled: frameworktypes.BoolValue(aws.ToBool(c.Value.AudioStreamingEnabled)),
				ImmediateHandoff:      frameworktypes.BoolValue(aws.ToBool(c.Value.ImmediateHandoff)),
			})
		case *types.MultiAgentConfigurationMemberDelegateAgentConfiguration:
			applicationID, aiAgentID := flattenAgentTarget(c.Value.AgentTarget)
			instruction, examples := flattenMultiAgentInstruction(c.Value.Instruction)
			delegates = append(delegates, AgentDelegateModel{
				ApplicationID: applicationID,
				AIAgentID:     aiAgentID,
				Instruction:   instruction,
				Examples:      examples,
			})
		}
	}
	return handoffs, delegates
}

func expandAgentTarget(applicationID, aiAgentID frameworktypes.String) types.AgentTarget {
	if !applicationID.IsNull() && !applicationID.IsUnknown() {
		return &types.AgentTargetMemberApplicationId{Value: applicationID.ValueString()}
	}
	return &types.AgentTargetMemberAiAgentId{Value: aiAgentID.ValueString()}
}

func flattenAgentTarget(target types.AgentTarget) (applicationID, aiAgentID frameworktypes.String) {
	switch t := target.(type) {
	case *types.AgentTargetMemberApplicationId:
		return frameworktypes.StringValue(t.Value), frameworktypes.StringNull()
	case *types.AgentTargetMemberAiAgentId:
		return frameworktypes.StringNull(), frameworktypes.StringValue(t.Value)
	}
	return frameworktypes.StringNull(), frameworktypes.StringNull()
}

func expandMultiAgentInstruction(instruction frameworktypes.String, examples []frameworktypes.String) *types.MultiAgentInstruction {
	if instruction.IsNull() && len(examples) == 0 {
		return nil
	}
	expanded := &types.MultiAgentInstruction{}
	if !instruction.IsNull() && !instruction.IsUnknown() {
		expanded.Instruction = aws.String(instruction.ValueString())
	}
	for _, example := range examples {
		expanded.Examples = append(expanded.Examples, example.ValueString())
	}
	return expanded
}

func flattenMultiAgentInstruction(instruction *types.MultiAgentInstruction) (frameworktypes.String, []frameworktypes.String) {
	if instruction == nil {
		return frameworktypes.StringNull(), nil
	}
	var examples []frameworktypes.String
	for _, example := range instruction.Examples {
		examples = append(examples, frameworktypes.StringValue(example))
	}
	return frameworktypes.StringPointerValue(instruction.Instruction), examples
}

func handoffsEqual(plan, state []AgentHandoffModel) bool {
	if len(plan) != len(state) {
		return false
	}
	for i := range plan {
		p, s := plan[i], state[i]
		if !p.ApplicationID.Equal(s.ApplicationID) || !p.AIAgentID.Equal(s.AIAgentID) ||
			!p.Instruction.Equal(s.Instruction) || !stringsEqual(p.Examples, s.Examples) ||
			!p.AudioStreamingEnabled.Equal(s.AudioStreamingEnabled) || !p.ImmediateHandoff.Equal(s.ImmediateHandoff) {
			return false
		}
	}
	return true
}

func delegatesEqual(plan, state []AgentDelegateModel) bool {
	if len(plan) != len(state) {
		return false
	}
	for i := range plan {
		p, s := plan[i], state[i]
		if !p.ApplicationID.Equal(s.ApplicationID) || !p.AIAgentID.Equal(s.AIAgentID) ||
			!p.Instruction.Equal(s.Instruction) || !stringsEqual(p.Examples, s.Examples) {
			return false
		}
	}
	return true
}

func stringsEqual(plan, state []frameworktypes.String) bool {
	if len(plan) != len(state) {
		return false
	}
	for i := range plan {
		if !plan[i].Equal(state[i]) {
			return false
		}
	}
	return true
}
