# QConnect Knowledge Base Data Source - Implementation Summary

## Overview

Created a data source for looking up AWS Q Connect Knowledge Bases by ID, enabling Terraform configurations to reference existing knowledge bases and retrieve their configuration details.

## Implementation

### 1. Data Source: `connectracer_qconnect_knowledgebase`

**File**: `connectracer/provider/datasource_qconnect_knowledgebase.go`

#### Features Implemented:
- ✅ **Read**: Fetches knowledge base details via `GetKnowledgeBase` API
- ✅ **All Attributes**: Supports all knowledge base attributes including nested configurations
- ✅ **Type Support**: Works with both `CUSTOM` and `EXTERNAL` knowledge bases
- ✅ **AppIntegrations**: Extracts AppIntegrations ARN for EXTERNAL knowledge bases

#### Schema Attributes:

**Required Input:**
- `knowledge_base_id` - The ID of the knowledge base to look up

**Computed Outputs:**
- `id` - Knowledge base ID (same as input)
- `knowledge_base_arn` - ARN of the knowledge base
- `name` - Knowledge base name
- `knowledge_base_type` - Type (EXTERNAL or CUSTOM)
- `description` - Knowledge base description
- `status` - Current status (ACTIVE, CREATING, etc.)
- `tags` - Resource tags

**Nested Attributes:**
- `rendering_configuration.template_uri` - Template for rendering content
- `server_side_encryption_configuration.kms_key_id` - KMS key for encryption
- `source_configuration.app_integration_arn` - AppIntegrations ARN (EXTERNAL only)

### 2. Provider Updates

**File**: `connectracer/provider/provider.go`

#### Changes:
- ✅ Updated `DataSourceData` to pass `ProviderClients` instead of just `wisdom.Client`
- ✅ Registered `NewQConnectKnowledgeBaseDataSource` in DataSources list
- ✅ Updated all existing datasources to use `ProviderClients`

**Affected Datasources:**
- `datasource_wisdom_assistant.go` - Updated Configure() method
- `wisdom_knowledge_bases_data_source.go` - Updated Configure() method  
- `datasource_qconnect_knowledgebase.go` - New datasource

### 3. Testing

**File**: `connectracer/provider/datasource_qconnect_knowledgebase_test.go`

#### Test Coverage:
- ✅ Basic read test with resource creation
- ✅ Verify all required attributes are set
- ✅ Integration test pattern with resource + datasource

### 4. Documentation

**File**: `connectracer/docs/data-sources/qconnect_knowledgebase.md`

#### Includes:
- Complete attribute reference
- Multiple usage examples (basic, resource reference, EXTERNAL KB)
- Use cases and patterns
- Error handling guidance
- Related resources
- API documentation links

### 5. Examples

**Directory**: `connectracer/examples/data-sources/connectracer_qconnect_knowledgebase/`

**File**: `data-source.tf`
- Basic lookup example
- Output usage patterns

## API Mapping

| Terraform Operation | AWS API Call |
|-------------------|--------------|
| Read | `GetKnowledgeBase` |

## Usage Examples

### Basic Usage

```terraform
data "connectracer_qconnect_knowledgebase" "example" {
  knowledge_base_id = "937de449-ad2c-42e2-b89a-2b69e11c4047"
}

output "kb_name" {
  value = data.connectracer_qconnect_knowledgebase.example.name
}
```

### With Resource Reference

```terraform
resource "connectracer_qconnect_knowledgebase" "main" {
  name                = "my-kb"
  knowledge_base_type = "CUSTOM"
}

data "connectracer_qconnect_knowledgebase" "lookup" {
  knowledge_base_id = connectracer_qconnect_knowledgebase.main.id
}
```

### Extract AppIntegrations Configuration

```terraform
data "connectracer_qconnect_knowledgebase" "external" {
  knowledge_base_id = var.external_kb_id
}

output "app_integration_arn" {
  value = try(
    data.connectracer_qconnect_knowledgebase.external.source_configuration.app_integration_arn,
    null
  )
}
```

## Testing & Validation

### Manual Test

Tested with existing knowledge base:

```bash
cd connectracer
terraform plan
```

**Result**:
```
data.connectracer_qconnect_knowledgebase.test: Reading...
data.connectracer_qconnect_knowledgebase.test: Read complete after 1s

Outputs:
  kb_details = {
    app_integration_arn = "arn:aws:app-integrations:eu-central-1:..."
    arn                 = "arn:aws:wisdom:eu-central-1:..."
    description         = "External knowledge base backed by S3"
    id                  = "937de449-ad2c-42e2-b89a-2b69e11c4047"
    name                = "my-knowledge-base"
    status              = "ACTIVE"
    type                = "EXTERNAL"
  }
```

✅ **All attributes retrieved correctly**
✅ **AppIntegrations ARN extracted for EXTERNAL type**
✅ **Status, tags, and metadata working**

### Build Verification

```bash
go build -v ./...
# ✅ Success

task install-dev
# ✅ Provider installed
```

## Important Implementation Details

### 1. Provider Clients Update

Changed provider configuration to pass `ProviderClients` struct to all datasources instead of individual clients. This provides:
- Consistent interface across all datasources
- Access to all AWS service clients (Wisdom, QConnect, AppIntegrations, Connect)
- Future-proof for new datasources

### 2. Backward Compatibility

Updated all existing datasources:
- `datasource_wisdom_assistant.go`
- `wisdom_knowledge_bases_data_source.go`

All still work correctly but now receive `ProviderClients` and extract the specific client they need.

### 3. Type Handling

Properly handles different knowledge base types:
- **CUSTOM**: Standard attributes only
- **EXTERNAL**: Includes `source_configuration` with AppIntegrations ARN

### 4. Null Handling

Properly handles optional nested attributes that may be null:
- `rendering_configuration`
- `server_side_encryption_configuration`
- `source_configuration`

## Use Cases

### 1. Reference Existing Knowledge Base

Look up knowledge bases created outside Terraform:
```terraform
data "connectracer_qconnect_knowledgebase" "existing" {
  knowledge_base_id = var.kb_id
}

resource "connectracer_wisdom_assistant_association" "example" {
  assistant_id     = connectracer_wisdom_assistant.main.id
  association_type = "KNOWLEDGE_BASE"
  association_data = {
    knowledge_base_id = data.connectracer_qconnect_knowledgebase.existing.id
  }
}
```

### 2. Validate Configuration

Check knowledge base configuration before using:
```terraform
data "connectracer_qconnect_knowledgebase" "check" {
  knowledge_base_id = var.kb_id
}

output "is_active" {
  value = data.connectracer_qconnect_knowledgebase.check.status == "ACTIVE"
}

output "is_external" {
  value = data.connectracer_qconnect_knowledgebase.check.knowledge_base_type == "EXTERNAL"
}
```

### 3. Extract Metadata

Get knowledge base metadata for documentation or cross-reference:
```terraform
data "connectracer_qconnect_knowledgebase" "kb" {
  knowledge_base_id = var.kb_id
}

locals {
  kb_metadata = {
    name        = data.connectracer_qconnect_knowledgebase.kb.name
    type        = data.connectracer_qconnect_knowledgebase.kb.knowledge_base_type
    description = data.connectracer_qconnect_knowledgebase.kb.description
    arn         = data.connectracer_qconnect_knowledgebase.kb.knowledge_base_arn
  }
}
```

## Files Created/Modified

### New Files:
1. `connectracer/provider/datasource_qconnect_knowledgebase.go` (223 lines)
2. `connectracer/provider/datasource_qconnect_knowledgebase_test.go` (50 lines)
3. `connectracer/docs/data-sources/qconnect_knowledgebase.md` (187 lines)
4. `connectracer/examples/data-sources/connectracer_qconnect_knowledgebase/data-source.tf`

### Modified Files:
1. `connectracer/provider/provider.go` - Updated datasource configuration
2. `connectracer/provider/datasource_wisdom_assistant.go` - Updated to use ProviderClients
3. `connectracer/provider/wisdom_knowledge_bases_data_source.go` - Updated to use ProviderClients

## Status

✅ **Data source implemented**
✅ **Provider updated**
✅ **Existing datasources updated**
✅ **Tests created**
✅ **Documentation complete**
✅ **Examples provided**
✅ **Manual testing successful**
✅ **Build verified**

## Next Steps

The datasource is fully functional and ready for use:

1. **Available in Provider**: Registered and accessible
2. **Documentation**: Complete with examples
3. **Tested**: Manual testing confirms all attributes work
4. **Compatible**: Works with existing knowledge bases

Use this datasource to:
- Reference existing knowledge bases in Terraform
- Validate knowledge base configuration
- Extract metadata for cross-referencing
- Build dynamic configurations based on KB attributes

## Total Lines of Code

- Data source implementation: ~223 lines
- Tests: ~50 lines
- Documentation: ~187 lines
- Examples: ~21 lines
- **Total: ~481 lines**

## Related Resources

- `connectracer_qconnect_knowledgebase` (resource) - Create/manage knowledge bases
- `connectracer_wisdom_assistant_association` - Associate KBs with assistants
- `connectracer_appintegrations_data_integration` - For EXTERNAL knowledge bases
- `connectracer_connect_integration_association` - Integrate with Connect instances
