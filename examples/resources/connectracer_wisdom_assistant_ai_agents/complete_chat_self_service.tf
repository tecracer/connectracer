# Example: wire orchestration + self-service agents to a Q in Connect assistant.
# This file is not used for registry doc generation — see resource.tf for the
# minimal snippets shown on the resource documentation page.

resource "connectracer_wisdom_assistant" "assistant" {
  name = "chat-self-service-assistant"
  type = "AGENT"

  tags = {
    Environment = "production"
  }
}

resource "connectracer_connect_ai_agent" "self_service" {
  assistant_id      = connectracer_wisdom_assistant.assistant.id
  name              = "self-service-agent"
  type              = "SELF_SERVICE"
  visibility_status = "PUBLISHED"
  create_version    = true

  self_service_configuration {
    self_service_answer_generation_ai_prompt_id = var.self_service_answer_prompt_id
    self_service_pre_processing_ai_prompt_id    = var.self_service_preprocessing_prompt_id
  }
}

resource "connectracer_connect_ai_agent" "orchestration" {
  assistant_id      = connectracer_wisdom_assistant.assistant.id
  name              = "orchestration-agent"
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
  description = "Transfer to a human agent when self-service cannot resolve the issue."

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

  depends_on = [connectracer_connect_ai_agent.orchestration]
}

resource "connectracer_wisdom_assistant_ai_agents" "orchestration" {
  assistant_id = connectracer_wisdom_assistant.assistant.id

  ai_agent_configuration = {
    ORCHESTRATION = "${connectracer_connect_ai_agent.orchestration.id}:$LATEST"
  }

  orchestrator_use_cases = {
    ORCHESTRATION = "Connect.SelfService"
  }

  depends_on = [
    connectracer_connect_ai_agent.orchestration,
    connectracer_connect_ai_tool.escalate,
  ]
}

resource "connectracer_wisdom_assistant_ai_agents" "self_service" {
  assistant_id = connectracer_wisdom_assistant.assistant.id

  ai_agent_configuration = {
    SELF_SERVICE = "${connectracer_connect_ai_agent.self_service.id}:$LATEST"
  }

  depends_on = [connectracer_connect_ai_agent.self_service]
}

variable "self_service_answer_prompt_id" {
  type        = string
  description = "Answer-generation prompt ID for the SELF_SERVICE agent."
}

variable "self_service_preprocessing_prompt_id" {
  type        = string
  description = "Pre-processing prompt ID for the SELF_SERVICE agent."
}

variable "orchestration_prompt_id" {
  type        = string
  description = "Orchestration prompt ID for the ORCHESTRATION agent."
}

variable "connect_instance_arn" {
  type        = string
  description = "ARN of the Amazon Connect instance."
}

variable "primary_locale_id" {
  type        = string
  default     = "en_US"
  description = "Primary locale for the orchestration agent."
}
