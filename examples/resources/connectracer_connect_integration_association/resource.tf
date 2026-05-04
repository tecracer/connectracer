# Example: Connect Integration Association with Wisdom Knowledge Base

# First, create a QConnect Knowledge Base
resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "my-knowledge-base"
  knowledge_base_type = "CUSTOM"
  description         = "Knowledge base for customer support"

  tags = {
    Environment = "production"
    Team        = "customer-support"
  }
}

# Associate the knowledge base with a Connect instance
resource "connectracer_connect_integration_association" "kb_association" {
  instance_id      = "12eac4f3-e7d2-47f2-a2ac-f10753b6146c"
  integration_type = "WISDOM_KNOWLEDGE_BASE"
  integration_arn  = connectracer_qconnect_knowledgebase.example.knowledge_base_arn

  tags = {
    Environment = "production"
    Team        = "customer-support"
  }
}
