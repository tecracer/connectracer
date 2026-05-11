terraform {
  required_providers {
    connectracer = {
      source  = "registry.terraform.io/tecracer/connectracer"
      version = "0.1.2"
    }
  }
}

provider "connectracer" {}

# Test Wisdom Assistant
resource "connectracer_wisdom_assistant" "example" {
  name        = "test-assistant-v2"
  description = "Test assistant to verify tag handling with tags_all"
  type        = "AGENT"

  tags = {
    Environment = "test"
    Version     = "v2"
  }
}

# Note: Commented out because it requires a real S3 bucket
# # Test AppIntegrations Data Integration
# resource "connectracer_appintegrations_data_integration" "example" {
#   name        = "test-integration"
#   description = "Test data integration to verify tag handling"
#   source_uri  = "s3://test-bucket-for-connectracer-test"
#   kms_key     = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
#
#   tags = {
#     Environment = "test"
#   }
# }

output "assistant_tags" {
  description = "User-provided tags only"
  value       = connectracer_wisdom_assistant.example.tags
}

output "assistant_tags_all" {
  description = "All tags including provider-added AmazonConnectEnabled"
  value       = connectracer_wisdom_assistant.example.tags_all
}
