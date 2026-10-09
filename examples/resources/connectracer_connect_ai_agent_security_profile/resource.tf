# Associates a security profile with an orchestration AI agent, so the agent is
# authorized to invoke governed tools (flow module tools, AgentCore/MCP tools,
# out-of-the-box tools). RETURN_TO_CONTROL tools (Complete/Escalate) don't need this.
resource "connectracer_connect_ai_agent_security_profile" "orchestration" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  ai_agent_arn        = connectracer_connect_ai_agent.orchestration.ai_agent_arn
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
}

# An orchestration AI agent that hands calls over to an external agent (A2A)
# needs the association on its versions too. With ai_agent_version set, the
# security profile is also associated with :$LATEST, :$SAVED and :<version>,
# and the numbered association follows when the agent publishes a new version.
resource "connectracer_connect_ai_agent_security_profile" "voice_handoff" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  ai_agent_arn        = connectracer_connect_ai_agent.voice_handoff.ai_agent_arn
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
  ai_agent_version    = connectracer_connect_ai_agent.voice_handoff.version_number
}
