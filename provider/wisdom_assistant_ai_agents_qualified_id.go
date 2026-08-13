// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/qconnect"
)

type qualifiedAIAgentID struct {
	agentID   string
	qualifier string
}

func parseQualifiedAIAgentID(value string) qualifiedAIAgentID {
	idx := strings.LastIndex(value, ":")
	if idx == -1 {
		return qualifiedAIAgentID{agentID: value, qualifier: ""}
	}
	return qualifiedAIAgentID{
		agentID:   value[:idx],
		qualifier: value[idx+1:],
	}
}

func isLatestQualifier(qualifier string) bool {
	switch strings.ToUpper(strings.TrimSpace(qualifier)) {
	case "", "$LATEST", "LATEST":
		return true
	default:
		return false
	}
}

func latestAIAgentVersion(ctx context.Context, client *qconnect.Client, assistantID, agentID string) (int64, error) {
	output, err := client.GetAIAgent(ctx, &qconnect.GetAIAgentInput{
		AssistantId: aws.String(assistantID),
		AiAgentId:   aws.String(agentID),
	})
	if err != nil {
		return 0, err
	}
	if output.VersionNumber == nil {
		return 0, fmt.Errorf("GetAIAgent returned no version number for agent %q", agentID)
	}
	return *output.VersionNumber, nil
}

func resolveQualifiedAIAgentVersion(ctx context.Context, client *qconnect.Client, assistantID string, parsed qualifiedAIAgentID) (int64, error) {
	if isLatestQualifier(parsed.qualifier) {
		return latestAIAgentVersion(ctx, client, assistantID, parsed.agentID)
	}
	version, err := strconv.ParseInt(parsed.qualifier, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid AI agent version qualifier %q: %w", parsed.qualifier, err)
	}
	return version, nil
}

// qualifiedAIAgentIDsSemanticallyEqual reports whether two qualified AI agent IDs refer to
// the same agent version, treating $LATEST as the current published version from GetAIAgent.
func qualifiedAIAgentIDsSemanticallyEqual(ctx context.Context, client *qconnect.Client, assistantID, configured, awsValue string) (bool, error) {
	configuredParsed := parseQualifiedAIAgentID(configured)
	awsParsed := parseQualifiedAIAgentID(awsValue)

	if configuredParsed.agentID != awsParsed.agentID {
		return false, nil
	}
	if configured == awsValue {
		return true, nil
	}

	configuredVersion, err := resolveQualifiedAIAgentVersion(ctx, client, assistantID, configuredParsed)
	if err != nil {
		return false, err
	}
	awsVersion, err := resolveQualifiedAIAgentVersion(ctx, client, assistantID, awsParsed)
	if err != nil {
		return false, err
	}
	return configuredVersion == awsVersion, nil
}
