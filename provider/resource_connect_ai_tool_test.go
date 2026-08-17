// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveOmittedToolFields(t *testing.T) {
	t.Parallel()

	strList := func(values ...string) frameworktypes.List {
		list, diags := frameworktypes.ListValueFrom(t.Context(), frameworktypes.StringType, values)
		if diags.HasError() {
			t.Fatalf("failed to build test list: %v", diags)
		}
		return list
	}
	nullList := frameworktypes.ListNull(frameworktypes.StringType)

	tests := []struct {
		name                     string
		awsHasDescription        bool
		freshDescription         frameworktypes.String
		priorDescription         frameworktypes.String
		awsHasInstruction        bool
		freshInstruction         frameworktypes.String
		priorInstruction         frameworktypes.String
		freshInstructionExamples frameworktypes.List
		priorInstructionExamples frameworktypes.List
		wantDescription          frameworktypes.String
		wantInstruction          frameworktypes.String
		wantInstructionExamples  frameworktypes.List
	}{
		{
			name:                     "AWS returns both fields: use fresh values",
			awsHasDescription:        true,
			freshDescription:         frameworktypes.StringValue("new description"),
			priorDescription:         frameworktypes.StringValue("old description"),
			awsHasInstruction:        true,
			freshInstruction:         frameworktypes.StringValue("new instruction"),
			priorInstruction:         frameworktypes.StringValue("old instruction"),
			freshInstructionExamples: strList("new example"),
			priorInstructionExamples: strList("old example"),
			wantDescription:          frameworktypes.StringValue("new description"),
			wantInstruction:          frameworktypes.StringValue("new instruction"),
			wantInstructionExamples:  strList("new example"),
		},
		{
			name:                     "AWS omits both fields but a prior value exists: fall back to prior",
			awsHasDescription:        false,
			freshDescription:         frameworktypes.StringNull(),
			priorDescription:         frameworktypes.StringValue("old description"),
			awsHasInstruction:        false,
			freshInstruction:         frameworktypes.StringNull(),
			priorInstruction:         frameworktypes.StringValue("old instruction"),
			freshInstructionExamples: nullList,
			priorInstructionExamples: strList("old example"),
			wantDescription:          frameworktypes.StringValue("old description"),
			wantInstruction:          frameworktypes.StringValue("old instruction"),
			wantInstructionExamples:  strList("old example"),
		},
		{
			name:                     "AWS omits both fields and there was never a prior value: stays null",
			awsHasDescription:        false,
			freshDescription:         frameworktypes.StringNull(),
			priorDescription:         frameworktypes.StringNull(),
			awsHasInstruction:        false,
			freshInstruction:         frameworktypes.StringNull(),
			priorInstruction:         frameworktypes.StringNull(),
			freshInstructionExamples: nullList,
			priorInstructionExamples: nullList,
			wantDescription:          frameworktypes.StringNull(),
			wantInstruction:          frameworktypes.StringNull(),
			wantInstructionExamples:  nullList,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotDescription, gotInstruction, gotInstructionExamples := resolveOmittedToolFields(
				tc.awsHasDescription, tc.freshDescription, tc.priorDescription,
				tc.awsHasInstruction, tc.freshInstruction, tc.priorInstruction,
				tc.freshInstructionExamples, tc.priorInstructionExamples,
			)

			if !gotDescription.Equal(tc.wantDescription) {
				t.Errorf("description = %v, want %v", gotDescription, tc.wantDescription)
			}
			if !gotInstruction.Equal(tc.wantInstruction) {
				t.Errorf("instruction = %v, want %v", gotInstruction, tc.wantInstruction)
			}
			if !gotInstructionExamples.Equal(tc.wantInstructionExamples) {
				t.Errorf("instructionExamples = %v, want %v", gotInstructionExamples, tc.wantInstructionExamples)
			}
		})
	}
}

func TestResolveOmittedStringField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		awsHasField bool
		fresh       frameworktypes.String
		prior       frameworktypes.String
		want        frameworktypes.String
	}{
		{
			name:        "AWS returns the field: use the fresh value",
			awsHasField: true,
			fresh:       frameworktypes.StringValue("new schema"),
			prior:       frameworktypes.StringValue("old schema"),
			want:        frameworktypes.StringValue("new schema"),
		},
		{
			name:        "AWS omits the field but a prior value exists: fall back to prior",
			awsHasField: false,
			fresh:       frameworktypes.StringNull(),
			prior:       frameworktypes.StringValue("old schema"),
			want:        frameworktypes.StringValue("old schema"),
		},
		{
			name:        "AWS omits the field and there was never a prior value: stays null",
			awsHasField: false,
			fresh:       frameworktypes.StringNull(),
			prior:       frameworktypes.StringNull(),
			want:        frameworktypes.StringNull(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveOmittedStringField(tc.awsHasField, tc.fresh, tc.prior)
			if !got.Equal(tc.want) {
				t.Errorf("resolveOmittedStringField() = %v, want %v", got, tc.want)
			}
		})
	}
}
