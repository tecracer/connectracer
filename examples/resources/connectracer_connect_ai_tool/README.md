# connectracer_connect_ai_tool

Manages a single tool configuration within an Amazon Q in Connect **Orchestration AI Agent**.

Tools define capabilities the AI model can invoke during a conversation — for example escalating to a human agent (`RETURN_TO_CONTROL`), calling an MCP server (`MODEL_CONTEXT_PROTOCOL`), or injecting a constant value (`CONSTANT`).

Because the AWS API stores tools as a list on the agent, this resource performs a **read-modify-write** on every create, update, and delete. Multiple `connectracer_connect_ai_tool` resources can be attached to the same agent; they coexist safely as long as they are applied sequentially (Terraform's default).

The parent agent must be of type `ORCHESTRATION` and must already exist before any tool resources are created for it.

## Example Usage

### Workshop escalation tool (RETURN_TO_CONTROL)

```hcl
resource "connectracer_connect_ai_agent" "orchestration" {
  assistant_id      = var.assistant_id
  name              = "orchestration-agent"
  type              = "ORCHESTRATION"
  visibility_status = "PUBLISHED"
  create_version    = true

  orchestration_configuration {
    orchestration_ai_prompt_id = var.orchestration_prompt_id
    connect_instance_arn       = var.connect_instance_arn
  }
}

resource "connectracer_connect_ai_tool" "escalate" {
  assistant_id = connectracer_connect_ai_agent.orchestration.assistant_id
  ai_agent_id  = connectracer_connect_ai_agent.orchestration.id

  tool_name   = "Escalate"
  tool_type   = "RETURN_TO_CONTROL"
  title       = "Escalate to Human Agent"
  description = "Transfer to a human agent when the AI cannot resolve the issue."

  instruction = "Call this tool whenever the customer requests a human, or when the issue cannot be resolved through self-service."

  input_schema_json = jsonencode({
    type = "object"
    properties = {
      escalationReason  = { type = "string", description = "Why the contact is being escalated." }
      escalationSummary = { type = "string", description = "Conversation summary for the receiving agent." }
      intent            = { type = "string", description = "Primary customer intent." }
      sentiment         = { type = "string", enum = ["POSITIVE", "NEUTRAL", "NEGATIVE", "MIXED"] }
    }
    required = ["escalationReason", "sentiment"]
  })
}
```

### MCP tool with user confirmation

```hcl
resource "connectracer_connect_ai_tool" "cancel_order" {
  assistant_id = var.assistant_id
  ai_agent_id  = var.orchestration_agent_id

  tool_name                  = "CancelOrder"
  tool_type                  = "MODEL_CONTEXT_PROTOCOL"
  description                = "Cancel an existing order on behalf of the customer."
  user_confirmation_required = true

  input_schema_json = jsonencode({
    type       = "object"
    properties = {
      orderId = { type = "string", description = "The order ID to cancel." }
      reason  = { type = "string", description = "Reason for cancellation." }
    }
    required = ["orderId"]
  })
}
```

## Argument Reference

### Required

* `assistant_id` — The identifier of the Amazon Q in Connect assistant. Changing this forces a new resource.
* `ai_agent_id` — The identifier of the Orchestration AI Agent. Changing this forces a new resource.
* `tool_name` — The unique name of the tool within the agent. Used as the key for all read-modify-write operations. Changing this forces a new resource.
* `tool_type` — The tool type. Valid values: `RETURN_TO_CONTROL`, `MODEL_CONTEXT_PROTOCOL`, `CONSTANT`.

### Optional

* `title` — A short human-readable title.
* `description` — Description shown to the AI model as part of the tool contract.
* `input_schema_json` — JSON string describing the input parameters (JSON Schema format). Use `jsonencode()` to construct this from a Terraform map.
* `output_schema_json` — JSON string describing the tool output.
* `instruction` — Free-form text telling the model when and how to call this tool.
* `instruction_examples` — List of example strings illustrating usage.
* `user_confirmation_required` — If `true`, the user must confirm the action before the tool executes.

## Attribute Reference

* `id` — Composite identifier: `assistant_id/ai_agent_id/tool_name`.

## Import

Import a tool that already exists on an agent:

```shell
terraform import connectracer_connect_ai_tool.escalate \
  "12345678-1234-1234-1234-123456789012/60dfa473-aaaa-bbbb-cccc-dddddddddddd/Escalate"
```

## Concurrency Note

Each CRUD operation reads the agent's full tool list, applies the change, and writes the list back in a single `UpdateAIAgent` call. If two tool resources belonging to the same agent are applied **in parallel** (e.g. via `-parallelism=N` with `N > 1`) a last-write-wins race condition can occur. Terraform's default sequential apply avoids this. If you use high parallelism, add explicit `depends_on` chains between tools on the same agent.
