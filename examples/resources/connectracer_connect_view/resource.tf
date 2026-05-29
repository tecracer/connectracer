# ==============================================================================
# Saved (draft) view — useful during development before publishing
# ==============================================================================
resource "connectracer_connect_view" "draft" {
  instance_id = "12345678-1234-1234-1234-123456789012"
  name        = "customer-details-draft"
  description = "Draft view showing customer account details"
  status      = "SAVED"

  template = jsonencode({
    Name = "CustomerDetails"
    Type = "ItemViewer"
    Properties = {
      TitleText = "Customer Account"
      Items = [
        {
          Key   = "$.Contact.Attributes.CustomerName"
          Label = "Name"
        },
        {
          Key   = "$.Contact.Attributes.AccountNumber"
          Label = "Account #"
        }
      ]
    }
  })

  actions = [
    "NEXT",
    "PREVIOUS",
    "END"
  ]

  tags = {
    Environment = "development"
    Team        = "agent-experience"
  }
}

# ==============================================================================
# Published view with an initial immutable version
# A new version is always created automatically on every subsequent update.
# ==============================================================================
resource "connectracer_connect_view" "customer_details" {
  instance_id    = "12345678-1234-1234-1234-123456789012"
  name           = "customer-details"
  description    = "View showing customer account details for the agent workspace"
  status         = "PUBLISHED"
  create_version = true

  # version_description is stamped on the version created by create_version (on
  # first apply) and on every auto-created version after an update.
  version_description = "Initial release"

  template = jsonencode({
    Name = "CustomerDetails"
    Type = "ItemViewer"
    Properties = {
      TitleText = "Customer Account"
      Items = [
        {
          Key   = "$.Contact.Attributes.CustomerName"
          Label = "Name"
        },
        {
          Key   = "$.Contact.Attributes.AccountNumber"
          Label = "Account #"
        },
        {
          Key   = "$.Contact.Attributes.CustomerTier"
          Label = "Tier"
        }
      ]
    }
  })

  actions = [
    "NEXT",
    "PREVIOUS",
    "TRANSFER",
    "END"
  ]

  tags = {
    Environment = "production"
    Team        = "agent-experience"
  }
}

# ==============================================================================
# Case creation flow — multi-step form view
# ==============================================================================
resource "connectracer_connect_view" "case_creation" {
  instance_id         = "12345678-1234-1234-1234-123456789012"
  name                = "case-creation-form"
  description         = "Guided form for agents to open a new support case"
  status              = "PUBLISHED"
  create_version      = true
  version_description = "v1 - initial case form"

  template = jsonencode({
    Name = "CaseCreation"
    Type = "Form"
    Properties = {
      TitleText = "Open New Case"
      Fields = [
        {
          Id       = "subject"
          Label    = "Subject"
          Type     = "Text"
          Required = true
        },
        {
          Id      = "category"
          Label   = "Category"
          Type    = "Dropdown"
          Options = ["Billing", "Technical", "Account", "Other"]
        },
        {
          Id    = "description"
          Label = "Description"
          Type  = "TextArea"
        }
      ]
    }
  })

  actions = [
    "SUBMIT",
    "CANCEL"
  ]

  tags = {
    Environment = "production"
    Team        = "case-management"
  }
}

# ==============================================================================
# Output the qualified IDs (view_id:version) for use in Contact Flows
# ==============================================================================
output "customer_details_view_qualified_id" {
  description = "Qualified ID of the customer details view for use in a Contact Flow"
  value       = connectracer_connect_view.customer_details.qualified_id
}

output "case_creation_view_qualified_id" {
  description = "Qualified ID of the case creation view for use in a Contact Flow"
  value       = connectracer_connect_view.case_creation.qualified_id
}
