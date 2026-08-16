# A flow module built with the restricted "module as tool" block set (CheckStaffing,
# CheckHoursOfOperation, GetQueueMetrics, InvokeLambdaFunction, ...), marked so it can be
# invoked outside of a flow — e.g. by a Q in Connect orchestration AI agent.
#
# Build and test the module visually first in the Connect admin console's "Create module
# as tool" designer (it only offers the supported tool-module blocks), then export it and
# paste the JSON below — the same round-trip already used for modules/routing_flows/flows/*.tpl
# in the connect-basic repo. Do not hand-author the block content; the exact parameter names
# for blocks like CheckStaffing aren't part of this provider's documentation.
resource "connectracer_connect_flow_module_tool" "agent_availability" {
  instance_id = "12345678-1234-1234-1234-123456789012"
  name        = "CheckAgentAvailability"
  description = "Reports whether an agent is currently staffed on the Basic Routing profile."

  external_invocation_enabled = true

  content = file("${path.module}/agent_availability_module.json") # exported from the console, not hand-written
}
