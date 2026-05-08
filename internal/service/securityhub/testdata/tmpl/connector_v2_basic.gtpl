resource "aws_securityhub_connector_v2" "test" {
{{ template "region" }}
  name = var.rName

  provider_json = jsonencode({
    ProjectKey = "TEST"
  })

{{- template "tags" . }}

  depends_on = [aws_securityhub_aggregator_v2.test]
}

resource "aws_securityhub_aggregator_v2" "test" {
{{- template "region" }}
  region_linking_mode = "SPECIFIED_REGIONS"
  linked_regions      = ["ap-southeast-1"]

  depends_on = [aws_securityhub_account_v2.test]
}

resource "aws_securityhub_account_v2" "test" {
{{- template "region" }}
}