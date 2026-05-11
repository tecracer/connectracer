terraform {
  required_providers {
    connectracer = {
      source  = "registry.terraform.io/tecracer/connectracer"
      version = "0.1.2"
    }
  }
}

provider "connectracer" {}

data "connectracer_wisdom_knowledge_bases" "example" {}

output "knowledge_bases" {
  description = "List of all AWS Wisdom knowledge bases"
  value       = data.connectracer_wisdom_knowledge_bases.example.knowledge_bases
}

output "knowledge_base_count" {
  description = "Number of knowledge bases found"
  value       = length(data.connectracer_wisdom_knowledge_bases.example.knowledge_bases)
}
