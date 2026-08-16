// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
)

func TestConnectSecurityProfileFlowModuleComposeID(t *testing.T) {
	t.Parallel()

	r := &ConnectSecurityProfileFlowModuleResource{}
	got := r.composeID("instance-1", "sp-1", "module-1")
	want := "instance-1/sp-1/module-1"
	if got != want {
		t.Fatalf("composeID() = %q, want %q", got, want)
	}
}

func TestConnectSecurityProfileFlowModuleFindFlowModule(t *testing.T) {
	t.Parallel()

	r := &ConnectSecurityProfileFlowModuleResource{}
	modules := []types.FlowModule{
		{FlowModuleId: aws.String("module-1"), Type: types.FlowModuleTypeMcp},
		{FlowModuleId: aws.String("module-2"), Type: types.FlowModuleTypeMcp},
	}

	found := r.findFlowModule(modules, "module-2")
	if found == nil {
		t.Fatal("findFlowModule() = nil, want module-2")
	}
	if aws.ToString(found.FlowModuleId) != "module-2" {
		t.Fatalf("findFlowModule() FlowModuleId = %q, want module-2", aws.ToString(found.FlowModuleId))
	}

	notFound := r.findFlowModule(modules, "module-missing")
	if notFound != nil {
		t.Fatalf("findFlowModule() = %v, want nil", notFound)
	}

	empty := r.findFlowModule(nil, "module-1")
	if empty != nil {
		t.Fatalf("findFlowModule() on nil slice = %v, want nil", empty)
	}
}
