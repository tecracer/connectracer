# connectracer_connect_security_profile_flow_module

Grants a Security Profile permission to invoke a Flow Module as a tool — the same governance mechanism Amazon Connect uses to control which MCP tools an AI agent (or human agent) can access, reusing the existing security profile framework.

## Read-modify-write

`UpdateSecurityProfile` replaces the security profile's entire `AllowedFlowModules` list — there's no per-entry associate/disassociate API (unlike, say, `AssociateApprovedOrigin`/`DisassociateApprovedOrigin`). This resource therefore performs a read-modify-write on every create, update, and delete: it re-fetches the security profile's other fields (`Description`, `Permissions`, `Applications`, `AllowedAccessControlHierarchyGroupId`, `AllowedAccessControlTags`, `HierarchyRestrictedResources`, `GranularAccessControlConfiguration`) and re-supplies them unchanged, so it can coexist with an `aws_connect_security_profile` resource managing the rest of the security profile — the same pattern `connectracer_connect_ai_tool` uses for an AI agent's tool list.

Multiple `connectracer_connect_security_profile_flow_module` resources can target the same `security_profile_id` for different flow modules, as long as they are applied sequentially (Terraform's default behavior — do not run these with `-parallelism` changes that would let them race).

## Example Usage

```hcl
resource "connectracer_connect_security_profile_flow_module" "agent_availability" {
  instance_id         = var.instance_id
  security_profile_id = var.agent_security_profile_id
  flow_module_id      = connectracer_connect_flow_module_tool.agent_availability.id
  # type defaults to "MCP", currently the only value the AWS API accepts.
}
```

## Related resources

| Resource | Role |
|----------|------|
| `connectracer_connect_flow_module_tool` | Creates the flow module with `external_invocation_enabled = true` |
| `connectracer_connect_ai_tool` | MCP/RETURN_TO_CONTROL/CONSTANT tools directly on an orchestration AI agent |

## Import

```shell
terraform import connectracer_connect_security_profile_flow_module.agent_availability \
  "12345678-1234-1234-1234-123456789012/134bb661-64ba-4069-856a-d9b086034cf6/abcdefab-1234-1234-1234-abcdefabcdef"
```

The import ID is `instance_id/security_profile_id/flow_module_id`.
