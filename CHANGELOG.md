# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [Unreleased]
### Added
- `connectracer_connect_flow_module_tool`: manages a Flow Module with `ExternalInvocationConfiguration` enabled, so it can be invoked outside of a flow as a tool (e.g. Amazon Connect's "module as tool" support for Q in Connect orchestrator AI agents). Owns the full flow module (content, name, description, settings, tags) since `ExternalInvocationConfiguration` is create-only on the AWS API. `description` is required and a version is released automatically via `CreateContactFlowModuleVersion` on creation and on every `content`/`settings` change — confirmed against a live instance that a tool module without a description or without a released version silently fails to appear as grantable in any security profile, even though `ExternalInvocationConfiguration.Enabled` already reads back as `true`. New computed `mcp_tool_id` attribute (`aws_custom_flows__<flow_module_id>_<version>`) — the actual value `UpdateAIAgent` expects as a `MODEL_CONTEXT_PROTOCOL` tool's `tool_id`, reverse-engineered from a console-created tool module since it's undocumented and the raw flow module `id` is rejected with "not found in MCP tools".
- `connectracer_connect_security_profile_flow_module`: grants a Security Profile permission to invoke a Flow Module as a tool (`AllowedFlowModules` on `UpdateSecurityProfile`), with a safe read-modify-write so it coexists with `aws_connect_security_profile` managing the rest of the security profile
- `connectracer_connect_ai_agent_security_profile`: associates a Security Profile with a Q in Connect AI Agent (`AssociateSecurityProfiles`/`DisassociateSecurityProfiles`, `EntityType=AI_AGENT`) — required before an agent can invoke any governed tool (Flow Module tools, AgentCore/MCP tools, out-of-the-box tools); `RETURN_TO_CONTROL` tools don't need it
- `connectracer_connect_ai_tool`: new optional `tool_id` attribute — required for `MODEL_CONTEXT_PROTOCOL` tools (`UpdateAIAgent` otherwise rejects them with "require toolId as input for MCP identifier"); for a flow module tool this is the flow module's own `id`. Create/Update retry on eventual consistency when `UpdateAIAgent` reports the `tool_id` as "not found in MCP tools" shortly after the security profile grant that makes it visible to QConnect.

### Fixed
- `connectracer_connect_ai_tool`: preserve `description`/`instruction` in state when `GetAIAgent` omits them on read — confirmed for a `MODEL_CONTEXT_PROTOCOL` tool backed by a flow module, where a just-applied description/instruction read back as absent on the very next refresh, causing a perpetual plan diff
- Examples and unit tests for all three new resources

## [0.5.0] - 2026-08-13
### Added
- `connectracer_wisdom_assistant_ai_agents`: optional `orchestrator_use_cases` map (required when assigning an `ORCHESTRATION` agent; e.g. `Connect.SelfService`)
- `connectracer_lexv2models_bot_locale`: optional `build_on_apply` to run `BuildBotLocale` after create/update
- `connectracer_q_in_connect_intent`: optional `build_locale_on_apply` to build the locale after intent create/update
- Shared Lex locale build helper (`provider/lex_bot_locale_build.go`)
- Provider documentation for `wisdom_assistant_ai_agents`, `lexv2models_bot_locale`, `q_in_connect_intent`, and other previously undocumented resources
- Example and unit tests for orchestrator use cases and AI agent qualifier handling

### Fixed
- `connectracer_connect_ai_agent`: with `create_version = true`, publish a new version only when agent configuration actually changes; preserve `version_number` in state when AWS omits it on read
- `connectracer_connect_ai_prompt`: preserve `version_number` in state when AWS omits it on read
- `connectracer_wisdom_assistant_ai_agents`: Read compares AI agent assignments semantically so equivalent qualifiers (e.g. `$LATEST` vs `:7`) no longer cause perpetual plan drift, while real console changes (different agent, version, or removal) are still detected

## [0.4.7] - 2026-07-11
### Fix
- Flows will be update, not replaced
- connectracer_connect_view.agent_screen_pop update errors
  - `version` inconsistency**
  - `tags` inconsistency**

## [0.2.5] - 2026-05-29
### Added
  - Tools and Orchestration agent as seperate resources 
  
## [0.2.4] - 2026-05-29
### Added
  - View Resource

## [0.2.2] - 2026-05-27
### Fix 
  - readAndPopulateModel: module.ai_agents.connectracer_connect_ai_agent Version and modified_time problem
 
## [0.1.12] - 2026-05-11
### Added 
  - connectracer_connect_ai_agent
  - 
## [0.1.11] - 2026-05-11
### Added 
  - connectracer_connect_ai_prompt
  - tested with create, update, delete for answer generation
  
## [0.1.8] - 2026-05-11
### Added
  - Approved Origins (https://docs.aws.amazon.com/connect/latest/APIReference/API_AssociateApprovedOrigin.html)

## [0.1.7] - 2026-05-07
  - Numbering skips due to deployment restrictions
### Added
  - connect rules, see https://docs.aws.amazon.com/connect/latest/APIReference/rules-api.html (not tested)

## [0.1.3] - 2026-05-07

### Fixed
- **Tag Handling with tags/tags_all Pattern**: Completely rewrote tag handling to follow the AWS provider pattern. This fixes the "Provider produced inconsistent result after apply" errors.
  - Added `tags_all` computed attribute to all resources that auto-add the `AmazonConnectEnabled` tag
  - `tags` attribute now contains only user-provided tags (Optional only, not Computed)
  - `tags_all` attribute contains all tags including provider-added tags (Computed only)
  - Resources affected: `connectracer_wisdom_assistant`, `connectracer_wisdom_assistant_association`, `connectracer_qconnect_knowledgebase`, `connectracer_appintegrations_data_integration`
  - This is a **breaking change**: If you were referencing `.tags` in outputs or data sources to get all tags including the auto-added ones, you should now use `.tags_all` instead

- **Integration Association Create Function**: Fixed `connectracer_connect_integration_association` resource errors:
  - Removed broken manual `Read` call from `Create` function
  - Fixed "Value Conversion Error: Received null value" error
  - Fixed "Provider returned invalid result object after apply" for `source_type` field
  - The Create function now properly handles optional fields that may be null (like `source_type` for WISDOM integrations)
  - All fields are now correctly populated and saved to state in a single operation

### Changed
- **Tags attribute behavior**: The `tags` attribute no longer includes provider-added tags. Use `tags_all` to access all tags including `AmazonConnectEnabled = "True"`

## [0.1.2] - 2026-05-07

### Fixed
- **Tag Plan Validation**: Removed plan modifier that was adding `AmazonConnectEnabled` tag during planning phase, which caused "Provider produced invalid plan" errors. The tag is now added only during resource creation/update and stored in state. This fixes Terraform's plan validation while maintaining automatic tag functionality.
- Note: Version 0.1.2 did not fully resolve the issue and was superseded by 0.1.3

## [0.1.1] - 2026-05-05

### Added

#### Resources
- `connectracer_wisdom_assistant` - Manage AWS Wisdom assistants for AI-powered agent assistance
- `connectracer_appintegrations_data_integration` - Manage data integrations for S3-backed knowledge bases
- `connectracer_qconnect_knowledgebase` - Manage Q Connect knowledge bases (EXTERNAL/CUSTOM types)
- `connectracer_wisdom_assistant_association` - Associate knowledge bases with assistants
- `connectracer_connect_integration_association` - Associate Wisdom resources with Connect instances

#### Data Sources
- `connectracer_wisdom_knowledge_bases` - List all Wisdom knowledge bases in AWS account
- `connectracer_wisdom_assistants` - List all Wisdom assistants
- `connectracer_qconnect_knowledgebase` - Get details of a specific knowledge base

#### Features
- **Automatic Tag Management**: All Wisdom and AppIntegrations resources automatically add `AmazonConnectEnabled = "True"` tag during creation
- **AWS SDK v2 Integration**: Modern, maintained Go SDK for AWS services
- **Shared Tag Utilities**: Consistent tag handling across all resources via `tags_utils.go`
- **Comprehensive Documentation**: Full examples including S3, KMS, and EventBridge configuration
- **Build Automation**: Taskfile with `build`, `install-dev`, and `clean` commands

### Implementation Details
- Amazon Connect Customer S3 Knowledge Base fully implemented
- Resources use Terraform Plugin Framework for modern provider development
- Tag plan modifiers prevent "inconsistent result" errors
- Complete CRUD operations with import support for all resources
