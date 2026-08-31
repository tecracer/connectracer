# ==============================================================================
# Minimal
# ==============================================================================
resource "connectracer_lexv2models_intent" "minimal" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  name        = "OrderStatus"

  sample_utterance {
    utterance = "where is my order"
  }
}

# ==============================================================================
# An intent whose whole answer is one slot value
# ==============================================================================
# The only utterance is the slot itself, so nothing matches the intent beside it and every
# answer goes through the slot's own resolution. Listing the words here instead would match
# the intent directly, skip slot elicitation, and leave the answer empty.
#
# The utterance cannot be left out either: a locale with no utterance anywhere refuses to
# build with "A locale must have at least one custom intent with a valid utterance".
resource "connectracer_lexv2models_intent" "recording_consent" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  name        = "RecordingConsent"
  description = "Asks whether the call may be recorded"

  sample_utterance {
    utterance = "{answer}"
  }
}

resource "aws_lexv2models_slot" "answer" {
  bot_id       = "ABCDE12345"
  bot_version  = "DRAFT"
  locale_id    = "de_DE"
  intent_id    = connectracer_lexv2models_intent.recording_consent.intent_id
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

# The priorities live in their own resource, since the slot is created with the intent's id
# and an intent referencing its slots would be a cycle. This intent does not model slot
# priorities and carries them over on every update, so the two never overwrite each other.
resource "connectracer_lexv2models_intent_slot_priorities" "recording_consent" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "de_DE"
  intent_id   = connectracer_lexv2models_intent.recording_consent.intent_id

  slot_priority {
    priority = 1
    slot_id  = aws_lexv2models_slot.answer.slot_id
  }
}

# ==============================================================================
# Built-in intent
# ==============================================================================
resource "connectracer_lexv2models_intent" "fallback" {
  bot_id                  = "ABCDE12345"
  bot_version             = "DRAFT"
  locale_id               = "en_US"
  name                    = "FallbackIntent"
  parent_intent_signature = "AMAZON.FallbackIntent"
}
