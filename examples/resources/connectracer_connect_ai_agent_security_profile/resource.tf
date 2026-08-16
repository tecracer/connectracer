# Associates a security profile with an orchestration AI agent, so the agent is
# authorized to invoke governed tools (flow module tools, AgentCore/MCP tools,
# out-of-the-box tools). RETURN_TO_CONTROL tools (Complete/Escalate) don't need this.
resource "connectracer_connect_ai_agent_security_profile" "orchestration" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  ai_agent_arn        = connectracer_connect_ai_agent.orchestration.ai_agent_arn
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
}
