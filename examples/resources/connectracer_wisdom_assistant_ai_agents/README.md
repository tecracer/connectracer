# connectracer_wisdom_assistant_ai_agents

Assigns default AI agents to an Amazon Q in Connect assistant. Each agent type (e.g. `SELF_SERVICE`, `ORCHESTRATION`) can be mapped to an agent ID and version.

When assigning an `ORCHESTRATION` agent for chat self-service, you must also set `orchestrator_use_cases` with `Connect.SelfService`.

## Example Usage

### Self-service agent only

```hcl
resource "connectracer_wisdom_assistant_ai_agents" "self_service" {
  assistant_id = var.assistant_id

  ai_agent_configuration = {
    SELF_SERVICE = "${connectracer_connect_ai_agent.self_service.id}:$LATEST"
  }
}
```

### Orchestration agent for chat self-service

```hcl
resource "connectracer_wisdom_assistant_ai_agents" "orchestration" {
  assistant_id = var.assistant_id

  ai_agent_configuration = {
    ORCHESTRATION = "${connectracer_connect_ai_agent.orchestration.id}:$LATEST"
  }

  orchestrator_use_cases = {
    ORCHESTRATION = "Connect.SelfService"
  }
}
```

Use `$LATEST` for the newest published version, or a numeric version (e.g. `:1`) for a pinned release.

## Complete wiring

See [`complete_chat_self_service.tf`](./complete_chat_self_service.tf) for assistant → AI agents → escalation tool → agent assignment in one configuration.

For the full chat stack including Lex locale build and Q-in-Connect intent, see [`../../chat-self-service/`](../../chat-self-service/).

## Related resources

| Resource | Role |
|----------|------|
| `connectracer_connect_ai_agent` | Create SELF_SERVICE / ORCHESTRATION agents |
| `connectracer_connect_ai_tool` | Attach RETURN_TO_CONTROL escalation tools to orchestration agents |
| `connectracer_lexv2models_bot_locale` | Lex locale with optional `build_on_apply` |
| `connectracer_q_in_connect_intent` | AMAZON.QInConnectIntent with optional `build_locale_on_apply` |

## Import

```shell
terraform import connectracer_wisdom_assistant_ai_agents.orchestration \
  "12345678-1234-1234-1234-123456789012"
```

The import ID is the assistant ID. Existing agent assignments are read from AWS on import.
