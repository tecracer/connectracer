#!/bin/bash
# Import an existing Lex V2 intent into Terraform state.
# Format: bot_id/bot_version/locale_id/intent_id

terraform import connectracer_lexv2models_intent.recording_consent \
  "ABCDE12345/DRAFT/de_DE/IIIII11111"
