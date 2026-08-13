# Assign a default SELF_SERVICE AI agent to a Q in Connect assistant.
# Use agent_id:$LATEST or agent_id:version_number as the value.
resource "connectracer_wisdom_assistant_ai_agents" "self_service" {
  assistant_id = "12345678-1234-1234-1234-123456789012"

  ai_agent_configuration = {
    SELF_SERVICE = "97a0c52f-aaaa-bbbb-cccc-dddddddddddd:$LATEST"
  }
}

# Orchestration agent for chat self-service.
# orchestrator_use_cases is required when assigning an ORCHESTRATION agent.
resource "connectracer_wisdom_assistant_ai_agents" "orchestration" {
  assistant_id = "12345678-1234-1234-1234-123456789012"

  ai_agent_configuration = {
    ORCHESTRATION = "539eb230-bbbb-cccc-dddd-eeeeeeeeeeee:$LATEST"
  }

  orchestrator_use_cases = {
    ORCHESTRATION = "Connect.SelfService"
  }
}
