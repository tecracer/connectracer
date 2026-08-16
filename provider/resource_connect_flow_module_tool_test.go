// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"reflect"
	"sort"
	"testing"
)

func TestMcpToolID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flowModuleID string
		version      int64
		want         string
	}{
		{
			name:         "typical id and first version",
			flowModuleID: "a72275dc-a520-4fa1-8559-851ee9739e42",
			version:      1,
			want:         "aws_custom_flows__a72275dc-a520-4fa1-8559-851ee9739e42_1",
		},
		{
			name:         "later version",
			flowModuleID: "c8aa42b1-38d1-4558-8762-717fa20c0439",
			version:      2,
			want:         "aws_custom_flows__c8aa42b1-38d1-4558-8762-717fa20c0439_2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := mcpToolID(tc.flowModuleID, tc.version); got != tc.want {
				t.Fatalf("mcpToolID(%q, %d) = %q, want %q", tc.flowModuleID, tc.version, got, tc.want)
			}
		})
	}
}

func TestDiffTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		oldTags    map[string]string
		newTags    map[string]string
		wantAdd    map[string]string
		wantRemove []string
	}{
		{
			name:       "no change",
			oldTags:    map[string]string{"env": "dev"},
			newTags:    map[string]string{"env": "dev"},
			wantAdd:    map[string]string{},
			wantRemove: nil,
		},
		{
			name:       "add new key",
			oldTags:    map[string]string{},
			newTags:    map[string]string{"env": "dev"},
			wantAdd:    map[string]string{"env": "dev"},
			wantRemove: nil,
		},
		{
			name:       "update changed value",
			oldTags:    map[string]string{"env": "dev"},
			newTags:    map[string]string{"env": "prod"},
			wantAdd:    map[string]string{"env": "prod"},
			wantRemove: nil,
		},
		{
			name:       "remove missing key",
			oldTags:    map[string]string{"env": "dev", "team": "cx"},
			newTags:    map[string]string{"env": "dev"},
			wantAdd:    map[string]string{},
			wantRemove: []string{"team"},
		},
		{
			name:       "add, update, and remove together",
			oldTags:    map[string]string{"keep": "same", "update": "old", "drop": "x"},
			newTags:    map[string]string{"keep": "same", "update": "new", "fresh": "y"},
			wantAdd:    map[string]string{"update": "new", "fresh": "y"},
			wantRemove: []string{"drop"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			add, remove := diffTags(tc.oldTags, tc.newTags)

			if !reflect.DeepEqual(add, tc.wantAdd) {
				t.Fatalf("diffTags() add = %v, want %v", add, tc.wantAdd)
			}

			sort.Strings(remove)
			sort.Strings(tc.wantRemove)
			if !reflect.DeepEqual(remove, tc.wantRemove) {
				t.Fatalf("diffTags() remove = %v, want %v", remove, tc.wantRemove)
			}
		})
	}
}
