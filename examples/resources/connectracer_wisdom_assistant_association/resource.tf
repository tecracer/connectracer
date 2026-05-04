resource "connectracer_wisdom_assistant" "example" {
  name = "example-assistant"
  type = "AGENT"

  description = "Example Wisdom assistant"

  tags = {
    Environment = "production"
    Team        = "support"
  }
}

resource "connectracer_qconnect_knowledgebase" "example" {
  name                = "example-knowledge-base"
  knowledge_base_type = "CUSTOM"

  description = "Example knowledge base for Wisdom"

  tags = {
    Environment = "production"
    Team        = "support"
  }
}

resource "connectracer_wisdom_assistant_association" "example" {
  assistant_id     = connectracer_wisdom_assistant.example.id
  association_type = "KNOWLEDGE_BASE"

  association_data = {
    knowledge_base_id = connectracer_qconnect_knowledgebase.example.id
  }

  tags = {
    Environment = "production"
    Team        = "support"
  }
}
