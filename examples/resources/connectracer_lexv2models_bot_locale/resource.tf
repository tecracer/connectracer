# ==============================================================================
# Minimal — required fields only
# ==============================================================================
resource "connectracer_lexv2models_bot_locale" "minimal" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"

  nlu_intent_confidence_threshold = 0.4
}

# ==============================================================================
# Standard — voice + NLU threshold
# Equivalent to what the official aws_lexv2models_bot_locale resource supports.
# ==============================================================================
resource "connectracer_lexv2models_bot_locale" "standard" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  description = "English (US) locale for the customer service bot"

  nlu_intent_confidence_threshold = 0.4

  voice_settings {
    voice_id = "Joanna"
    engine   = "neural"
  }
}

# ==============================================================================
# Unified speech — foundation model handles both recognition and synthesis.
# This is the key feature missing from the official provider.
# ==============================================================================
resource "connectracer_lexv2models_bot_locale" "unified_speech" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  description = "English locale with foundation-model-based speech"

  nlu_intent_confidence_threshold = 0.4
  speech_detection_sensitivity    = "HighNoiseTolerance"

  unified_speech_settings {
    speech_foundation_model {
      model_arn = "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-sonic-v1:0"
      voice_id  = "tiffany"
    }
  }
}

# ==============================================================================
# Deepgram — third-party speech recognition via Secrets Manager
# ==============================================================================
resource "connectracer_lexv2models_bot_locale" "deepgram" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  description = "English locale using Deepgram for transcription"

  nlu_intent_confidence_threshold = 0.4

  speech_recognition_settings {
    speech_model_preference = "Deepgram"

    speech_model_config {
      deepgram_config {
        # ARN of the Secrets Manager secret containing the Deepgram API key
        api_token_secret_arn = "arn:aws:secretsmanager:us-east-1:123456789012:secret:deepgram-api-key-AbCdEf"
        model_id             = "nova-2"
      }
    }
  }
}

# ==============================================================================
# Generative AI — build-time helpers + runtime NLU and slot improvement
# ==============================================================================
resource "connectracer_lexv2models_bot_locale" "generative_ai" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  description = "English locale with full generative AI capability"

  nlu_intent_confidence_threshold = 0.4

  voice_settings {
    voice_id = "Matthew"
    engine   = "generative"
  }

  generative_ai_settings {
    buildtime_settings {
      descriptive_bot_builder {
        enabled = true

        bedrock_model_specification {
          model_arn = "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0"
        }
      }

      sample_utterance_generation {
        enabled = true

        bedrock_model_specification {
          model_arn = "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0"
        }
      }
    }

    runtime_settings {
      nlu_improvement {
        enabled           = true
        assisted_nlu_mode = "Primary"

        intent_disambiguation_settings {
          enabled                       = true
          custom_disambiguation_message = "Could you clarify what you mean?"
          max_disambiguation_intents    = 3
        }
      }

      slot_resolution_improvement {
        enabled = true

        bedrock_model_specification {
          model_arn = "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0"
        }
      }
    }
  }
}

# ==============================================================================
# Audio filler — plays a sound while Lex processes the utterance
# ==============================================================================
resource "connectracer_lexv2models_bot_locale" "audio_filler" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  description = "English locale with audio filler to reduce perceived latency"

  nlu_intent_confidence_threshold = 0.4

  unified_speech_settings {
    speech_foundation_model {
      model_arn = "arn:aws:bedrock:eu-central-1::foundation-model/amazon.nova-2-sonic-v1:0"
    }
  }

  audio_filler_settings {
    enabled    = true
    audio_type = "MELODY_CHIPPER_CHIME"

    # Wait 2500 ms after user finishes speaking before playing the filler
    start_delay_in_milliseconds = 2500

    # Keep playing for at least 3000 ms even if the response is ready sooner
    minimum_play_duration_in_milliseconds = 3000

    # Insert a 500 ms gap between filler end and bot response
    response_delivery_delay_in_milliseconds = 500
  }
}

resource "connectracer_lexv2models_bot_locale" "full" {
  bot_id      = "ABCDE12345"
  bot_version = "DRAFT"
  locale_id   = "en_US"
  description = "Production locale — all settings enabled"

  nlu_intent_confidence_threshold = 0.5
  speech_detection_sensitivity    = "Default"

  # Foundation-model-based speech (not available in the official provider)
  unified_speech_settings {
    speech_foundation_model {
      model_arn = "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-sonic-v1:0"
      voice_id  = "tiffany"
    }
  }

  generative_ai_settings {
    buildtime_settings {
      descriptive_bot_builder {
        enabled = true

        bedrock_model_specification {
          model_arn     = "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0"
          custom_prompt = "You are a helpful assistant. Describe bot intents clearly and concisely."
          trace_status  = "DISABLED"

          guardrail {
            identifier = "my-guardrail-id"
            version    = "1"
          }
        }
      }

      sample_utterance_generation {
        enabled = true

        bedrock_model_specification {
          model_arn = "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0"
        }
      }
    }

    runtime_settings {
      nlu_improvement {
        enabled           = true
        assisted_nlu_mode = "Primary"

        intent_disambiguation_settings {
          enabled                       = true
          custom_disambiguation_message = "I found a few options. Which did you mean?"
          max_disambiguation_intents    = 3
        }
      }

      slot_resolution_improvement {
        enabled = true

        bedrock_model_specification {
          model_arn = "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0"
        }
      }
    }
  }
}

# ==============================================================================
# Outputs — reference computed values from other resources
# ==============================================================================
output "unified_speech_locale_status" {
  description = "Build status of the unified-speech locale"
  value       = connectracer_lexv2models_bot_locale.unified_speech.bot_locale_status
}

output "full_locale_id" {
  description = "Composite resource ID (bot_id/bot_version/locale_id) of the full locale"
  value       = connectracer_lexv2models_bot_locale.full.id
}
