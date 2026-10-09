# Allows an external AI agent on a security profile. An orchestration AI agent
# that carries this security profile may then hand calls over to it.
resource "connectracer_connect_security_profile_ai_agent" "voice_agent" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
  ai_agent_arn        = connectracer_connect_app_integration.voice_agent.arn
  # type defaults to "THIRD_PARTY", currently the only value the AWS API accepts.
}

# The orchestration AI agent carries the security profile, on its versions too.
resource "connectracer_connect_ai_agent_security_profile" "voice_handoff" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  ai_agent_arn        = connectracer_connect_ai_agent.voice_handoff.ai_agent_arn
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
  ai_agent_version    = connectracer_connect_ai_agent.voice_handoff.version_number
}
