# ==============================================================================
# The cycle this resource exists for
# ==============================================================================
# aws_lexv2models_slot is created with the id of its intent, and Lex refuses to build a
# locale whose intent has slots without priorities. Declaring the priorities on the intent
# would therefore close the loop:
#
#   intent -> slot_priority -> slot -> intent_id -> intent
#
# So the intent is created without them and this resource sets them afterwards.

resource "aws_lexv2models_intent" "recording_consent" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  name        = "RecordingConsent"
  description = "Asks whether the call may be recorded"
}

resource "aws_lexv2models_slot" "answer" {
  bot_id       = "ABCDE12345"
  bot_version  = "DRAFT"
  locale_id    = "de_DE"
  intent_id    = aws_lexv2models_intent.recording_consent.intent_id
  name         = "answer"
  slot_type_id = "ZYXWV98765"

  value_elicitation_setting {
    slot_constraint = "Required"

    prompt_specification {
      max_retries     = 1
      allow_interrupt = true

      message_group {
        message {
          plain_text_message { value = "Duerfen wir das Gespraech aufzeichnen?" }
        }
      }
    }
  }
}

# ==============================================================================
# One slot
# ==============================================================================
resource "connectracer_lexv2models_intent_slot_priorities" "recording_consent" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  intent_id   = aws_lexv2models_intent.recording_consent.intent_id

  slot_priority {
    priority = 1
    slot_id  = aws_lexv2models_slot.answer.slot_id
  }
}

# ==============================================================================
# Several slots — Lex elicits them in ascending priority
# ==============================================================================
resource "connectracer_lexv2models_intent_slot_priorities" "book_appointment" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  intent_id   = "IIIII11111"

  slot_priority {
    priority = 1
    slot_id  = "SSSSS11111"
  }

  slot_priority {
    priority = 2
    slot_id  = "SSSSS22222"
  }
}

# ==============================================================================
# Build the locale afterwards
# ==============================================================================
# The priorities have to be in place before the build, otherwise it fails with
# "Slot ids [...] in intent ... don't define a slot priority".
resource "connectracer_lexv2models_bot_locale_build" "consent" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "de_DE"

  triggers = {
    slot_id = aws_lexv2models_slot.answer.slot_id
  }

  depends_on = [connectracer_lexv2models_intent_slot_priorities.recording_consent]
}
