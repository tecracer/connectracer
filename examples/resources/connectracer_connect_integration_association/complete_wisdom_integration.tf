# Example: Complete Wisdom Integration with Connect

# Create a Wisdom Assistant
resource "connectracer_wisdom_assistant" "example" {
  name        = "my-assistant"
  type        = "AGENT"
  description = "Wisdom Assistant for agent assistance"

  tags = {
    Environment = "production"
  }
}

# Create a QConnect Knowledge Base
resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "my-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "Knowledge base content"

  tags = {
    Environment = "production"
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
    Environment = "production"
  }
}

# Associate the Wisdom Assistant with Connect Instance
resource "connectracer_connect_integration_association" "assistant" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c"
  integration_type = "WISDOM_ASSISTANT"
  integration_arn  = connectracer_wisdom_assistant.example.assistant_arn

  tags = {
    Environment = "production"
  }
}

# Also associate the Knowledge Base directly with Connect Instance
resource "connectracer_connect_integration_association" "knowledge_base" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c"
  integration_type = "WISDOM_KNOWLEDGE_BASE"
  integration_arn  = connectracer_qconnect_knowledgebase.example.knowledge_base_arn

  tags = {
    Environment = "production"
  }
}

# Outputs
output "assistant_association_id" {
  description = "ID of the assistant integration association"
  value       = connectracer_connect_integration_association.assistant.id
}

output "kb_association_id" {
  description = "ID of the knowledge base integration association"
  value       = connectracer_connect_integration_association.knowledge_base.id
}
