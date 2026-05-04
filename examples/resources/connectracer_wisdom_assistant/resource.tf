resource "connectracer_wisdom_assistant" "example" {
  name        = "my-assistant"
  type        = "AGENT"
  description = "My Wisdom Assistant for customer support"

  tags = {
    Environment = "production"
    Team        = "customer-support"
  }
}

# Example with KMS encryption
resource "connectracer_wisdom_assistant" "encrypted" {
  name        = "encrypted-assistant"
  type        = "AGENT"
  description = "Encrypted Wisdom Assistant"

  server_side_encryption_configuration {
    kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  }

  tags = {
    Environment = "production"
    Encrypted   = "true"
  }
}
