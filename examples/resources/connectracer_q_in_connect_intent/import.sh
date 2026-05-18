#!/bin/bash
# Import an existing Q in Connect intent into Terraform state.
# Format: bot_id/bot_version/locale_id/intent_id

terraform import connectracer_q_in_connect_intent.q_in_connect_intent \
  "ABCDE12345/DRAFT/en_US/ZYXWV98765"
