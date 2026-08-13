# connectracer_q_in_connect_intent

Manages the built-in `AMAZON.QInConnectIntent` on a Lex V2 bot, linking it to an Amazon Q in Connect assistant.

## build_locale_on_apply (v0.5.0+)

Creating or updating a Q-in-Connect intent changes the Lex locale draft. Set `build_locale_on_apply = true` to rebuild the locale after apply and wait until it reaches `Built`.

```hcl
resource "connectracer_q_in_connect_intent" "q_in_connect" {
  bot_id      = var.lex_bot_id
  bot_version = "DRAFT"
  locale_id   = "en_US"

  name                    = "AmazonQinConnect"
  description             = "Routes chat to Amazon Q in Connect"
  parent_intent_signature = "AMAZON.QInConnectIntent"

  fulfillment_code_hook {
    enabled = false
    active  = true
  }

  q_in_connect_intent_configuration {
    q_in_connect_assistant_configuration {
      assistant_arn = connectracer_wisdom_assistant.assistant.id
    }
  }

  build_locale_on_apply = true
}
```

See [`resource.tf`](./resource.tf) for minimal and full examples (assistant created in the same config).

For the end-to-end chat self-service stack, see [`../../chat-self-service/`](../../chat-self-service/).
