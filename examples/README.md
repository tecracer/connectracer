# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The document generation tool looks for files in the following locations by default. All other *.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or are testable even if some parts are not relevant for the documentation.

* **provider/provider.tf** example file for the provider index page
* **data-sources/`full data source name`/data-source.tf** example file for the named data source page
* **resources/`full resource name`/resource.tf** example file for the named data source page

## End-to-end examples

| Folder | Description |
|--------|-------------|
| [`chat-self-service/`](./chat-self-service/) | Lex chat + orchestration agent + agent assignment + locale build |
| [`resources/connectracer_connect_integration_association/complete_wisdom_integration.tf`](./resources/connectracer_connect_integration_association/complete_wisdom_integration.tf) | Wisdom assistant + knowledge base + Connect integration |

Resource folders may also include extra `*.tf` files (e.g. `complete_chat_self_service.tf`) that are ignored by doc generation but show fuller wiring.
