// Copyright tecRacer Group 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHoursOfOperationOverrideIDRoundTrips(t *testing.T) {
	t.Parallel()

	id := composeHoursOfOperationOverrideID("instance-1", "hours-1", "override-1")

	instanceID, hoursOfOperationID, overrideID, err := parseHoursOfOperationOverrideID(id)
	if err != nil {
		t.Fatalf("parseHoursOfOperationOverrideID(%q) error: %v", id, err)
	}
	if instanceID != "instance-1" || hoursOfOperationID != "hours-1" || overrideID != "override-1" {
		t.Fatalf("parseHoursOfOperationOverrideID(%q) = %q, %q, %q", id, instanceID, hoursOfOperationID, overrideID)
	}
}

func TestHoursOfOperationOverrideIDRejectsOtherShapes(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"", "instance-1/hours-1", "instance-1//override-1", "a/b/c/d"} {
		if _, _, _, err := parseHoursOfOperationOverrideID(id); err == nil {
			t.Errorf("parseHoursOfOperationOverrideID(%q) returned no error", id)
		}
	}
}

func TestAClosedRangeIsSentAsAnEmptyList(t *testing.T) {
	t.Parallel()

	days := overrideConfigToAPI(nil)

	if days == nil || len(days) != 0 {
		t.Fatalf("overrideConfigToAPI(nil) = %#v, want an empty, non-nil list", days)
	}
}

func TestAClosedRangeIsReadBackAsNoConfig(t *testing.T) {
	t.Parallel()

	if config := overrideConfigFromAPI([]types.HoursOfOperationOverrideConfig{}); config != nil {
		t.Fatalf("overrideConfigFromAPI(empty) = %#v, want nil", config)
	}
}

func TestOpeningHoursSurviveTheRoundTrip(t *testing.T) {
	t.Parallel()

	config := []OverrideOpeningHoursModel{
		{
			Day:       frameworktypes.StringValue("MONDAY"),
			StartTime: OverrideTimeModel{Hours: frameworktypes.Int32Value(8), Minutes: frameworktypes.Int32Value(30)},
			EndTime:   OverrideTimeModel{Hours: frameworktypes.Int32Value(12), Minutes: frameworktypes.Int32Value(0)},
		},
	}

	days := overrideConfigToAPI(config)
	if days[0].Day != types.OverrideDaysMonday || aws.ToInt32(days[0].StartTime.Hours) != 8 || aws.ToInt32(days[0].EndTime.Hours) != 12 {
		t.Fatalf("overrideConfigToAPI() = %#v", days[0])
	}

	back := overrideConfigFromAPI(days)
	if len(back) != 1 || back[0] != config[0] {
		t.Fatalf("overrideConfigFromAPI() = %#v, want %#v", back, config)
	}
}
