# Fix Summary: Tags Handling - v0.1.3

## Problem
The Terraform provider was producing "Provider produced inconsistent result after apply" errors when creating resources that automatically add the `AmazonConnectEnabled = "True"` tag.

### Root Cause
1. The `tags` attribute was configured as `Optional: true, Computed: true`
2. User provides tags in configuration: `tags = {Environment = "test"}`
3. Terraform plans these exact tags
4. Provider adds `AmazonConnectEnabled = "True"` during Create/Update
5. State ends up with: `tags = {Environment = "test", AmazonConnectEnabled = "True"}`
6. Terraform detects inconsistency: planned tags ≠ actual tags in state
7. Error: "Provider produced inconsistent result after apply"

## Solution Implemented
Adopted the AWS provider pattern for handling provider-managed tags using separate `tags` and `tags_all` attributes.

### Changes Made

#### 1. Schema Updates (All 4 Resources)
- **`tags` attribute**: Changed from `Optional+Computed` to `Optional` only
  - Contains user-provided tags only
  - No provider modifications
  
- **`tags_all` attribute**: Added as `Computed` only
  - Contains all tags including provider-added `AmazonConnectEnabled`
  - Populated from AWS API responses
  - Used in state for complete tag visibility

#### 2. Resource Model Updates
Added `TagsAll frameworktypes.Map` field to all resource models:
- `WisdomAssistantResourceModel`
- `WisdomAssistantAssociationResourceModel`
- `QConnectKnowledgeBaseResourceModel`
- `AppIntegrationsDataIntegrationResourceModel`

#### 3. Create/Update Logic
```go
// Build complete tag set with required tags
allTags, err := ensureRequiredTags(ctx, plan.Tags)

// Send to AWS API
CreateInput{
    Tags: allTags,
}

// Update state
state.Tags = plan.Tags  // Preserve user input unchanged
state.TagsAll, _ = frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, allTags)
```

#### 4. Read Logic
```go
// Populate tags_all from AWS response
if output.Tags != nil {
    data.TagsAll, diags = frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Tags)
}
// data.Tags remains unchanged from plan/config
```

## Files Modified
1. `connectracer/provider/resource_wisdom_assistant.go`
2. `connectracer/provider/resource_wisdom_assistant_association.go`
3. `connectracer/provider/resource_qconnect_knowledgebase.go`
4. `connectracer/provider/resource_appintegrations_data_integration.go`
5. `connectracer/README.md` - Updated documentation
6. `connectracer/changelog.md` - Added v0.1.3 entry

## Testing Results

### Before Fix
```
Error: Provider produced inconsistent result after apply

When applying changes to connectracer_wisdom_assistant.example, provider
"provider[\"registry.terraform.io/tecracer/connectracer\"]" produced an
unexpected new value: .tags: new element "AmazonConnectEnabled" has appeared.
```

### After Fix
```
Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:

assistant_tags = tomap({
  "Environment" = "test"
  "Version" = "v2"
})
assistant_tags_all = tomap({
  "AmazonConnectEnabled" = "True"
  "Environment" = "test"
  "Version" = "v2"
})
```

✅ **No errors!** Tags work as expected with proper separation of concerns.

## Breaking Change
This is technically a **breaking change** for users who were referencing `.tags` to get all tags:

### Migration Required
**Before (v0.1.2 and earlier):**
```hcl
output "all_tags" {
  value = connectracer_wisdom_assistant.example.tags
}
```

**After (v0.1.3+):**
```hcl
output "user_tags" {
  value = connectracer_wisdom_assistant.example.tags
}

output "all_tags" {
  value = connectracer_wisdom_assistant.example.tags_all
}
```

## Benefits
1. ✅ Fixes Terraform plan/apply validation errors
2. ✅ Follows AWS provider best practices
3. ✅ Clear separation between user tags and provider tags
4. ✅ Aligns with Terraform Plugin Framework semantics
5. ✅ Users can reference complete tag set via `tags_all`
6. ✅ No more "inconsistent result" errors

## Version Info
- **Fixed in:** v0.1.3
- **Date:** 2026-05-07
- **Previous attempt:** v0.1.2 (partial fix, did not fully resolve the issue)
