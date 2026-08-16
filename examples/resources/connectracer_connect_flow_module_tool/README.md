# connectracer_connect_flow_module_tool

Manages an Amazon Connect Flow Module with `ExternalInvocationConfiguration` enabled, so it can be invoked outside of a flow as a tool — the "Flow module tools" mechanism for Q in Connect orchestrator AI agents (as opposed to hosting your own external MCP server).

## Why a separate resource instead of `aws_connect_contact_flow_module`

`ExternalInvocationConfiguration` can only be set on `CreateContactFlowModule` — there is no corresponding field on `UpdateContactFlowModuleMetadata` in the AWS SDK. That means it cannot be layered onto an existing module after the fact the way `connectracer_wisdom_assistant_ai_agents` layers onto an existing Wisdom assistant. This resource therefore owns the whole flow module (content, name, description, settings, tags), not just the tool flag.

Flipping `external_invocation_enabled` after creation forces replacement of the module, since there is no update path for it.

## Module content

Amazon Connect only allows a restricted set of blocks in a tool module — see "Module as tool supported blocks" in the [Flow modules admin guide](https://docs.aws.amazon.com/connect/latest/adminguide/contact-flow-modules.html#module-tool-supported-blocks). Notably this includes `CheckStaffing` and `CheckHoursOfOperation` (real-time agent/staffing checks), `GetQueueMetrics`, and `InvokeLambdaFunction`.

Build and test the module visually in the Connect admin console ("Create module as tool"), export the resulting flow JSON, and pass it as `content` — do not hand-author the JSON. Invalid content for a tool module is rejected by the API at create/update time.

## Example Usage

```hcl
resource "connectracer_connect_flow_module_tool" "agent_availability" {
  instance_id = var.instance_id
  name        = "CheckAgentAvailability"
  description = "Reports whether an agent is currently staffed on the Basic Routing profile."

  external_invocation_enabled = true

  content = file("${path.module}/agent_availability_module.json")
}
```

## Granting an AI agent access

Creating the module is not enough on its own — a security profile must also be granted permission to invoke it. See [`connectracer_connect_security_profile_flow_module`](../connectracer_connect_security_profile_flow_module/).

Two easy-to-miss prerequisites, confirmed against a live instance, neither of which is enforced by `CreateContactFlowModule` itself (it happily accepts a module without them, and `ExternalInvocationConfiguration.Enabled` reads back as `true` regardless):

- **`description` is required.** A flow module without a description isn't invocable as a tool and won't show up as grantable in any security profile — hence `description` is a required argument on this resource, not optional.
- **A version must be released.** `AllowedFlowModules` grants (and the security profile UI's flow module picker) only recognize a module once at least one version exists via `CreateContactFlowModuleVersion`. This resource releases one automatically on creation and again whenever `content`/`settings` change — see the `version` attribute.
- **`tool_id` on `connectracer_connect_ai_tool` is not the flow module's `id`.** `UpdateAIAgent` rejects the raw id with "not found in MCP tools". Use the computed `mcp_tool_id` attribute instead (format `aws_custom_flows__<flow_module_id>_<version>`, reverse-engineered from a console-created tool module since it isn't documented anywhere).

## Import

```shell
terraform import connectracer_connect_flow_module_tool.agent_availability \
  "12345678-1234-1234-1234-123456789012/abcdefab-1234-1234-1234-abcdefabcdef"
```

The import ID is `instance_id/flow_module_id`.
