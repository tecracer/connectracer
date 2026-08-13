# connectracer_lexv2models_bot_locale

Manages a Lex V2 bot locale with settings not available in the official AWS provider (unified speech, generative AI, audio filler, and more).

## build_on_apply (v0.5.0+)

Lex locales must be built before they can serve traffic. Set `build_on_apply = true` to run `BuildBotLocale` automatically after create and update; Terraform waits until `bot_locale_status` is `Built`.

```hcl
resource "connectracer_lexv2models_bot_locale" "en_us" {
  bot_id      = var.lex_bot_id
  bot_version = "DRAFT"
  locale_id   = "en_US"

  nlu_intent_confidence_threshold = 0.4
  build_on_apply                  = true
}
```

Pair this with `connectracer_q_in_connect_intent` and `build_locale_on_apply = true` when you add or change Q-in-Connect intents on the same locale.

See [`resource.tf`](./resource.tf) for minimal, unified-speech, generative-AI, and full production examples.
