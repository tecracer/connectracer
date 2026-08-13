# ──────────────────────────────────────────────────────────────────────────────
# Minimal example — Q in Connect intent with only the required fields
# ──────────────────────────────────────────────────────────────────────────────

resource "connectracer_q_in_connect_intent" "q_in_connect_intent" {
  bot_id      = "ABCDE12345" # 10-char Lex V2 bot ID
  bot_version = "DRAFT"
  locale_id   = "en_US"

  name                    = "AmazonQinConnect"
  description             = "AmazonQinConnect"
  parent_intent_signature = "AMAZON.QInConnectIntent"

  fulfillment_code_hook {
    enabled = false
    active  = true
  }

  q_in_connect_intent_configuration {
    q_in_connect_assistant_configuration {
      assistant_arn = "arn:aws:wisdom:eu-central-1:123456789012:assistant/f8eb5589-1c15-42a1-a0c7-44a4296a076d"
    }
  }
}

# ──────────────────────────────────────────────────────────────────────────────
# Full example — referencing a Q Connect assistant created in the same config
# ──────────────────────────────────────────────────────────────────────────────

# The Amazon Q in Connect assistant that will be linked to the Lex intent
resource "connectracer_wisdom_assistant" "assistant" {
  name = "my-connect-assistant"
  type = "AGENT"

  tags = {
    Environment = "production"
    ManagedBy   = "terraform"
  }
}

# Lex V2 bot locale data — the bot and locale must already exist;
# bot_id and locale_id are typically imported from existing infrastructure.
resource "connectracer_q_in_connect_intent" "full" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"

  name                    = "AmazonQinConnect"
  description             = "Routes conversation to Amazon Q in Connect"
  parent_intent_signature = "AMAZON.QInConnectIntent"

  # For the built-in AMAZON.QInConnectIntent pattern the fulfillment Lambda
  # is not used (enabled = false) but the hook must be active so that Lex
  # picks up the post-fulfillment success response from Q in Connect.
  fulfillment_code_hook {
    enabled = false
    active  = true
  }

  q_in_connect_intent_configuration {
    q_in_connect_assistant_configuration {
      assistant_arn = connectracer_wisdom_assistant.assistant.id
    }
  }

  # When true, connectracer runs BuildBotLocale for this intent's locale
  # after create/update and waits until the locale reaches Built (v0.5.0+).
  build_locale_on_apply = true
}
