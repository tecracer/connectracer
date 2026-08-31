# connectracer_lexv2models_intent

Manages an Amazon Lex V2 intent without deleting the parts of it that other resources own.

## Why not `aws_lexv2models_intent`

`UpdateIntent` replaces the entire intent. A resource that models a field therefore deletes that field whenever the configuration omits it, and the official `aws_lexv2models_intent` models `slot_priority` that way.

Slot priorities cannot be declared on the intent, because an `aws_lexv2models_slot` is created with its intent's id and the reference back would be a cycle. They have to come from `connectracer_lexv2models_intent_slot_priorities`. Combine that with the official intent resource and the two never converge:

```
priorities exist  -> aws_lexv2models_intent plans:  - slot_priority
priorities gone   -> the priorities resource plans: + slot_priority
```

Every plan reports a change, every apply flips the state, and whichever resource runs last wins. `lifecycle { ignore_changes = [slot_priority] }` silences it, but a configuration that is only correct if you already knew to add that is a trap.

This resource models name, description, parent signature and sample utterances. Everything else on the intent is read back before every update and sent again unchanged:

`SlotPriorities`, `DialogCodeHook`, `FulfillmentCodeHook`, `IntentConfirmationSetting`, `IntentClosingSetting`, `InputContexts`, `OutputContexts`, `KendraConfiguration`, `InitialResponseSetting`, `QnAIntentConfiguration`

So whatever set those keeps them, and no `lifecycle` exception is needed.

## Sample utterances

A locale needs at least one custom intent with an utterance. Without one the build fails with:

```
The locale 'en_US' doesn't have any utterances.
A locale must have at least one custom intent with a valid utterance.
```

For an intent whose whole answer is a single slot value, make the utterance the slot itself (`{answer}`). Listing the words at intent level instead matches the intent directly, skips slot elicitation, and leaves the slot empty.

## Reserved names

`YesIntent` and `NoIntent` are reserved. Creating them succeeds and the locale build then fails with `The intent name YesIntent matches a reserved intent name`.

## Example Usage

```hcl
resource "connectracer_lexv2models_intent" "recording_consent" {
  bot_id      = aws_lexv2models_bot.consent.id
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  name        = "RecordingConsent"
  description = "Asks whether the call may be recorded"

  sample_utterance {
    utterance = "{answer}"
  }
}
```

## Related resources

| Resource | Role |
|----------|------|
| `connectracer_lexv2models_intent_slot_priorities` | Sets the slot elicitation order, which cannot live on the intent |
| `connectracer_lexv2models_bot_locale_build` | Builds the locale once intent, slots and slot types exist |

## Import

Format `bot_id/bot_version/locale_id/intent_id`, see `import.sh`.
