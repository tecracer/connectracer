# connectracer Provider

The ConnectRacer provider enables Terraform management of AWS Connect and related services including Amazon Q Connect (formerly Wisdom) and AppIntegrations.

## Features

### Automatic Tag Management
All resources in this provider automatically add the **`AmazonConnectEnabled = "True"` tag** required for AWS Connect service-linked role access. This tag is:
- Automatically added during resource creation if not provided
- Preserved during updates
- Available in the `tags_all` computed attribute
- User-provided tags are kept separate in the `tags` attribute

#### Tag Attributes
- **`tags`** (Optional): User-defined tags only. This attribute contains exactly what you configure, without any provider-added tags.
- **`tags_all`** (Computed): All tags including the provider-added `AmazonConnectEnabled = "True"`. Use this in outputs or data sources when you need to see all tags.
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

## Using the provider

```bash
terraform init -upgrade
```

### Override example

File `~/.terraformrc`

```json
provider_installation {
  filesystem_mirror {
    path    = "/Users/joendoe/.terraform.d/plugins"
    include = ["registry.terraform.io/tecracer/connectracer"]
  }
  direct {
    exclude = ["registry.terraform.io/tecracer/connectracer"]
  }
```
