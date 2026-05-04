# Look up an existing QConnect Knowledge Base by ID
data "connectracer_qconnect_knowledgebase" "example" {
  knowledge_base_id = "937de449-ad2c-42e2-b89a-2b69e11c4047"
}

# Use the datasource outputs
output "kb_name" {
  value = data.connectracer_qconnect_knowledgebase.example.name
}

output "kb_arn" {
  value = data.connectracer_qconnect_knowledgebase.example.knowledge_base_arn
}

output "kb_type" {
  value = data.connectracer_qconnect_knowledgebase.example.knowledge_base_type
}

output "kb_status" {
  value = data.connectracer_qconnect_knowledgebase.example.status
}
