# Fix Summary: Provider Produced Invalid Plan Error (v0.1.2)

## Problem

The provider was throwing the following errors:

### During `terraform plan`:
```
Error: Provider produced invalid plan

Provider "registry.terraform.io/tecracer/connectracer" planned an invalid value for 
connectracer_wisdom_assistant.example.tags: planned value does not match config value.
```

### During `terraform apply`:
```
Error: Provider produced inconsistent result after apply

When applying changes to connectracer_wisdom_assistant.example, provider produced an 
unexpected new value: .tags: new element "AmazonConnectEnabled" has appeared.
```

## Root Cause

The provider was using a **plan modifier** (`ensureConnectEnabledTagModifier` and later `defaultTagsModifier`) to automatically add the `AmazonConnectEnabled = "True"` tag during the planning phase. 

This caused Terraform's validation to fail because:
1. **During planning**: Terraform detected that the planned value (with `AmazonConnectEnabled`) didn't match the configuration value (without it)
2. **During apply**: Terraform detected that the final state contained tags that weren't in the original plan

Terraform's framework validates that:
- The plan must match the configuration for `Optional` attributes
- The apply result must match what was shown in the plan

## Solution

Removed all plan modifiers and implemented a simpler approach:

### What Changed:
1. **Removed** `ensureConnectEnabledTagModifier` and `defaultTagsModifier` from `tags_utils.go`
2. **Removed** all `PlanModifiers` from the tags schema definition in:
   - `resource_wisdom_assistant.go`
   - `resource_wisdom_assistant_association.go`
   - `resource_qconnect_knowledgebase.go`
   - `resource_appintegrations_data_integration.go`

### What Stayed:
- The `ensureRequiredTags()` function still adds `AmazonConnectEnabled = "True"` during Create/Update operations
- Tags are still marked as both `Optional` and `Computed`
- The Read operation still populates tags from AWS into Terraform state

### How It Works Now:
1. **User creates resource** with tags: `{Environment = "production", Team = "support"}`
2. **Provider adds required tag** during Create: `{Environment = "production", Team = "support", AmazonConnectEnabled = "True"}`
3. **Read operation** fetches the resource from AWS and updates state with all tags
4. **State now contains** all three tags, including the auto-added one

### Trade-off:
- The `AmazonConnectEnabled` tag will **not** appear in the plan output
- It **will** appear in the state after apply
- This is acceptable behavior for `Computed` attributes in Terraform

## Files Modified

1. `connectracer/provider/tags_utils.go` - Removed plan modifiers, added explanatory comment
2. `connectracer/provider/resource_wisdom_assistant.go` - Removed PlanModifiers from tags
3. `connectracer/provider/resource_wisdom_assistant_association.go` - Removed PlanModifiers from tags
4. `connectracer/provider/resource_qconnect_knowledgebase.go` - Removed PlanModifiers from tags
5. `connectracer/provider/resource_appintegrations_data_integration.go` - Removed PlanModifiers from tags
6. `connectracer/README.md` - Updated documentation to clarify tag behavior
7. `connectracer/changelog.md` - Documented the fix in v0.1.2

## Testing

The provider now compiles cleanly and `terraform plan` works without errors. The `AmazonConnectEnabled` tag is still automatically added during resource creation, but this happens transparently without causing plan validation errors.

## Version

This fix is released as **v0.1.2** (2026-05-07)
