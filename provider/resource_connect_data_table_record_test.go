// Copyright tecRacer Group 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDataTableRecordIDRoundTrips(t *testing.T) {
	t.Parallel()

	id := composeDataTableRecordID("instance-1", "table-1", "record-1")

	instanceID, dataTableID, recordID, err := parseDataTableRecordID(id)
	if err != nil {
		t.Fatalf("parseDataTableRecordID(%q) error: %v", id, err)
	}
	if instanceID != "instance-1" || dataTableID != "table-1" || recordID != "record-1" {
		t.Fatalf("parseDataTableRecordID(%q) = %q, %q, %q", id, instanceID, dataTableID, recordID)
	}
}

func TestDataTableRecordIDRejectsOtherShapes(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"", "instance-1/table-1", "instance-1//record-1", "a/b/c/d"} {
		if _, _, _, err := parseDataTableRecordID(id); err == nil {
			t.Errorf("parseDataTableRecordID(%q) returned no error", id)
		}
	}
}

func TestEveryValueIsWrittenWithTheLockConnectHolds(t *testing.T) {
	t.Parallel()

	activeLock := &types.DataTableLockVersion{Value: aws.String("lock-active")}
	messageLock := &types.DataTableLockVersion{Value: aws.String("lock-message")}
	primaryValues := dataTablePrimaryValuesToAPI(map[string]frameworktypes.String{
		"override": frameworktypes.StringValue("*"),
		"hours":    frameworktypes.StringValue("aps"),
	})

	values := dataTableValuesToAPI(map[string]frameworktypes.String{
		"message": frameworktypes.StringValue("Closed"),
		"active":  frameworktypes.StringValue("true"),
	}, primaryValues, map[string]*types.DataTableLockVersion{"active": activeLock, "message": messageLock})

	if len(values) != 2 {
		t.Fatalf("dataTableValuesToAPI() returned %d values, want 2", len(values))
	}
	if aws.ToString(values[0].AttributeName) != "active" || values[0].LockVersion != activeLock {
		t.Errorf("values[0] = %s with lock %v, want active with its own lock", aws.ToString(values[0].AttributeName), values[0].LockVersion)
	}
	if aws.ToString(values[1].AttributeName) != "message" || values[1].LockVersion != messageLock {
		t.Errorf("values[1] = %s with lock %v, want message with its own lock", aws.ToString(values[1].AttributeName), values[1].LockVersion)
	}
	if aws.ToString(values[0].PrimaryValues[0].AttributeName) != "hours" {
		t.Errorf("primary values are not sorted by name: %#v", values[0].PrimaryValues)
	}
}

func held(values map[string]string) []types.DataTableValueSummary {
	record := make([]types.DataTableValueSummary, 0, len(values))
	for name, value := range values {
		record = append(record, types.DataTableValueSummary{AttributeName: aws.String(name), Value: aws.String(value)})
	}
	return record
}

func TestOnlyValuesThatDifferFromConnectAreUpdated(t *testing.T) {
	t.Parallel()

	plan := map[string]frameworktypes.String{
		"active":  frameworktypes.StringValue("true"),
		"message": frameworktypes.StringValue("New"),
	}

	changes := diffDataTableValues(nil, plan, held(map[string]string{"active": "true", "message": "Old"}))

	if len(changes.update) != 1 || changes.update["message"].ValueString() != "New" {
		t.Errorf("update = %#v, want only message", changes.update)
	}
	if len(changes.create) != 0 || len(changes.remove) != 0 {
		t.Errorf("create = %#v, remove = %#v, want neither", changes.create, changes.remove)
	}
}

func TestTakingOverAMatchingRecordWritesNothing(t *testing.T) {
	t.Parallel()

	plan := map[string]frameworktypes.String{
		"active":  frameworktypes.StringValue("true"),
		"message": frameworktypes.StringValue("Closed"),
	}

	changes := diffDataTableValues(nil, plan, held(map[string]string{"active": "true", "message": "Closed"}))

	if len(changes.update) != 0 || len(changes.create) != 0 || len(changes.remove) != 0 {
		t.Errorf("changes = %#v, want none", changes)
	}
}

func TestAValueConnectDoesNotHoldIsCreated(t *testing.T) {
	t.Parallel()

	plan := map[string]frameworktypes.String{"message": frameworktypes.StringValue("Closed")}

	changes := diffDataTableValues(plan, plan, nil)

	if len(changes.create) != 1 || len(changes.update) != 0 {
		t.Errorf("create = %#v, update = %#v, want message created", changes.create, changes.update)
	}
}

func TestADroppedValueIsDeletedOnlyWhileConnectHoldsIt(t *testing.T) {
	t.Parallel()

	previous := map[string]frameworktypes.String{
		"active":  frameworktypes.StringValue("true"),
		"message": frameworktypes.StringValue("Closed"),
		"note":    frameworktypes.StringValue("gone already"),
	}
	plan := map[string]frameworktypes.String{"message": frameworktypes.StringValue("Closed")}

	changes := diffDataTableValues(previous, plan, held(map[string]string{"active": "true", "message": "Closed"}))

	if !reflect.DeepEqual(changes.remove, []string{"active"}) {
		t.Errorf("remove = %#v, want [active]", changes.remove)
	}
}

func TestConflictsAreRetriedUntilTheWriteSucceeds(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retryDataTableConflicts(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("active: Concurrency conflict while updating Attribute")
		}
		return nil
	})

	if err != nil || calls != 3 {
		t.Errorf("retryDataTableConflicts() = %v after %d calls, want success after 3", err, calls)
	}
}

func TestOtherErrorsAreNotRetried(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retryDataTableConflicts(context.Background(), func() error {
		calls++
		return errors.New("AccessDeniedException")
	})

	if err == nil || calls != 1 {
		t.Errorf("retryDataTableConflicts() = %v after %d calls, want the error after 1", err, calls)
	}
}

func TestBothLockErrorsCountAsConflicts(t *testing.T) {
	t.Parallel()

	for _, message := range []string{
		"active: Concurrency conflict while updating Attribute",
		"Provided lock does not match expected lock ea027232",
	} {
		if !isDataTableConflict(errors.New(message)) {
			t.Errorf("isDataTableConflict(%q) = false", message)
		}
	}
}

func TestReadKeepsOnlyManagedValues(t *testing.T) {
	t.Parallel()

	record := []types.DataTableValueSummary{
		{AttributeName: aws.String("message"), Value: aws.String("Closed")},
		{AttributeName: aws.String("note"), Value: aws.String("added in the console")},
	}
	managed := map[string]frameworktypes.String{"message": frameworktypes.StringValue("Old")}

	values := dataTableValuesFromAPI(record, managed)

	if len(values) != 1 || values["message"].ValueString() != "Closed" {
		t.Errorf("dataTableValuesFromAPI() = %#v, want only message", values)
	}
}

func TestAnImportTakesEveryValue(t *testing.T) {
	t.Parallel()

	record := []types.DataTableValueSummary{
		{AttributeName: aws.String("active"), Value: aws.String("true")},
		{AttributeName: aws.String("message"), Value: aws.String("Closed")},
	}

	if values := dataTableValuesFromAPI(record, nil); len(values) != 2 {
		t.Errorf("dataTableValuesFromAPI(nil) = %#v, want both values", values)
	}
}

func TestPrimaryValuesAreReadFromTheRecord(t *testing.T) {
	t.Parallel()

	record := []types.DataTableValueSummary{{
		AttributeName: aws.String("message"),
		PrimaryValues: []types.PrimaryValueResponse{
			{AttributeName: aws.String("hours"), Value: aws.String("aps")},
			{AttributeName: aws.String("override"), Value: aws.String("*")},
		},
	}}

	primary := dataTablePrimaryValuesFromAPI(record)

	if primary["hours"].ValueString() != "aps" || primary["override"].ValueString() != "*" {
		t.Errorf("dataTablePrimaryValuesFromAPI() = %#v", primary)
	}
}

func TestFailedValuesAreReportedByAttribute(t *testing.T) {
	t.Parallel()

	failed := []types.BatchUpdateDataTableValueFailureResult{
		{AttributeName: aws.String("message"), Message: aws.String("Provided lock does not match expected lock")},
	}

	if got := dataTableFailures(failed); got != "message: Provided lock does not match expected lock" {
		t.Errorf("dataTableFailures() = %q", got)
	}
	if got := dataTableFailures([]types.BatchUpdateDataTableValueFailureResult{}); got != "" {
		t.Errorf("dataTableFailures(empty) = %q, want empty", got)
	}
}
