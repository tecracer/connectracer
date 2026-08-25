// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect/types"
)

func TestSanitizePreservedTools(t *testing.T) {
	t.Parallel()

	mcp := types.ToolConfiguration{
		ToolName:    aws.String("CheckAgentAvailability"),
		ToolType:    types.ToolTypeModelContextProtocol,
		ToolId:      aws.String("aws_custom_flows__abc_1"),
		Title:       aws.String("Check agent availability"),
		Description: aws.String("Checks whether a human agent is available right now"),
		Instruction: &types.ToolInstruction{Instruction: aws.String("Always call this before ...")},
	}

	t.Run("keeps the instruction but not gateway-owned fields on an MCP tool", func(t *testing.T) {
		t.Parallel()
		got := sanitizePreservedTools([]types.ToolConfiguration{mcp})
		if len(got) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(got))
		}
		if got[0].Instruction == nil || aws.ToString(got[0].Instruction.Instruction) != "Always call this before ..." {
			t.Errorf("instruction not preserved: %+v", got[0].Instruction)
		}
		// Description and Title are gateway-owned and must NOT be echoed back:
		// preserving them pins a value AWS overwrites, producing a permanent plan diff.
		if got[0].Description != nil {
			t.Errorf("description must be stripped, got %q", aws.ToString(got[0].Description))
		}
		if got[0].Title != nil {
			t.Errorf("title must be stripped, got %q", aws.ToString(got[0].Title))
		}
		if aws.ToString(got[0].ToolId) != aws.ToString(mcp.ToolId) {
			t.Errorf("tool id not preserved: %q", aws.ToString(got[0].ToolId))
		}
	})

	t.Run("still strips server-owned fields on an MCP tool", func(t *testing.T) {
		t.Parallel()
		withSchema := mcp
		withSchema.OutputFilters = []types.ToolOutputFilter{{}}
		got := sanitizePreservedTools([]types.ToolConfiguration{withSchema})
		if got[0].OutputFilters != nil {
			t.Errorf("output filters should be stripped, got %+v", got[0].OutputFilters)
		}
	})

	t.Run("leaves a non-MCP tool untouched", func(t *testing.T) {
		t.Parallel()
		rtc := types.ToolConfiguration{
			ToolName:      aws.String("Escalate"),
			ToolType:      types.ToolTypeReturnToControl,
			Description:   aws.String("Transfer to a human"),
			Instruction:   &types.ToolInstruction{Instruction: aws.String("Call this when ...")},
			OutputFilters: []types.ToolOutputFilter{{}},
		}
		got := sanitizePreservedTools([]types.ToolConfiguration{rtc})
		if got[0].OutputFilters == nil {
			t.Error("non-MCP tool must be passed through unchanged")
		}
		if aws.ToString(got[0].Description) != "Transfer to a human" {
			t.Error("non-MCP description must survive")
		}
	})

	t.Run("empty input yields nil", func(t *testing.T) {
		t.Parallel()
		if sanitizePreservedTools(nil) != nil {
			t.Error("expected nil for empty input")
		}
	})
}
