# connectracer_connect_hours_of_operation_override

Replaces the weekly schedule of an hours of operation for a date range, such as a Christmas closure.

## Why not awscc

`awscc_connect_hours_of_operation` holds the overrides as a list on the hours of operation. A closed range is an override without any opening hours, `OverrideConfig: []`, and awscc drops the empty list before it calls Cloud Control, which then refuses the override for the missing key. The aws provider has no overrides at all. This resource calls `CreateHoursOfOperationOverride` directly and sends the empty list, so the hours of operation can stay on `aws_connect_hours_of_operation`.

## Closed or shorter hours

- Without `config` the hours of operation are closed for the whole range.
- With `config` only the listed days are open, with the given times.

The resource creates a one-off override (`STANDARD`). Connect accepts `CLOSED` only for a recurring override, which this resource does not support.

## In a flow

A *Check hours of operation* block with *Check override* enabled branches on `ClosedByOverride` and `OpenedByOverride`. `$.HoursOfOperation.AppliedOverride` then holds the override's `name`.

## Example Usage

```hcl
resource "connectracer_connect_hours_of_operation_override" "christmas" {
  instance_id           = var.instance_id
  hours_of_operation_id = aws_connect_hours_of_operation.support.hours_of_operation_id
  name                  = "Christmas"
  effective_from        = "2026-12-24"
  effective_till        = "2026-12-26"
}
```

## Import

```shell
terraform import connectracer_connect_hours_of_operation_override.christmas \
  "12345678-1234-1234-1234-123456789012/abcdef12-3456-7890-abcd-ef1234567890/fedcba98-7654-3210-fedc-ba9876543210"
```

The import ID is `instance_id/hours_of_operation_id/hours_of_operation_override_id`.
