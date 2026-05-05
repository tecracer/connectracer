# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


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
- **Automatic Tag Management**: All Wisdom and AppIntegrations resources automatically add `AmazonConnectEnabled = "True"` tag
- **Custom Plan Modifier**: Required tags are visible in Terraform plans before applying
- **AWS SDK v2 Integration**: Modern, maintained Go SDK for AWS services
- **Shared Tag Utilities**: Consistent tag handling across all resources via `tags_utils.go`
- **Comprehensive Documentation**: Full examples including S3, KMS, and EventBridge configuration
- **Build Automation**: Taskfile with `build`, `install-dev`, and `clean` commands

### Implementation Details
- Amazon Connect Customer S3 Knowledge Base fully implemented
- Resources use Terraform Plugin Framework for modern provider development
- Tag plan modifiers prevent "inconsistent result" errors
- Complete CRUD operations with import support for all resources
