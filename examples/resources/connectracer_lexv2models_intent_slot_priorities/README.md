# connectracer_lexv2models_intent_slot_priorities

Sets the slot elicitation order on an existing Amazon Lex V2 intent, from a resource of its own.

## Why this is not an argument on the intent

An `aws_lexv2models_slot` is created with the id of the intent it belongs to, and Lex refuses to build a locale whose intent has slots without priorities. Declaring the priorities on the intent closes the loop:

```
intent -> slot_priority.slot_id -> slot -> intent_id -> intent
```

Terraform reports that as `Cycle: aws_lexv2models_slot.x, <intent>.y`. Splitting the priorities out is the same shape as `aws_security_group_rule` next to `aws_security_group`: the association becomes its own resource so the graph stays acyclic.

Without it the locale build fails with:

```
Slot ids [answer] in intent RecordingConsent don't define a slot priority.
Update the intent to add a priority to these slots.
```

## Use it with `connectracer_lexv2models_intent`

Not with `aws_lexv2models_intent`. That resource carries `slot_priority` in its own schema, so it plans the priorities away on every refresh while this resource plans them back, and the two never converge. `connectracer_lexv2models_intent` does not model slot priorities and carries them over on every update, which is what makes the pair correct without a `lifecycle` exception.

## Read-modify-write

`UpdateIntent` replaces the entire intent, so this resource re-fetches it first (`DescribeIntent`) and sends every other field back unchanged: `IntentName`, `Description`, `ParentIntentSignature`, `SampleUtterances`, `DialogCodeHook`, `FulfillmentCodeHook`, `IntentConfirmationSetting`, `IntentClosingSetting`, `InputContexts`, `OutputContexts`, `KendraConfiguration`, `InitialResponseSetting` and `QnAIntentConfiguration`. Anything omitted would be silently dropped from the live intent.

That is the same pattern `connectracer_connect_ai_tool` and `connectracer_connect_security_profile_flow_module` use, and it carries the same caveat: two resources writing the same intent in parallel race each other. Keep one per intent.

`Delete` clears the priorities again and tolerates an intent that is already gone, since the intent is normally destroyed in the same apply.

## Example Usage

```hcl
resource "connectracer_lexv2models_intent_slot_priorities" "recording_consent" {
  bot_id      = aws_lexv2models_bot.consent.id
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  intent_id   = connectracer_lexv2models_intent.recording_consent.intent_id

  slot_priority {
    priority = 1
    slot_id  = aws_lexv2models_slot.answer.slot_id
  }
}
```

## Related resources

| Resource | Role |
|----------|------|
| `connectracer_lexv2models_bot_locale_build` | Builds the locale once the priorities are set |
| `connectracer_lexv2models_bot_locale` | Creates the locale itself |

## Import

Format `bot_id/bot_version/locale_id/intent_id`, see `import.sh`.
