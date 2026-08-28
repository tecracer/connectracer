# ==============================================================================
# Minimal — build a locale once everything in it exists
# ==============================================================================
resource "connectracer_lexv2models_bot_locale_build" "minimal" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
}

# ==============================================================================
# Standard — rebuild whenever the locale's contents change
# ==============================================================================
# Anything in triggers forces a replacement, and a replacement is a fresh build. Reference
# whatever the built locale has to reflect: ids alone catch created and destroyed objects,
# values catch edits to objects that keep their id.
resource "connectracer_lexv2models_bot_locale_build" "consent" {
  bot_id      = aws_lexv2models_bot.consent.id
  bot_version = "DRAFT"
  locale_id   = "de_DE"

  triggers = {
    slot_type_id = aws_lexv2models_slot_type.consent_answer.slot_type_id
    slot_id      = aws_lexv2models_slot.answer.slot_id
    yes_values   = join(",", var.consent_yes_utterances)
    no_values    = join(",", var.consent_no_utterances)
  }

  depends_on = [connectracer_lexv2models_intent_slot_priorities.recording_consent]
}

# ==============================================================================
# One build per locale
# ==============================================================================
# BuildBotLocale takes a single locale, so a multi-locale bot needs one of these each.
resource "connectracer_lexv2models_bot_locale_build" "per_locale" {
  for_each = toset(["de_DE", "en_US"])

  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = each.value

  triggers = {
    intent_id = "IIIII11111"
  }
}
