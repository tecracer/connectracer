# connectracer_connect_integration_association

Manages an AWS Connect Integration Association resource.

Creates an integration association between an Amazon Connect instance and external services such as Wisdom knowledge bases, Wisdom assistants, or other supported integrations.

## Example Usage

### Basic Wisdom Knowledge Base Integration

```terraform
resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "my-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "Knowledge base for customer support"
}

resource "connectracer_connect_integration_association" "kb" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c"
  integration_type = "WISDOM_KNOWLEDGE_BASE"
  integration_arn  = connectracer_qconnect_knowledgebase.example.knowledge_base_arn

  tags = {
    Environment = "production"
  }
}
```

### Wisdom Assistant Integration

```terraform
resource "connectracer_wisdom_assistant" "example" {
  name        = "my-assistant"
  type        = "AGENT"
  description = "Wisdom Assistant for agent assistance"
}

resource "connectracer_connect_integration_association" "assistant" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c"
  integration_type = "WISDOM_ASSISTANT"
  integration_arn  = connectracer_wisdom_assistant.example.assistant_arn

  tags = {
    Environment = "production"
  }
}
```

### Complete Wisdom Setup

See the [complete_wisdom_integration.tf](../../examples/resources/connectracer_connect_integration_association/complete_wisdom_integration.tf) example for a full setup including assistant, knowledge base, and all necessary associations.

## Argument Reference

The following arguments are required:

* `instance_id` - (Required, Forces new resource) The identifier of the Amazon Connect instance.
* `integration_type` - (Required, Forces new resource) The type of integration. Valid values:
  - `WISDOM_ASSISTANT` - Associate a Wisdom assistant
  - `WISDOM_KNOWLEDGE_BASE` - Associate a Wisdom knowledge base
  - `CASES_DOMAIN` - Associate a Cases domain
  - `APPLICATION` - Associate an external application
  - `FILE_SCANNER` - Associate a file scanner
* `integration_arn` - (Required, Forces new resource) The Amazon Resource Name (ARN) of the integration. For `WISDOM_ASSISTANT` or `WISDOM_KNOWLEDGE_BASE`, this should be the ARN of the Wisdom assistant or knowledge base.

The following arguments are optional:

* `source_application_url` - (Optional, Forces new resource) The URL for the external application. Required when `integration_type` is `APPLICATION`.
* `source_application_name` - (Optional, Forces new resource) The name of the external application. Required when `integration_type` is `APPLICATION`.
* `source_type` - (Optional, Forces new resource, Computed) The type of the data source. Valid values: `SALESFORCE`, `ZENDESK`, `CASES`.
* `tags` - (Optional) A map of tags to assign to the integration association.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The identifier for the integration association (IntegrationAssociationId).
* `integration_association_arn` - The Amazon Resource Name (ARN) of the integration association.

## Import

Connect Integration Association can be imported using a composite ID of `instance_id/integration_association_id`, e.g.,

```
$ terraform import connectracer_connect_integration_association.example 12eac4f3-e7d2-47f2-a2ac-f10753b6146c/b5a204fa-2bf9-410a-8142-64c8de1239ad
```

## Notes

### Immutable Fields

Most fields are immutable and require resource replacement when changed:
- `instance_id`
- `integration_type`
- `integration_arn`
- `source_application_url`
- `source_application_name`
- `source_type`

Only `tags` can be updated in-place.

### Integration Types

Different integration types require different configurations:

- **WISDOM_ASSISTANT**: Requires an ARN from a `connectracer_wisdom_assistant` resource
- **WISDOM_KNOWLEDGE_BASE**: Requires an ARN from a `connectracer_qconnect_knowledgebase` resource
- **APPLICATION**: Requires `source_application_url` and `source_application_name`
- **CASES_DOMAIN**: Requires an ARN from an AWS Cases domain
- **FILE_SCANNER**: Requires an ARN from a file scanner integration

### Prerequisites

Before creating a Connect integration association, ensure:

1. The Connect instance exists and is in an ACTIVE state
2. The integration resource (Wisdom assistant, knowledge base, etc.) is created
3. Appropriate IAM permissions are configured for Connect to access the integration

### IAM Permissions

The Connect instance requires appropriate IAM permissions to access the integrated service. For Wisdom integrations, ensure the Connect service role has permissions like:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "wisdom:GetAssistant",
        "wisdom:GetKnowledgeBase",
        "wisdom:GetRecommendations",
        "wisdom:QueryAssistant",
        "wisdom:NotifyRecommendationsReceived"
      ],
      "Resource": "*"
    }
  ]
}
```

## API Documentation

- [CreateIntegrationAssociation](https://docs.aws.amazon.com/connect/latest/APIReference/API_CreateIntegrationAssociation.html)
- [DeleteIntegrationAssociation](https://docs.aws.amazon.com/connect/latest/APIReference/API_DeleteIntegrationAssociation.html)
- [ListIntegrationAssociations](https://docs.aws.amazon.com/connect/latest/APIReference/API_ListIntegrationAssociations.html)
