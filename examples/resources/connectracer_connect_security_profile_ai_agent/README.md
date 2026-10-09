# connectracer_connect_security_profile_ai_agent

Allows an external AI agent on a Security Profile (`AllowedAIAgents`). An orchestration AI agent that carries the security profile may then collaborate with that agent, for example hand a call over to an `A2A_SERVER` application. The AWS guide puts it this way: the security profile attached to the AI agent must explicitly allow the third-party application, or collaboration fails at runtime.

## Read-modify-write

`UpdateSecurityProfile` replaces the security profile's entire `AllowedAIAgents` list, and there is no per-entry API. This resource therefore reads the security profile on every create, update, and delete and re-supplies its other fields unchanged (`Description`, `Permissions`, `Applications`, `AllowedFlowModules`, `AllowedAccessControlHierarchyGroupId`, `AllowedAccessControlTags`, `HierarchyRestrictedResources`, `GranularAccessControlConfiguration`). It can coexist with an `aws_connect_security_profile` resource and with `connectracer_connect_security_profile_flow_module` resources on the same security profile.

Multiple `connectracer_connect_security_profile_ai_agent` resources can target the same `security_profile_id` for different agents, as long as they are applied sequentially (Terraform's default behavior).

## Example Usage

```hcl
resource "connectracer_connect_security_profile_ai_agent" "voice_agent" {
  instance_id         = var.instance_id
  security_profile_id = var.ai_agent_security_profile_id
  ai_agent_arn        = connectracer_connect_app_integration.voice_agent.arn
  # type defaults to "THIRD_PARTY", currently the only value the AWS API accepts.
}
```

## Import

```shell
terraform import connectracer_connect_security_profile_ai_agent.voice_agent instance_id/security_profile_id/ai_agent_arn
```

The ARN keeps its own slashes. Only the first two slashes separate the parts.
