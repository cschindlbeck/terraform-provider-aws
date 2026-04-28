# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_securityhub_account_v2" "test" {}

resource "aws_securityhub_aggregator_v2" "test" {
  region_linking_mode = "SPECIFIED_REGIONS"
  linked_regions      = ["eu-west-1"]

  depends_on = [aws_securityhub_account_v2.test]
}

resource "aws_securityhub_connector_v2" "test" {

  region = var.region

  name = var.rName
  provider_json = jsonencode({
    ProjectKey = "TEST"
  })

  depends_on = [aws_securityhub_aggregator_v2.test]
}

variable "region" {
  description = "Region to deploy resource in"
  type        = string
  nullable    = false
}
