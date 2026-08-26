# connectracer_connect_ai_agent_security_profile

Associates a Security Profile with a Q in Connect AI Agent, via `AssociateSecurityProfiles` (`EntityType=AI_AGENT`).

## Why this is needed

Without a security profile, an AI Agent cannot invoke any *governed* tool — that covers Flow Module tools (see [`connectracer_connect_security_profile_flow_module`](../connectracer_connect_security_profile_flow_module/)), AgentCore/MCP tools, and out-of-the-box tools (Cases, Tasks, Customer Profiles, Knowledge Base retrieval) — even when the tool itself and its security-profile-level grant (`AllowedFlowModules`, etc.) are otherwise fully configured. This resource is the missing link between "the security profile is allowed to use tool X" and "this specific agent has that security profile at all."

`RETURN_TO_CONTROL` tools (`connectracer_connect_ai_tool` with `tool_type = "RETURN_TO_CONTROL"`, e.g. the default `Complete`/`Escalate` tools) are **not** governed this way — they don't access any protected resource, they just end the AI turn and hand control back to the contact flow. They work without any security profile association.

Unlike `connectracer_connect_security_profile_flow_module` (which has to read-modify-write `UpdateSecurityProfile`'s full `AllowedFlowModules` list), this resource wraps a true associate/disassociate API — no read-modify-write needed.

## Example Usage

```hcl
resource "connectracer_connect_ai_agent_security_profile" "orchestration" {
  instance_id         = var.instance_id
  ai_agent_arn        = connectracer_connect_ai_agent.orchestration.ai_agent_arn
  security_profile_id = var.ai_agent_security_profile_id
}
```

## Related resources

| Resource | Role |
|----------|------|
| `connectracer_connect_ai_agent` | Creates the orchestration AI agent |
| `connectracer_connect_security_profile_flow_module` | Grants the security profile access to a specific flow module tool |
| `connectracer_connect_flow_module_tool` | Creates the flow module with `external_invocation_enabled = true` |

## Import

```shell
terraform import connectracer_connect_ai_agent_security_profile.orchestration \
  "12345678-1234-1234-1234-123456789012/arn:aws:wisdom:eu-central-1:123456789012:ai-agent/97a0c52f-aaaa-bbbb-cccc-dddddddddddd/134bb661-64ba-4069-856a-d9b086034cf6"
```

The import ID is `instance_id/ai_agent_arn/security_profile_id`. Since the ARN itself contains `/`, parsing takes the first segment as `instance_id` and the last segment as `security_profile_id`, treating everything in between as the ARN.
