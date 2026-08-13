terraform {
  required_version = ">= 1.0"

  required_providers {
    connectracer = {
      source  = "tecracer/connectracer"
      version = ">= 0.5.0"
    }
  }
}

# ------------------------------------------------------------------------------
# Variables — override via terraform.tfvars
# ------------------------------------------------------------------------------

variable "lex_bot_id" {
  type        = string
  description = "Existing Lex V2 bot ID (10 characters)."
}

variable "connect_instance_arn" {
  type        = string
  description = "ARN of the Amazon Connect instance."
}

variable "orchestration_prompt_id" {
  type        = string
  description = "Orchestration AI prompt ID for the ORCHESTRATION agent."
}

variable "self_service_answer_prompt_id" {
  type        = string
  description = "Answer-generation prompt ID for the optional SELF_SERVICE agent."
  default     = null
}

variable "primary_locale_id" {
  type        = string
  default     = "en_US"
  description = "Lex locale and orchestration agent locale."
}

# ------------------------------------------------------------------------------
# Q in Connect assistant
# ------------------------------------------------------------------------------

resource "connectracer_wisdom_assistant" "assistant" {
  name = "chat-self-service"
  type = "AGENT"

  tags = {
    Example = "chat-self-service"
  }
}

# ------------------------------------------------------------------------------
# Orchestration agent + escalation tool
# ------------------------------------------------------------------------------

resource "connectracer_connect_ai_agent" "orchestration" {
  assistant_id      = connectracer_wisdom_assistant.assistant.id
  name              = "orchestration"
  type              = "ORCHESTRATION"
  visibility_status = "PUBLISHED"
  create_version    = true

  orchestration_configuration {
    orchestration_ai_prompt_id = var.orchestration_prompt_id
    connect_instance_arn       = var.connect_instance_arn
    locale                     = var.primary_locale_id
  }
}

resource "connectracer_connect_ai_tool" "escalate" {
  assistant_id = connectracer_connect_ai_agent.orchestration.assistant_id
  ai_agent_id  = connectracer_connect_ai_agent.orchestration.id

  tool_name   = "Escalate"
  tool_type   = "RETURN_TO_CONTROL"
  title       = "Escalate to Human Agent"
  description = "Return control to the contact flow for human escalation."

  instruction = "Call when the customer requests a human or the issue cannot be resolved."

  input_schema_json = jsonencode({
    type = "object"
    properties = {
      escalationReason  = { type = "string" }
      escalationSummary = { type = "string" }
      sentiment         = { type = "string", enum = ["POSITIVE", "NEUTRAL", "NEGATIVE", "MIXED"] }
    }
    required = ["escalationReason", "sentiment"]
  })
}

# ------------------------------------------------------------------------------
# Assign orchestration agent for Connect.SelfService (required for chat)
# ------------------------------------------------------------------------------

resource "connectracer_wisdom_assistant_ai_agents" "orchestration" {
  assistant_id = connectracer_wisdom_assistant.assistant.id

  ai_agent_configuration = {
    ORCHESTRATION = "${connectracer_connect_ai_agent.orchestration.id}:$LATEST"
  }

  orchestrator_use_cases = {
    ORCHESTRATION = "Connect.SelfService"
  }

  depends_on = [connectracer_connect_ai_tool.escalate]
}

# ------------------------------------------------------------------------------
# Optional SELF_SERVICE agent (agent-assist / knowledge lookup)
# ------------------------------------------------------------------------------

resource "connectracer_connect_ai_agent" "self_service" {
  count = var.self_service_answer_prompt_id != null ? 1 : 0

  assistant_id      = connectracer_wisdom_assistant.assistant.id
  name              = "self-service"
  type              = "SELF_SERVICE"
  visibility_status = "PUBLISHED"
  create_version    = true

  self_service_configuration {
    self_service_answer_generation_ai_prompt_id = var.self_service_answer_prompt_id
  }
}

resource "connectracer_wisdom_assistant_ai_agents" "self_service" {
  count = var.self_service_answer_prompt_id != null ? 1 : 0

  assistant_id = connectracer_wisdom_assistant.assistant.id

  ai_agent_configuration = {
    SELF_SERVICE = "${connectracer_connect_ai_agent.self_service[0].id}:$LATEST"
  }
}

# ------------------------------------------------------------------------------
# Lex locale + Q-in-Connect intent (with automatic locale build)
# ------------------------------------------------------------------------------

resource "connectracer_lexv2models_bot_locale" "primary" {
  bot_id      = var.lex_bot_id
  bot_version = "DRAFT"
  locale_id   = var.primary_locale_id

  nlu_intent_confidence_threshold = 0.4
  build_on_apply                  = true
}

resource "connectracer_q_in_connect_intent" "q_in_connect" {
  bot_id      = var.lex_bot_id
  bot_version = "DRAFT"
  locale_id   = var.primary_locale_id

  name                    = "AmazonQinConnect"
  description             = "Amazon Q in Connect chat self-service"
  parent_intent_signature = "AMAZON.QInConnectIntent"

  fulfillment_code_hook {
    enabled = false
    active  = true
  }

  q_in_connect_intent_configuration {
    q_in_connect_assistant_configuration {
      assistant_arn = connectracer_wisdom_assistant.assistant.id
    }
  }

  build_locale_on_apply = true

  depends_on = [
    connectracer_lexv2models_bot_locale.primary,
    connectracer_wisdom_assistant_ai_agents.orchestration,
  ]
}

# ------------------------------------------------------------------------------
# Outputs
# ------------------------------------------------------------------------------

output "assistant_id" {
  value       = connectracer_wisdom_assistant.assistant.id
  description = "Q in Connect assistant ID — use in Connect integration association."
}

output "orchestration_agent_id" {
  value       = connectracer_connect_ai_agent.orchestration.id
  description = "Orchestration AI agent ID."
}

output "lex_locale_status" {
  value       = connectracer_lexv2models_bot_locale.primary.bot_locale_status
  description = "Lex locale build status after apply."
}
