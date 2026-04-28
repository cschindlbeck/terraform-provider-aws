---
subcategory: "Security Hub"
layout: "aws"
page_title: "AWS: aws_securityhub_automation_rule_v2"
description: |-
  Manages a Security Hub V2 automation rule.
---

# Resource: aws_securityhub_automation_rule_v2

Manages a Security Hub V2 Automation Rule, which automatically updates or takes action on findings that match specified criteria.

~> **NOTE:** Automation rules must be created in the aggregation (home) region. A Security Hub V2 Aggregator (`aws_securityhub_aggregator_v2`) must exist before creating automation rules.

## Example Usage

### Basic

```terraform
resource "aws_securityhub_account_v2" "example" {}

resource "aws_securityhub_aggregator_v2" "example" {
  region_linking_mode = "ALL_REGIONS"

  depends_on = [aws_securityhub_account_v2.example]
}

resource "aws_securityhub_automation_rule_v2" "example" {
  rule_name   = "suppress-guardduty-low"
  description = "Suppress low severity GuardDuty findings"
  rule_order  = 100
  rule_status = "ENABLED"

  criteria_json = jsonencode({
    CompositeFilters = [
      {
        StringFilters = [
          {
            FieldName = "metadata.product.name"
            Filter = {
              Comparison = "EQUALS"
              Value      = "GuardDuty"
            }
          }
        ]
      }
    ]
    CompositeOperator = "AND"
  })

  actions_json = jsonencode([
    {
      Type = "FINDING_FIELDS_UPDATE"
      FindingFieldsUpdate = {
        SeverityId = 99
        StatusId   = 3
        Comment    = "Low severity GuardDuty finding suppressed"
      }
    }
  ])

  depends_on = [aws_securityhub_aggregator_v2.example]
}
```

## Argument Reference

This resource supports the following arguments:

* `rule_name` - (Required) The name of the automation rule.
* `description` - (Required) A description of the automation rule.
* `rule_order` - (Required) The priority of the rule. Lower values indicate higher priority.
* `rule_status` - (Optional) The status of the rule. Valid values: `ENABLED`, `DISABLED`. Defaults to `ENABLED`.
* `criteria_json` - (Required) JSON-encoded OCSF finding criteria for the rule. Uses `OcsfFindingFilters` structure with `CompositeFilters` and `CompositeOperator`.
* `actions_json` - (Required) JSON-encoded list of actions (max 1). Each action has a `Type` (`FINDING_FIELDS_UPDATE` or `EXTERNAL_INTEGRATION`) and corresponding configuration.
* `tags` - (Optional) Map of tags to assign to the resource. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `arn` - ARN of the automation rule.
* `tags_all` - Map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import a Security Hub V2 Automation Rule using its ARN. For example:

```terraform
import {
  to = aws_securityhub_automation_rule_v2.example
  id = "arn:aws:securityhub:us-east-1:123456789012:automation-rule/v2/example-id"
}
```

Using `terraform import`, import a Security Hub V2 Automation Rule using its ARN. For example:

```console
% terraform import aws_securityhub_automation_rule_v2.example arn:aws:securityhub:us-east-1:123456789012:automation-rule/v2/example-id
```
