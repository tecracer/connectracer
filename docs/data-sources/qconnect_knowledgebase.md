# connectracer_qconnect_knowledgebase (Data Source)

Data source for looking up an AWS Q Connect Knowledge Base by ID.

## Example Usage

### Basic Usage

```terraform
data "connectracer_qconnect_knowledgebase" "example" {
  knowledge_base_id = "937de449-ad2c-42e2-b89a-2b69e11c4047"
}

output "knowledge_base_name" {
  value = data.connectracer_qconnect_knowledgebase.example.name
}

output "knowledge_base_arn" {
  value = data.connectracer_qconnect_knowledgebase.example.knowledge_base_arn
}
```

### Using with Resource Reference

```terraform
# Create a knowledge base
resource "connectracer_qconnect_knowledgebase" "main" {
  name                = "my-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "My knowledge base"
}

# Look it up using datasource
data "connectracer_qconnect_knowledgebase" "lookup" {
  knowledge_base_id = connectracer_qconnect_knowledgebase.main.id
}

# Use datasource attributes
output "kb_status" {
  value = data.connectracer_qconnect_knowledgebase.lookup.status
}
```

### External Knowledge Base with AppIntegrations

```terraform
data "connectracer_qconnect_knowledgebase" "external" {
  knowledge_base_id = "b3897cb5-3c95-4481-b5b4-1e8013bfaebe"
}

# Access the AppIntegrations ARN
output "app_integration_arn" {
  value = data.connectracer_qconnect_knowledgebase.external.source_configuration.app_integration_arn
}
```

## Argument Reference

The following arguments are required:

* `knowledge_base_id` - (Required) The ID of the knowledge base to look up.

## Attribute Reference

In addition to the argument above, the following attributes are exported:

* `id` - The ID of the knowledge base (same as `knowledge_base_id`).
* `knowledge_base_arn` - The ARN of the knowledge base.
* `name` - The name of the knowledge base.
* `knowledge_base_type` - The type of knowledge base. Valid values: `EXTERNAL`, `CUSTOM`.
* `description` - The description of the knowledge base.
* `status` - The status of the knowledge base (e.g., `ACTIVE`, `CREATING`, `DELETING`).
* `tags` - Tags applied to the knowledge base.

### Nested Attributes

#### `rendering_configuration`

* `template_uri` - A URI template containing exactly one variable in `${variableName}` format.

#### `server_side_encryption_configuration`

* `kms_key_id` - The KMS key ID or ARN used for encryption.

#### `source_configuration`

Only present for `EXTERNAL` knowledge bases:

* `app_integration_arn` - The Amazon Resource Name (ARN) of the AppIntegrations DataIntegration.

## Use Cases

### 1. Reference Existing Knowledge Base

Use this datasource to reference a knowledge base created outside of Terraform:

```terraform
data "connectracer_qconnect_knowledgebase" "existing" {
  knowledge_base_id = var.existing_kb_id
}

# Use it in other resources
resource "connectracer_wisdom_assistant_association" "example" {
  assistant_id     = connectracer_wisdom_assistant.main.id
  association_type = "KNOWLEDGE_BASE"
  
  association_data = {
    knowledge_base_id = data.connectracer_qconnect_knowledgebase.existing.id
  }
}
```

### 2. Validate Knowledge Base Configuration

Check the configuration of a knowledge base:

```terraform
data "connectracer_qconnect_knowledgebase" "check" {
  knowledge_base_id = var.kb_id
}

# Ensure it's the right type
output "is_external" {
  value = data.connectracer_qconnect_knowledgebase.check.knowledge_base_type == "EXTERNAL"
}

# Check if it's active
output "is_active" {
  value = data.connectracer_qconnect_knowledgebase.check.status == "ACTIVE"
}
```

### 3. Extract AppIntegrations Configuration

For EXTERNAL knowledge bases, extract the AppIntegrations configuration:

```terraform
data "connectracer_qconnect_knowledgebase" "external_kb" {
  knowledge_base_id = var.external_kb_id
}

# Output the AppIntegrations ARN for documentation
output "data_integration" {
  description = "AppIntegrations DataIntegration ARN"
  value       = try(
    data.connectracer_qconnect_knowledgebase.external_kb.source_configuration.app_integration_arn,
    "Not an EXTERNAL knowledge base"
  )
}
```

## Notes

### Knowledge Base Types

- **CUSTOM**: Knowledge base content is managed directly within QConnect
- **EXTERNAL**: Knowledge base content comes from external sources via AppIntegrations

### Status Values

Possible status values include:
- `CREATING` - Knowledge base is being created
- `ACTIVE` - Knowledge base is ready to use
- `DELETING` - Knowledge base is being deleted
- `DELETED` - Knowledge base has been deleted
- `CREATE_FAILED` - Knowledge base creation failed
- `DELETE_FAILED` - Knowledge base deletion failed

### Error Handling

If the knowledge base doesn't exist or you don't have permission to access it, the datasource will return an error:

```
Error: Error reading Q Connect Knowledge Base
Unable to read knowledge base <id>, got error: ...
```

## API Documentation

- [GetKnowledgeBase](https://docs.aws.amazon.com/wisdom/latest/APIReference/API_GetKnowledgeBase.html)

## Related Resources

- `connectracer_qconnect_knowledgebase` - Create and manage QConnect knowledge bases
- `connectracer_wisdom_assistant_association` - Associate knowledge bases with Wisdom assistants
- `connectracer_appintegrations_data_integration` - For EXTERNAL knowledge bases
- `connectracer_connect_integration_association` - Integrate knowledge bases with Connect instances
