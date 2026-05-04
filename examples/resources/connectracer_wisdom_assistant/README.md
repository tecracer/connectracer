# AWS Wisdom Assistant Resource

This resource allows you to create and manage AWS Wisdom assistants.

## Example Usage

### Basic Assistant

```hcl
resource "connectracer_wisdom_assistant" "example" {
  name        = "my-assistant"
  type        = "AGENT"
  description = "My Wisdom Assistant for customer support"

  tags = {
    Environment = "production"
    Team        = "customer-support"
  }
}
```

### Assistant with KMS Encryption

```hcl
resource "connectracer_wisdom_assistant" "encrypted" {
  name        = "encrypted-assistant"
  type        = "AGENT"
  description = "Encrypted Wisdom Assistant"

  server_side_encryption_configuration {
    kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  }

  tags = {
    Environment = "production"
    Encrypted   = "true"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the assistant.
* `type` - (Required) The type of assistant. Valid values: `AGENT`. Changing this forces a new resource to be created.
* `description` - (Optional) The description of the assistant.
* `tags` - (Optional) A map of tags to assign to the resource.
* `server_side_encryption_configuration` - (Optional) The KMS key used for encryption. See below.

### server_side_encryption_configuration

The `server_side_encryption_configuration` block supports:

* `kms_key_id` - (Optional) The KMS key ID or ARN. The customer managed key must have a policy that allows `kms:CreateGrant`, `kms:DescribeKey`, and `kms:Decrypt/kms:GenerateDataKey` permissions to the IAM identity using the key to invoke Wisdom.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the assistant.
* `assistant_arn` - The ARN of the assistant.
* `status` - The status of the assistant (e.g., `ACTIVE`).

## Import

Wisdom assistants can be imported using the assistant ID, e.g.,

```shell
terraform import connectracer_wisdom_assistant.example bc52879e-19e9-4ec4-ba5c-11f95ad7bb9f
```

## Update Limitations

**Important:** AWS Wisdom does not provide an API to update assistant properties after creation. The following limitations apply:

* **Tags only**: Only tags can be updated in-place using the `TagResource` and `UntagResource` APIs.
* **Name and Description**: Changing the `name` or `description` will trigger a warning but will NOT recreate the resource automatically. You must manually recreate the resource if you need to change these fields.
* **Type**: Changing the `type` will force a recreation of the resource.
* **Encryption Configuration**: Cannot be changed after creation. Changing this requires recreating the resource.

To work around these limitations for fields that cannot be updated:

1. Use `terraform taint` to mark the resource for recreation, or
2. Manually delete and recreate the resource, or
3. Set `create_before_destroy` lifecycle rule if you need zero downtime.

Example with lifecycle rule:

```hcl
resource "connectracer_wisdom_assistant" "example" {
  name        = "my-assistant"
  type        = "AGENT"
  description = "Updated description"

  lifecycle {
    create_before_destroy = true
  }
}
```
