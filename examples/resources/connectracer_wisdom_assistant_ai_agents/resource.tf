# Assign a default SELF_SERVICE AI agent to an existing Q in Connect assistant
resource "connectracer_wisdom_assistant_ai_agents" "self_service" {
  assistant_id = connectracer_wisdom_assistant.assistant.id

  ai_agent_configuration = {
    SELF_SERVICE = "${connectracer_connect_ai_agent.self_service.id}:$LATEST"
  }
}

# Orchestration agent for chat self-service (requires orchestrator_use_cases)
resource "connectracer_wisdom_assistant_ai_agents" "orchestration" {
  assistant_id = connectracer_wisdom_assistant.assistant.id

  ai_agent_configuration = {
    ORCHESTRATION = "${connectracer_connect_ai_agent.orchestration.id}:$LATEST"
  }

  orchestrator_use_cases = {
    ORCHESTRATION = "Connect.SelfService"
  }
}
