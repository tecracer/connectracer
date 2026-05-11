terraform {
  required_providers {
    connectracer = {
      source  = "registry.terraform.io/tecracer/connectracer"
      version = "0.1.3"
    }
  }
}

provider "connectracer" {}

# Create a Wisdom Assistant (this will work since we fixed tags_all)
resource "connectracer_wisdom_assistant" "example" {
  name        = "test-integration-assistant"
  type        = "AGENT"
  description = "Test Wisdom Assistant for integration testing"

  tags = {
    Environment = "test"
    Purpose     = "integration-test"
  }
}

# Create a QConnect Knowledge Base (this will work since we fixed tags_all)
resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "test-integration-kb"
  knowledge_base_type = "CUSTOM"
  description         = "Test Knowledge base for integration testing"

  tags = {
    Environment = "test"
    Purpose     = "integration-test"
  }
}

# Associate the Knowledge Base with the Wisdom Assistant
resource "connectracer_wisdom_assistant_association" "example" {
  assistant_id     = connectracer_wisdom_assistant.example.id
  association_type = "KNOWLEDGE_BASE"

  association_data = {
    knowledge_base_id = connectracer_qconnect_knowledgebase.example.id
  }

  tags = {
    Environment = "test"
  }
}

# Associate the Wisdom Assistant with Connect Instance
# NOTE: Replace with your actual Connect instance ID
resource "connectracer_connect_integration_association" "assistant" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c" # Replace with your instance ID
  integration_type = "WISDOM_ASSISTANT"
  integration_arn  = connectracer_wisdom_assistant.example.assistant_arn

  # Note: source_type is NOT required for WISDOM integrations
  # It will be set to null by the provider

  tags = {
    Environment = "test"
  }
}

# Also associate the Knowledge Base directly with Connect Instance
resource "connectracer_connect_integration_association" "knowledge_base" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c" # Replace with your instance ID
  integration_type = "WISDOM_KNOWLEDGE_BASE"
  integration_arn  = connectracer_qconnect_knowledgebase.example.knowledge_base_arn

  # Note: source_type is NOT required for WISDOM integrations

  tags = {
    Environment = "test"
  }
}

# Outputs
output "assistant_id" {
  description = "Wisdom Assistant ID"
  value       = connectracer_wisdom_assistant.example.id
}

output "assistant_tags_all" {
  description = "All assistant tags including provider-added"
  value       = connectracer_wisdom_assistant.example.tags_all
}

output "kb_id" {
  description = "Knowledge Base ID"
  value       = connectracer_qconnect_knowledgebase.example.id
}

output "kb_tags_all" {
  description = "All KB tags including provider-added"
  value       = connectracer_qconnect_knowledgebase.example.tags_all
}

output "assistant_association_id" {
  description = "ID of the assistant integration association"
  value       = connectracer_connect_integration_association.assistant.id
}

output "kb_association_id" {
  description = "ID of the knowledge base integration association"
  value       = connectracer_connect_integration_association.knowledge_base.id
}

output "assistant_source_type" {
  description = "Source type for assistant integration (should be null for WISDOM)"
  value       = connectracer_connect_integration_association.assistant.source_type
}

output "kb_source_type" {
  description = "Source type for KB integration (should be null for WISDOM)"
  value       = connectracer_connect_integration_association.knowledge_base.source_type
}
