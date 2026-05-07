---
subcategory: "CloudWatch Observability Admin"
layout: "aws"
page_title: "AWS: aws_observabilityadmin_telemetry_rule"
description: |-
  Manages an AWS CloudWatch Observability Admin Telemetry Rule.
---

# Resource: aws_observabilityadmin_telemetry_rule

Manages an AWS CloudWatch Observability Admin Telemetry Rule.

~> **NOTE:** Before using this resource, telemetry evaluation must be enabled for your AWS account. You can use the [`aws_observabilityadmin_telemetry_evaluation`](observabilityadmin_telemetry_evaluation.html) or [`aws_observabilityadmin_telemetry_evaluation_for_organization`](observabilityadmin_telemetry_evaluation_for_organization.html) resource to enable it.

## Example Usage

### Basic Usage

```terraform
resource "aws_observabilityadmin_telemetry_evaluation" "example" {}

resource "aws_observabilityadmin_telemetry_rule" "example" {
  rule_name = "example-telemetry-rule"

  rule {
    telemetry_type = "Logs"

    destination_configuration {
      destination_type = "CloudWatchLogs"
    }
  }

  depends_on = [aws_observabilityadmin_telemetry_evaluation.example]
}
```

### Advanced Configuration with CloudTrail Parameters

```terraform
resource "aws_observabilityadmin_telemetry_evaluation" "example" {}

resource "aws_observabilityadmin_telemetry_rule" "example" {
  rule_name = "advanced-telemetry-rule"

  rule {
    telemetry_type     = "Logs"
    resource_type      = "AWS::CloudTrail::Trail"
    scope              = "Account"
    selection_criteria = "resource.name == 'my-trail'"

    destination_configuration {
      destination_type    = "CloudWatchLogs"
      destination_pattern = "/aws/cloudtrail/logs"
      retention_in_days   = 30

      cloudtrail_parameters {
        advanced_event_selectors {
          name = "Log all data events"

          field_selectors {
            field  = "eventCategory"
            equals = ["Data"]
          }
        }
      }
    }
  }

  tags = {
    Environment = "production"
    Purpose     = "audit-logging"
  }

  depends_on = [aws_observabilityadmin_telemetry_evaluation.example]
}
```

## Argument Reference

This resource supports the following arguments:

* `rule_name` - (Required) Name of the telemetry rule. Must be between 1 and 100 characters and contain only alphanumeric characters, hyphens, underscores, periods, hash symbols, and forward slashes.
* `rule` - (Required) Configuration block for the telemetry rule. See [rule](#rule) below.
* `region` - (Optional) AWS region. If not specified, the provider region is used.
* `tags` - (Optional) Map of tags to assign to the resource. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

### rule

* `telemetry_type` - (Required) Type of telemetry data. Valid values: `Logs`, `Metrics`, `Traces`.
* `resource_type` - (Optional) AWS resource type to apply the rule to.
* `scope` - (Optional) Scope of the telemetry rule.
* `selection_criteria` - (Optional) Selection criteria for the telemetry rule.
* `telemetry_source_types` - (Optional) List of telemetry source types.
* `destination_configuration` - (Optional) Configuration block for the destination. See [destination_configuration](#destination_configuration) below.

### destination_configuration

* `destination_type` - (Optional) Type of destination. Valid values: `CloudWatchLogs`, `S3`, `Firehose`.
* `destination_pattern` - (Optional) Pattern for the destination.
* `retention_in_days` - (Optional) Number of days to retain the data. Must be at least 1.
* `cloudtrail_parameters` - (Optional) Configuration block for CloudTrail parameters. See [cloudtrail_parameters](#cloudtrail_parameters) below.
* `elb_load_balancer_logging_parameters` - (Optional) Configuration block for ELB load balancer logging parameters. See [elb_load_balancer_logging_parameters](#elb_load_balancer_logging_parameters) below.
* `log_delivery_parameters` - (Optional) Configuration block for log delivery parameters. See [log_delivery_parameters](#log_delivery_parameters) below.
* `vpc_flow_log_parameters` - (Optional) Configuration block for VPC flow log parameters. See [vpc_flow_log_parameters](#vpc_flow_log_parameters) below.
* `waf_logging_parameters` - (Optional) Configuration block for WAF logging parameters. See [waf_logging_parameters](#waf_logging_parameters) below.

### cloudtrail_parameters

* `advanced_event_selectors` - (Optional) List of advanced event selectors. See [advanced_event_selectors](#advanced_event_selectors) below.

### advanced_event_selectors

* `name` - (Optional) Name of the advanced event selector.
* `field_selectors` - (Required) List of field selectors. See [field_selectors](#field_selectors) below.

### field_selectors

* `field` - (Required) Field name for the selector.
* `equals` - (Optional) List of values that the field must equal.
* `not_equals` - (Optional) List of values that the field must not equal.
* `starts_with` - (Optional) List of values that the field must start with.
* `not_starts_with` - (Optional) List of values that the field must not start with.
* `ends_with` - (Optional) List of values that the field must end with.
* `not_ends_with` - (Optional) List of values that the field must not end with.

### elb_load_balancer_logging_parameters

* `field_delimiter` - (Optional) Field delimiter for the log format.
* `output_format` - (Optional) Output format for the logs.

### log_delivery_parameters

* `log_types` - (Optional) List of log types to deliver.

### vpc_flow_log_parameters

* `log_format` - (Optional) Log format for VPC flow logs.
* `max_aggregation_interval` - (Optional) Maximum aggregation interval in seconds.
* `traffic_type` - (Optional) Type of traffic to log.

### waf_logging_parameters

* `log_type` - (Optional) Type of WAF logs.
* `logging_filter` - (Optional) Configuration block for logging filter. See [logging_filter](#logging_filter) below.
* `redacted_fields` - (Optional) List of fields to redact from logs. See [redacted_fields](#redacted_fields) below.

### logging_filter

* `default_behavior` - (Required) Default behavior for the filter.
* `filters` - (Optional) List of filters. See [filters](#filters) below.

### filters

* `behavior` - (Required) Behavior for the filter.
* `requirement` - (Required) Requirement for the filter.
* `conditions` - (Required) List of conditions. See [conditions](#conditions) below.

### conditions

* `action_condition` - (Optional) Configuration block for action condition. See [action_condition](#action_condition) below.
* `label_name_condition` - (Optional) Configuration block for label name condition. See [label_name_condition](#label_name_condition) below.

### action_condition

* `action` - (Required) Action for the condition.

### label_name_condition

* `label_name` - (Optional) Label name for the condition.

### redacted_fields

* `method` - (Optional) HTTP method to redact.
* `query_string` - (Optional) Query string to redact.
* `uri_path` - (Optional) URI path to redact.
* `single_header` - (Optional) Configuration block for single header. See [single_header](#single_header) below.

### single_header

* `name` - (Optional) Name of the header to redact.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `id` - Name of the telemetry rule.
* `rule_arn` - ARN of the telemetry rule.
* `tags_all` - Map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

## Timeouts

[Configuration options](https://developer.hashicorp.com/terraform/language/resources/syntax#operation-timeouts):

* `create` - (Default `5m`)
* `update` - (Default `5m`)
* `delete` - (Default `5m`)

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import CloudWatch Observability Admin Telemetry Rules using the `rule_name`. For example:

```terraform
import {
  to = aws_observabilityadmin_telemetry_rule.example
  id = "example-telemetry-rule"
}
```

Using `terraform import`, import CloudWatch Observability Admin Telemetry Rules using the `rule_name`. For example:

```console
% terraform import aws_observabilityadmin_telemetry_rule.example example-telemetry-rule
```
