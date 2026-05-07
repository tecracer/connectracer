# connectracer Provider

The ConnectRacer provider enables Terraform management of AWS Connect and related services including Amazon Q Connect (formerly Wisdom) and AppIntegrations.

## Features

### Automatic Tag Management
All resources in this provider automatically add the **`AmazonConnectEnabled = "True"` tag** required for AWS Connect service-linked role access. This tag is:
- Automatically added during resource creation if not provided
- Preserved during updates
- Stored in Terraform state after creation
- Cannot be accidentally removed
## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24

## Building the Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:
1. Add override in `~/.terraformrc`


```shell
task dev-install
```

### Override example


```json
provider_installation {

  dev_overrides {
      "tecracer/connectracer" = "/Users/johndoe/connectracer"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```
