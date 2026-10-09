// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
)

// securityProfileState bundles every field UpdateSecurityProfile can set, so a resource that owns
// one entry of one list can re-supply everything else unchanged.
type securityProfileState struct {
	description                          *string
	allowedAccessControlHierarchyGroupID *string
	allowedAccessControlTags             map[string]string
	granularAccessControlConfiguration   *types.GranularAccessControlConfiguration
	hierarchyRestrictedResources         []string
	permissions                          []string
	applications                         []types.Application
	allowedFlowModules                   []types.FlowModule
	allowedAIAgents                      []types.AIAgent
}

// readSecurityProfileState fetches every field needed to safely round-trip an UpdateSecurityProfile
// call without clobbering fields owned by other resources (e.g. aws_connect_security_profile).
func readSecurityProfileState(ctx context.Context, client *connect.Client, instanceID, securityProfileID string) (*securityProfileState, error) {
	described, err := client.DescribeSecurityProfile(ctx, &connect.DescribeSecurityProfileInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	if err != nil {
		return nil, fmt.Errorf("describing security profile %s: %w", securityProfileID, err)
	}
	if described.SecurityProfile == nil {
		return nil, fmt.Errorf("DescribeSecurityProfile returned empty response for %s", securityProfileID)
	}
	sp := described.SecurityProfile

	state := &securityProfileState{
		description:                          sp.Description,
		allowedAccessControlHierarchyGroupID: sp.AllowedAccessControlHierarchyGroupId,
		allowedAccessControlTags:             sp.AllowedAccessControlTags,
		granularAccessControlConfiguration:   sp.GranularAccessControlConfiguration,
		hierarchyRestrictedResources:         sp.HierarchyRestrictedResources,
	}

	permPaginator := connect.NewListSecurityProfilePermissionsPaginator(client, &connect.ListSecurityProfilePermissionsInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for permPaginator.HasMorePages() {
		page, err := permPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile permissions: %w", err)
		}
		state.permissions = append(state.permissions, page.Permissions...)
	}

	appPaginator := connect.NewListSecurityProfileApplicationsPaginator(client, &connect.ListSecurityProfileApplicationsInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for appPaginator.HasMorePages() {
		page, err := appPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile applications: %w", err)
		}
		state.applications = append(state.applications, page.Applications...)
	}

	if state.allowedFlowModules, err = listAllowedFlowModules(ctx, client, instanceID, securityProfileID); err != nil {
		return nil, err
	}
	if state.allowedAIAgents, err = listAllowedAIAgents(ctx, client, instanceID, securityProfileID); err != nil {
		return nil, err
	}

	return state, nil
}

// writeSecurityProfileState calls UpdateSecurityProfile, re-supplying every field on sp so that
// only the caller's intended change takes effect.
func writeSecurityProfileState(ctx context.Context, client *connect.Client, instanceID, securityProfileID string, sp *securityProfileState) error {
	input := &connect.UpdateSecurityProfileInput{
		InstanceId:                           aws.String(instanceID),
		SecurityProfileId:                    aws.String(securityProfileID),
		Description:                          sp.description,
		AllowedAccessControlHierarchyGroupId: sp.allowedAccessControlHierarchyGroupID,
		AllowedAccessControlTags:             sp.allowedAccessControlTags,
		GranularAccessControlConfiguration:   sp.granularAccessControlConfiguration,
		HierarchyRestrictedResources:         sp.hierarchyRestrictedResources,
		Permissions:                          sp.permissions,
		Applications:                         sp.applications,
		AllowedFlowModules:                   sp.allowedFlowModules,
		AllowedAIAgents:                      sp.allowedAIAgents,
	}

	_, err := client.UpdateSecurityProfile(ctx, input)
	if err != nil {
		return fmt.Errorf("UpdateSecurityProfile failed: %w", err)
	}
	return nil
}

// listAllowedFlowModules returns the full, paginated AllowedFlowModules list for a security profile.
func listAllowedFlowModules(ctx context.Context, client *connect.Client, instanceID, securityProfileID string) ([]types.FlowModule, error) {
	var out []types.FlowModule
	paginator := connect.NewListSecurityProfileFlowModulesPaginator(client, &connect.ListSecurityProfileFlowModulesInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile flow modules: %w", err)
		}
		out = append(out, page.AllowedFlowModules...)
	}
	return out, nil
}

// listAllowedAIAgents returns the full, paginated AllowedAIAgents list for a security profile.
func listAllowedAIAgents(ctx context.Context, client *connect.Client, instanceID, securityProfileID string) ([]types.AIAgent, error) {
	var out []types.AIAgent
	paginator := connect.NewListSecurityProfileAIAgentsPaginator(client, &connect.ListSecurityProfileAIAgentsInput{
		InstanceId:        aws.String(instanceID),
		SecurityProfileId: aws.String(securityProfileID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing security profile AI agents: %w", err)
		}
		out = append(out, page.AllowedAIAgents...)
	}
	return out, nil
}
