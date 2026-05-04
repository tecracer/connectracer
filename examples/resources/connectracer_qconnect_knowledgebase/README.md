# AWS Q Connect Knowledge Base Resource

This resource allows you to create and manage AWS Q Connect (formerly Amazon Connect Wisdom) knowledge bases.

## Example Usage

### Basic Custom Knowledge Base

```hcl
resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "my-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "My Q Connect Knowledge Base for customer support"

  tags = {
    Environment = "production"
    Team        = "customer-support"
  }
}
```

### Knowledge Base with KMS Encryption

```hcl
resource "connectracer_qconnect_knowledgebase" "encrypted" {
  name                = "encrypted-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "Encrypted Q Connect Knowledge Base"

  server_side_encryption_configuration {
    kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  }

  tags = {
    Environment = "production"
    Encrypted   = "true"
  }
}
```

### External Knowledge Base (Salesforce Integration)

```hcl
resource "connectracer_qconnect_knowledgebase" "external" {
  name                = "external-knowledge-base"
  knowledge_base_type = "EXTERNAL"
  description         = "External Q Connect Knowledge Base from Salesforce"

  source_configuration {
    app_integration_arn = "arn:aws:app-integrations:us-east-1:123456789012:data-integration/salesforce-integration"
  }

  rendering_configuration {
    template_uri = "https://mycompany.salesforce.com/article/${Id}"
  }

  tags = {
    Environment = "production"
    Source      = "Salesforce"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the knowledge base.
* `knowledge_base_type` - (Required) The type of knowledge base. Valid values: `CUSTOM`, `EXTERNAL`. Changing this forces a new resource to be created.
* `description` - (Optional) The description of the knowledge base.
* `tags` - (Optional) A map of tags to assign to the resource.
* `rendering_configuration` - (Optional) Information about how to render the content. See below.
* `server_side_encryption_configuration` - (Optional) The KMS key used for encryption. See below.
* `source_configuration` - (Optional) The source of the knowledge base content. Only set for `EXTERNAL` knowledge bases. See below.

### rendering_configuration

The `rendering_configuration` block supports:

* `template_uri` - (Optional) A URI template containing exactly one variable in `${variableName}` format. This is used to construct URLs to knowledge base articles. For example: `https://example.com/article/${Id}`.

### server_side_encryption_configuration

The `server_side_encryption_configuration` block supports:

* `kms_key_id` - (Optional) The KMS key ID or ARN. The customer managed key must have a policy that allows `kms:CreateGrant`, `kms:DescribeKey`, `kms:Decrypt`, and `kms:GenerateDataKey*` permissions to the IAM identity using the key to invoke Q Connect.

### source_configuration

The `source_configuration` block supports:

* `app_integration_arn` - (Optional) The Amazon Resource Name (ARN) of the AppIntegrations DataIntegration to use for ingesting content from external sources.

**Note:** For external knowledge bases like Salesforce or ServiceNow, you cannot reuse Amazon AppIntegrations DataIntegrations. If you need to modify fields being ingested, you must delete and recreate both the DataIntegration and the knowledge base.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the knowledge base.
* `knowledge_base_arn` - The ARN of the knowledge base.
* `status` - The status of the knowledge base (e.g., `ACTIVE`, `CREATE_IN_PROGRESS`, `CREATE_FAILED`, `DELETE_IN_PROGRESS`).

## Import

Q Connect knowledge bases can be imported using the knowledge base ID, e.g.,

```shell
terraform import connectracer_qconnect_knowledgebase.example a1b2c3d4-5678-90ab-cdef-EXAMPLE11111
```

## Update Limitations

AWS Q Connect has limited support for updating knowledge base properties after creation. The following update capabilities are available:

### Updatable Fields

* **Tags**: Can be updated in-place using the `TagResource` and `UntagResource` APIs.
* **Template URI**: Can be updated in-place using the `UpdateKnowledgeBaseTemplateUri` API.

### Non-Updatable Fields

The following fields **cannot** be updated after creation and will trigger a warning:

* **Name**: Changing the `name` will display a warning but will NOT recreate the resource automatically.
* **Description**: Changing the `description` will display a warning but will NOT recreate the resource automatically.

### Force Recreation Fields

The following fields will force resource recreation:

* **knowledge_base_type**: Changing this requires a new resource.
* **Encryption Configuration**: Cannot be changed after creation. Requires recreating the resource.
* **Source Configuration**: Cannot be changed after creation. Requires recreating the resource.

### Working Around Update Limitations

To update non-updatable fields, you must recreate the resource. You can use one of the following approaches:

1. **Manual taint and apply**:
   ```shell
   terraform taint connectracer_qconnect_knowledgebase.example
   terraform apply
   ```

2. **Use lifecycle rules for zero downtime**:
   ```hcl
   resource "connectracer_qconnect_knowledgebase" "example" {
     name                = "updated-name"
     knowledge_base_type = "CUSTOM"
     description         = "Updated description"

     lifecycle {
       create_before_destroy = true
     }
   }
   ```

## External Knowledge Bases

When using external knowledge bases (Salesforce, ServiceNow, etc.):

1. Create the AppIntegrations DataIntegration first
2. Create the Q Connect knowledge base with the DataIntegration ARN
3. To modify ingested fields:
   - Delete the knowledge base
   - Delete the DataIntegration
   - Create a new DataIntegration with updated configuration
   - Create a new knowledge base

Example with dependencies:

```hcl
# Note: aws_appintegrations_data_integration is a separate resource
# This is just an example showing the relationship

resource "connectracer_qconnect_knowledgebase" "salesforce" {
  name                = "salesforce-kb"
  knowledge_base_type = "EXTERNAL"

  source_configuration {
    app_integration_arn = aws_appintegrations_data_integration.salesforce.arn
  }

  rendering_configuration {
    template_uri = "https://mycompany.salesforce.com/article/${Id}"
  }
}
```
