# connectracer_connect_data_table_record

A record of an Amazon Connect data table: its primary values and the values of its other attributes.

## Why not awscc

`awscc_connect_data_table_record` goes through Cloud Control, whose handler writes the values of a record one at a time. Every value carries a lock version, and on a table with `value_lock_level = "PRIMARY_VALUE"` the first write bumps the lock the second one still sends, so the second fails with *Provided lock does not match expected lock*. `VALUE` lets a record be created, but changing one still fails the same way. This resource reads the lock version of every value right before it writes and sends all values of the record in one `BatchUpdateDataTableValue` call.

## Values

- Attributes are addressed by name, in `primary_values` and in `values`.
- Every value is text, whatever the attribute's type: `"true"` for a boolean, `"42"` for a number.
- Other `primary_values` make another record, so a change replaces it.
- An attribute dropped from `values` is deleted from the record. An attribute the configuration never had, e.g. one set in the admin website, is left alone.
- A value changed outside Terraform is overwritten on the next apply.

## Values edited in the admin website

When the values belong to the admin website and Terraform should only seed them, ignore them after the create:

```hcl
resource "connectracer_connect_data_table_record" "support_closed" {
  # ...

  lifecycle {
    ignore_changes = [values]
  }
}
```

Terraform then creates the record with the configured values and leaves later edits alone. A changed value in the configuration only reaches a record Terraform creates.

## Taking over existing records

A record whose primary values exist already, e.g. one created by `awscc_connect_data_table_record`, is taken over on create rather than refused, and only the values that differ are written. Moving from awscc therefore needs no import:

```hcl
removed {
  from = awscc_connect_data_table_record.closed_message

  lifecycle {
    destroy = false
  }
}
```

## Parallel applies

Connect refuses concurrent writes to the same attribute, even across records, with *Concurrency conflict while updating Attribute*. Terraform writes records in parallel, so the resource reads the record again and retries a conflicting write, with a growing, jittered wait.

## Example Usage

```hcl
resource "connectracer_connect_data_table_record" "support_closed" {
  instance_id   = var.instance_id
  data_table_id = "abcdef12-3456-7890-abcd-ef1234567890"

  primary_values = {
    hours    = "support"
    override = "*"
  }

  values = {
    active  = "true"
    message = "We are closed right now. Please call again during our opening hours."
  }
}
```

## Import

```shell
terraform import connectracer_connect_data_table_record.support_closed \
  "12345678-1234-1234-1234-123456789012/abcdef12-3456-7890-abcd-ef1234567890/fedcba98-7654-3210-fedc-ba9876543210"
```

The import ID is `instance_id/data_table_id/record_id`. An imported record takes every value it holds, so `values` has to list all of them, or the next apply deletes the ones left out.
