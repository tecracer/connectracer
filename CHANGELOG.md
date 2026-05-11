# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.9] - 2026-05-11
### Added 
  - connectracer_connect_ai_prompt
  - 
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
