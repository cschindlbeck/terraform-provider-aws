# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_securityhub_account_v2" "test" {}

resource "aws_securityhub_aggregator_v2" "test" {
  region_linking_mode = "SPECIFIED_REGIONS"
  linked_regions      = ["us-east-1"]

  depends_on = [aws_securityhub_account_v2.test]
}

resource "aws_securityhub_automation_rule_v2" "test" {
  rule_name   = var.rName
  description = "test description"
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

  action {
    type = "FINDING_FIELDS_UPDATE"

    finding_fields_update {
      severity_id = 2
      status_id   = 1
      comment     = "Auto-updated by automation rule"
    }
  }

  region = var.region

  depends_on = [aws_securityhub_aggregator_v2.test]
}

variable "region" {
  description = "Region to deploy resource in"
  type        = string
  nullable    = false
}
