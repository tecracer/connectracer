# An MCP server that an AI agent can call as a tool.
resource "connectracer_connect_app_integration" "mcp_server" {
  name             = "order-tools"
  namespace        = "order-tools"
  description      = "MCP server with the order lookup tools"
  access_url       = "https://mcp.example.com/mcp"
  application_type = "MCP_SERVER"
}

# An external AI agent that Connect collaborates with over the A2A protocol.
# Connect presents the secret's value as the bearer token on the WebSocket
# upgrade, so the secret has to be encrypted with a customer managed KMS key
# whose policy lets connect.amazonaws.com decrypt it.
resource "connectracer_connect_app_integration" "voice_agent" {
  name             = "voice-agent"
  namespace        = "voice-agent"
  description      = "External voice agent, reached over A2A"
  access_url       = "wss://agents.example.com/a2a/voice-agent"
  application_type = "A2A_SERVER"

  auth_config = {
    auth_type                      = "API_KEY"
    credential_provider_identifier = aws_secretsmanager_secret.voice_agent_api_key.arn
  }
}

# Either application reaches the Connect instance through an association.
resource "connectracer_connect_integration_association" "voice_agent" {
  instance_id      = "12345678-1234-1234-1234-123456789012"
  integration_type = "APPLICATION"
  integration_arn  = connectracer_connect_app_integration.voice_agent.arn
}
