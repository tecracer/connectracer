resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "my-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "My Q Connect Knowledge Base for customer support"

  tags = {
    Environment = "production"
    Team        = "customer-support"
  }
}

# Example with KMS encryption
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

# Example with external knowledge base (requires AppIntegration)
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
