// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveVersionNumber(t *testing.T) {
	t.Parallel()

	unknown := frameworktypes.Int64Unknown()
	null := frameworktypes.Int64Null()
	known := frameworktypes.Int64Value(7)
	prior := frameworktypes.Int64Value(3)

	tests := []struct {
		name    string
		current frameworktypes.Int64
		prior   frameworktypes.Int64
		want    frameworktypes.Int64
	}{
		{"create_version false, no prior version", unknown, null, null},
		{"create_version false, keeps prior version", unknown, prior, prior},
		{"create_version true, keeps the new version", known, prior, known},
		{"null current falls back to prior", null, prior, prior},
		{"nothing known at all", null, null, null},
		{"unknown prior is not carried over", unknown, unknown, null},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveVersionNumber(tt.current, tt.prior)
			if !got.Equal(tt.want) {
				t.Fatalf("resolveVersionNumber(%v, %v) = %v, want %v", tt.current, tt.prior, got, tt.want)
			}
			if got.IsUnknown() {
				t.Fatal("result must never be unknown: Terraform rejects that after apply")
			}
		})
	}
}
