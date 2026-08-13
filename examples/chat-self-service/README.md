# Chat self-service with Amazon Q in Connect

End-to-end example wiring the connectracer resources used for **Lex chat + orchestration self-service**:

```
Wisdom assistant
    ├── ORCHESTRATION AI agent (+ Escalate tool)
    ├── SELF_SERVICE AI agent (optional, for agent-assist)
    ├── wisdom_assistant_ai_agents (orchestrator_use_cases = Connect.SelfService)
    ├── Lex bot locale (build_on_apply)
    └── Q-in-Connect intent (build_locale_on_apply)
```

This mirrors the pattern used in production Connect chat flows: Lex invokes Q in Connect, the orchestration agent handles the conversation, and a `RETURN_TO_CONTROL` tool hands off to the contact flow for human escalation.

## Prerequisites

- An existing Lex V2 bot ID (`bot_id`, 10 characters)
- Amazon Connect instance ARN
- AI prompt IDs from the Connect / Q console (or created elsewhere)
- Terraform >= 1.0 and the connectracer provider:

```hcl
terraform {
  required_providers {
    connectracer = {
      source  = "tecracer/connectracer"
      version = ">= 0.5.0"
    }
  }
}

provider "connectracer" {
  region = "eu-central-1"
}
```

## Usage

Copy this folder or reference it as a module. Set variables in `terraform.tfvars`:

```hcl
lex_bot_id                    = "ABCDE12345"
connect_instance_arn          = "arn:aws:connect:eu-central-1:123456789012:instance/..."
orchestration_prompt_id       = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
self_service_answer_prompt_id = "ffffffff-1111-2222-3333-444444444444"
primary_locale_id             = "en_US"
```

Then:

```shell
terraform init
terraform plan
terraform apply
```

## Resource-specific examples

| Step | Example path |
|------|----------------|
| Assistant | [`../resources/connectracer_wisdom_assistant/`](../resources/connectracer_wisdom_assistant/) |
| Orchestration agent | [`../resources/connectracer_connect_ai_agent/`](../resources/connectracer_connect_ai_agent/) |
| Escalation tool | [`../resources/connectracer_connect_ai_tool/`](../resources/connectracer_connect_ai_tool/) |
| Agent assignment | [`../resources/connectracer_wisdom_assistant_ai_agents/`](../resources/connectracer_wisdom_assistant_ai_agents/) |
| Lex locale + build | [`../resources/connectracer_lexv2models_bot_locale/`](../resources/connectracer_lexv2models_bot_locale/) |
| Q intent + build | [`../resources/connectracer_q_in_connect_intent/`](../resources/connectracer_q_in_connect_intent/) |

## Notes

- Apply order matters: create agents and tools before `wisdom_assistant_ai_agents`; create the locale before the Q-in-Connect intent.
- `build_on_apply` and `build_locale_on_apply` add apply time while Lex builds — expect several minutes per locale.
- Wire the Lex bot into your Connect chat flow separately (e.g. `GetUserInput` / `ConnectParticipantWithLexBot`).
