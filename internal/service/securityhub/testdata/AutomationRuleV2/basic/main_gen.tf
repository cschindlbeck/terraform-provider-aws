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

  actions_json = jsonencode([
    {
      Type = "FINDING_FIELDS_UPDATE"
      FindingFieldsUpdate = {
        SeverityId = 2
        StatusId   = 1
        Comment    = "Auto-updated by automation rule"
      }
    }
  ])

  depends_on = [aws_securityhub_aggregator_v2.test]
}
