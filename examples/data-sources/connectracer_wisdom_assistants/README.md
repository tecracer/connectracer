# AWS Wisdom Assistants Data Source

This data source allows you to retrieve information about AWS Wisdom assistants in your account.

## Example Usage

```hcl
data "connectracer_wisdom_assistants" "example" {
}

output "assistants" {
  value = data.connectracer_wisdom_assistants.example.assistants
}

output "assistant_names" {
  value = [for assistant in data.connectracer_wisdom_assistants.example.assistants : assistant.name]
}
```

## Argument Reference

This data source does not require any arguments.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A placeholder identifier for the data source.
* `assistants` - A list of assistant objects. Each object contains the following attributes:
  * `assistant_arn` - The ARN of the assistant.
  * `assistant_id` - The ID of the assistant.
  * `integration_configuration` - The integration configuration for the assistant:
    * `topic_integration_arn` - The ARN of the SNS topic integration.
  * `name` - The name of the assistant.
  * `status` - The status of the assistant (e.g., `ACTIVE`).
  * `tags` - A map of tags associated with the assistant.
  * `type` - The type of the assistant (e.g., `AGENT`).

## Example Response

The data source returns information in the following structure:

```hcl
assistants = [
  {
    assistant_arn = "arn:aws:wisdom:eu-central-1:687634808893:assistant/bc52879e-19e9-4ec4-ba5c-11f95ad7bb9f"
    assistant_id = "bc52879e-19e9-4ec4-ba5c-11f95ad7bb9f"
    integration_configuration = {
      topic_integration_arn = "arn:aws:sns:eu-central-1:984356714394:wisdom-dd658af6-5860-48d6-bae6-f19a6ae4bcd3"
    }
    name = "ql-kb-assistant"
    status = "ACTIVE"
    tags = {
      "map-migrated" = "mig2OWW4DE7H2"
      "AmazonConnectEnabled" = "True"
    }
    type = "AGENT"
  }
]
```
