---
subcategory: "Security Hub"
layout: "aws"
page_title: "AWS: aws_securityhub_connector_v2"
description: |-
  Manages a Security Hub V2 Connector for Jira Cloud integration.
---

# Resource: aws_securityhub_connector_v2

Manages a Security Hub V2 Connector, which integrates with Jira Cloud for automated ticket creation from findings.

~> **NOTE:** Connectors must be created in the aggregation (home) region. A Security Hub V2 Aggregator (`aws_securityhub_aggregator_v2`) must exist before creating connectors.

~> **NOTE:** After creation, the connector will be in `PENDING_AUTHORIZATION` status. Use the `auth_url` output to complete the OAuth authorization flow.

## Example Usage

### Jira Cloud

```terraform
resource "aws_securityhub_account_v2" "example" {}

resource "aws_securityhub_aggregator_v2" "example" {
  region_linking_mode = "ALL_REGIONS"

  depends_on = [aws_securityhub_account_v2.example]
}

resource "aws_securityhub_connector_v2" "example" {
  name = "jira-connector"
  provider_json = jsonencode({
    ProjectKey = "SEC"
  })

  depends_on = [aws_securityhub_aggregator_v2.example]
}

output "auth_url" {
  value = aws_securityhub_connector_v2.example.auth_url
}
```

### With Description and KMS Key

```terraform
resource "aws_securityhub_connector_v2" "example" {
  name        = "jira-connector"
  description = "Jira Cloud integration for security findings"
  kms_key_arn = aws_kms_key.example.arn
  provider_json = jsonencode({
    ProjectKey = "SEC"
  })

  depends_on = [aws_securityhub_aggregator_v2.example]
}
```

## Argument Reference

This resource supports the following arguments:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).
* `name` - (Required, Forces new resource) The name of the connector.
* `description` - (Optional) A description of the connector.
* `provider_json` - (Required, Forces new resource) JSON-encoded Jira Cloud provider configuration. Example: `jsonencode({ ProjectKey = "SEC" })`.
* `kms_key_arn` - (Optional, Forces new resource) ARN of KMS key for connector encryption.
* `tags` - (Optional) Map of tags to assign to the resource. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `arn` - ARN of the connector.
* `auth_url` - OAuth URL for connector authorization. Use this to complete the OAuth flow after creation.
* `connector_id` - ID of the connector.
* `connector_status` - Current status of the connector (`CONNECTED`, `FAILED_TO_CONNECT`, `PENDING_CONFIGURATION`, `PENDING_AUTHORIZATION`).
* `tags_all` - Map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

## Import

In Terraform v1.12.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `identity` attribute. For example:

```terraform
import {
  to = aws_securityhub_connector_v2.example
  identity = {
    arn = "arn:aws:securityhub:us-west-2:123456789012:connector/abc-123"
  }
}

resource "aws_securityhub_connector_v2" "example" {
  ### Configuration omitted for brevity ###
}
```

### Identity Schema

#### Required

- `arn` (String) Amazon Resource Name (ARN) of the Security Hub V2 Connector.

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import Security Hub V2 Connectors using `arn`. For example:

```terraform
import {
  to = aws_securityhub_connector_v2.example
  id = "arn:aws:securityhub:us-west-2:123456789012:connector/abc-123"
}
```

Using `terraform import`, import Security Hub V2 Connectors using `arn`. For example:

```console
% terraform import aws_securityhub_connector_v2.example arn:aws:securityhub:us-west-2:123456789012:connector/abc-123
```

~> **NOTE:** The `provider_json`, `auth_url`, and `connector_status` attributes are not populated on import and must be set in configuration where applicable.
