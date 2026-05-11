# AWS Provider Pattern Implementation for Tags

## Problem Fixed

Resources that automatically add the `AmazonConnectEnabled = "True"` tag were causing **"Provider produced inconsistent result after apply"** errors because:

1. The `tags` attribute was marked as `Optional: true, Computed: true`
2. User provides specific tags in their Terraform config (e.g., `tags = {Environment="test"}`)
3. Provider adds the required `AmazonConnectEnabled = "True"` tag during Create/Update
4. Terraform compares the plan (which said `tags = {Environment="test"}`) with the state (which has `tags = {Environment="test", AmazonConnectEnabled="True"}`)
5. This mismatch is detected as an inconsistency error

## Solution Implemented

We implemented the **AWS provider pattern** for handling tags, which separates user-provided tags from provider-managed tags:

### Schema Changes

For each resource, the schema now has two tag attributes:

```go
"tags": schema.MapAttribute{
    MarkdownDescription: "User-defined tags to apply to the resource.",
    Optional:            true,  // NOT Computed
    ElementType:         frameworktypes.StringType,
},
"tags_all": schema.MapAttribute{
    MarkdownDescription: "All tags including provider-added tags. The `AmazonConnectEnabled = \"True\"` tag is automatically added.",
    Computed:            true,  // NOT Optional
    ElementType:         frameworktypes.StringType,
},
```

### Model Changes

Added `TagsAll` field to each resource model:

```go
type ResourceModel struct {
    // ... other fields ...
    Tags    frameworktypes.Map `tfsdk:"tags"`
    TagsAll frameworktypes.Map `tfsdk:"tags_all"`
}
```

### Create/Update Logic

1. **Merge tags**: Call `ensureRequiredTags(ctx, data.Tags)` to create `allTags` with user tags + required tag
2. **Send to AWS**: Use `allTags` in API calls
3. **Store in state**:
   - `data.Tags` keeps user-provided tags from plan/config (unchanged)
   - `data.TagsAll` stores all tags (including provider-added ones)

```go
// In Create/Update:
allTags, err := ensureRequiredTags(ctx, data.Tags)
// ... send allTags to AWS API ...

// Note: data.Tags already contains user-provided tags from plan
// Store all tags (including provider-added) in state.TagsAll
tagsAllMap, _ := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, allTags)
data.TagsAll = tagsAllMap
```

### Read Logic

Populate `tags_all` from AWS API response, while preserving `tags` from config:

```go
// In Read:
// Populate tags_all from AWS response
if len(output.Tags) > 0 {
    tagsAllMap, _ := frameworktypes.MapValueFrom(ctx, frameworktypes.StringType, output.Tags)
    data.TagsAll = tagsAllMap
}
// Note: data.Tags is preserved from plan/config, not overwritten from AWS
```

## Modified Resources

The following resource files have been updated with this pattern:

1. `connectracer/provider/resource_wisdom_assistant.go`
2. `connectracer/provider/resource_wisdom_assistant_association.go`
3. `connectracer/provider/resource_qconnect_knowledgebase.go`
4. `connectracer/provider/resource_appintegrations_data_integration.go`

## How This Fixes the Issue

### Before (Broken)
- Plan: `tags = {Environment="test"}` (what user specified)
- State after apply: `tags = {Environment="test", AmazonConnectEnabled="True"}` (what AWS has)
- **Result**: Inconsistency error! Plan != State

### After (Fixed)
- Plan: `tags = {Environment="test"}`, `tags_all = <computed>`
- State after apply: `tags = {Environment="test"}`, `tags_all = {Environment="test", AmazonConnectEnabled="True"}`
- **Result**: Success! Plan matches State because:
  - `tags` in state matches `tags` in plan (both have only user tags)
  - `tags_all` was computed, so Terraform expects it to be populated by the provider

## User Impact

### Existing Users
Users with existing resources will need to run `terraform plan` which will show that `tags_all` is being added. This is a safe migration:

```hcl
# terraform plan output
~ resource "connectracer_wisdom_assistant" "example" {
    tags     = {Environment = "test"}
  + tags_all = {Environment = "test", AmazonConnectEnabled = "True"}
}
```

### New Users
New users can specify tags normally and will see both attributes in state:

```hcl
resource "connectracer_wisdom_assistant" "example" {
  name = "my-assistant"
  type = "AGENT"
  
  tags = {
    Environment = "production"
    Team        = "platform"
  }
}
```

State will show:
- `tags = {Environment = "production", Team = "platform"}`
- `tags_all = {Environment = "production", Team = "platform", AmazonConnectEnabled = "True"}`

## Testing Recommendations

1. **Create a new resource**: Verify that `tags_all` includes both user tags and `AmazonConnectEnabled = "True"`
2. **Update tags**: Add/remove user tags and verify `tags_all` updates correctly
3. **Import existing resource**: Verify that imported resources populate both `tags` and `tags_all` correctly
4. **No drift detection**: Run `terraform plan` multiple times to ensure no persistent drift

## References

This implementation follows the same pattern used by the official AWS Terraform provider for handling default tags at the provider level.
