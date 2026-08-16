# Grants a security profile permission to invoke a flow module as a tool.
# The AI agent(s) assigned this security profile can then invoke the module,
# e.g. from an orchestrator AI agent's MCP tool list.
resource "connectracer_connect_security_profile_flow_module" "agent_availability" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
  flow_module_id      = connectracer_connect_flow_module_tool.agent_availability.id
  # type defaults to "MCP", currently the only value the AWS API accepts.
}

# Multiple grants can target the same security profile — each resource performs
# a safe read-modify-write against the profile's AllowedFlowModules list, so this
# coexists with an aws_connect_security_profile resource managing everything else
# on the same security profile (permissions, applications, description, ...).
resource "connectracer_connect_security_profile_flow_module" "order_lookup" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  security_profile_id = "134bb661-64ba-4069-856a-d9b086034cf6"
  flow_module_id      = "5e35bd28-12b5-4a74-bbd0-581b12e961dc"
}
