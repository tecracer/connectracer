# connectracer_wisdom_assistant_association

Manages an AWS Wisdom Assistant Association resource. An assistant association creates a link between a Wisdom assistant and a knowledge base, allowing the assistant to use content from the knowledge base when providing recommendations.

## Example Usage

### Basic Usage

```terraform
resource "connectracer_wisdom_assistant" "example" {
  name = "example-assistant"
  type = "AGENT"
}

resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "example-knowledge-base"
  knowledge_base_type = "CUSTOM"
}

resource "connectracer_wisdom_assistant_association" "example" {
  assistant_id     = connectracer_wisdom_assistant.example.id
  association_type = "KNOWLEDGE_BASE"

  association_data = {
    knowledge_base_id = connectracer_qconnect_knowledgebase.example.id
  }
}
```

### With Tags

```terraform
resource "connectracer_wisdom_assistant_association" "example" {
  assistant_id     = connectracer_wisdom_assistant.example.id
  association_type = "KNOWLEDGE_BASE"

  association_data = {
    knowledge_base_id = connectracer_qconnect_knowledgebase.example.id
  }

  tags = {
    Environment = "production"
    Team        = "support"
    Purpose     = "Customer support knowledge base"
  }
}
```

## Argument Reference

The following arguments are required:

* `assistant_id` - (Required, Forces replacement) The ID of the Wisdom assistant. Can be obtained from the `id` attribute of a `connectracer_wisdom_assistant` resource.
* `association_type` - (Required, Forces replacement) The type of association. Currently only `KNOWLEDGE_BASE` is supported.
* `association_data` - (Required, Forces replacement) A configuration block containing the association data. See below.

The following arguments are optional:

* `tags` - (Optional) A map of tags to assign to the assistant association. Tags are the only attribute that can be updated after creation.

### association_data

The `association_data` block supports the following:

* `knowledge_base_id` - (Required) The ID of the knowledge base to associate with the assistant.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the assistant association.
* `assistant_association_arn` - The ARN of the assistant association.

## Import

Wisdom Assistant Associations can be imported using the `assistant_id` and `association_id` separated by a forward slash (`/`), e.g.,

```shell
terraform import connectracer_wisdom_assistant_association.example assistant-id-12345/association-id-67890
```

## Notes

* An assistant can have only one association with a knowledge base at a time.
* Assistant associations are immutable except for tags. If you need to change the `assistant_id`, `association_type`, or `knowledge_base_id`, you must create a new association.
* There is no AWS API to update assistant associations. Only tags can be modified after creation using the TagResource and UntagResource APIs.
* When the association is deleted, the assistant and knowledge base resources themselves are not affected.

## AWS Documentation

* [AWS Wisdom Assistant Association](https://docs.aws.amazon.com/wisdom/latest/APIReference/API_AssistantAssociationData.html)
* [CreateAssistantAssociation API](https://docs.aws.amazon.com/wisdom/latest/APIReference/API_CreateAssistantAssociation.html)
