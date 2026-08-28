# connectracer_lexv2models_bot_locale_build

Builds an Amazon Lex V2 bot locale and waits until it reaches `Built`.

## Why this is not an argument on the locale

`connectracer_lexv2models_bot_locale` has `build_on_apply`, and it is the right thing for a locale whose contents come from the same resource. It cannot serve a locale that has intents, slots and slot types declared against it, because it runs when the locale is created, which is before any of them exist. Making the locale depend on its own slots to reorder that is a cycle.

So the build becomes a step of its own, placed after everything it has to compile. Without it the locale stays unbuilt, the bot answers nothing, and the only symptom is at runtime.

## Triggers

Every attribute forces replacement, and a replacement is a fresh build. `triggers` is where that gets useful: put whatever the built locale has to reflect into it.

Ids alone only catch objects being created or destroyed. An edit that keeps the id, such as adding a synonym to a slot type, needs the value itself in the map:

```hcl
triggers = {
  slot_type_id = aws_lexv2models_slot_type.consent_answer.slot_type_id
  yes_values   = join(",", var.consent_yes_utterances)
}
```

`Read` deliberately does nothing. The locale's live status says whether it is built *now*, not whether it was built from what this resource last saw, so reconciling against it would only produce diffs nobody can act on.

`Delete` also does nothing. A build is an event, not an object that can be removed.

## Ordering

`BuildBotLocale` fails on an intent whose slots have no priorities, so `connectracer_lexv2models_intent_slot_priorities` belongs before this one. Reference it with `depends_on` when nothing else already orders them.

`BuildBotLocale` takes one locale, so a bot with several locales needs one of these per locale.

## Example Usage

```hcl
resource "connectracer_lexv2models_bot_locale_build" "consent" {
  bot_id      = aws_lexv2models_bot.consent.id
  bot_version = "DRAFT"
  locale_id   = "de_DE"

  triggers = {
    slot_id = aws_lexv2models_slot.answer.slot_id
  }

  depends_on = [connectracer_lexv2models_intent_slot_priorities.recording_consent]
}
```

## Related resources

| Resource | Role |
|----------|------|
| `connectracer_lexv2models_bot_locale` | Creates the locale, including `build_on_apply` for the simple case |
| `connectracer_lexv2models_intent_slot_priorities` | Has to run before the build when an intent has slots |

## Import

Not supported. There is nothing to import: the resource records that a build was run, which no API reports.
