// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package securityhub

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	awstypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_securityhub_automation_rule_v2", name="Automation Rule V2")
// @ArnIdentity
// @Tags(identifierAttribute="arn")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/securityhub;securityhub;securityhub.GetAutomationRuleV2Output")
// @Testing(serialize=true)
// @Testing(tagsTest=false)
// @Testing(hasNoPreExistingResource=true)
// @Testing(generator=false)
func newAutomationRuleV2Resource(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &automationRuleV2Resource{}, nil
}

type automationRuleV2Resource struct {
	framework.ResourceWithModel[automationRuleV2ResourceModel]
	framework.WithImportByIdentity
}

func (r *automationRuleV2Resource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"rule_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the automation rule.",
			},
			names.AttrDescription: schema.StringAttribute{
				Required:    true,
				Description: "A description of the automation rule.",
			},
			"rule_order": schema.Float64Attribute{
				Required:    true,
				Description: "The priority of the rule (lower values = higher priority).",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"rule_status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The status of the rule: ENABLED or DISABLED.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"criteria_json": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded OCSF finding criteria for the rule.",
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
		Blocks: map[string]schema.Block{
			"action": schema.ListNestedBlock{
				Description: "Actions to take when the rule matches. Maximum of 1 action.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						names.AttrType: schema.StringAttribute{
							Required:    true,
							Description: "The action type: FINDING_FIELDS_UPDATE or EXTERNAL_INTEGRATION.",
						},
					},
					Blocks: map[string]schema.Block{
						"finding_fields_update": schema.ListNestedBlock{
							Description: "Settings for updating finding fields.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"comment": schema.StringAttribute{
										Optional:    true,
										Description: "A comment for the finding.",
									},
									"severity_id": schema.Int32Attribute{
										Optional:    true,
										Description: "The severity ID to assign.",
										PlanModifiers: []planmodifier.Int32{
											int32planmodifier.UseStateForUnknown(),
										},
									},
									"status_id": schema.Int32Attribute{
										Optional:    true,
										Description: "The status ID to assign.",
										PlanModifiers: []planmodifier.Int32{
											int32planmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
						"external_integration_configuration": schema.ListNestedBlock{
							Description: "Settings for external integration actions.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"connector_arn": schema.StringAttribute{
										Required:    true,
										Description: "The ARN of the connector.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *automationRuleV2Resource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data automationRuleV2ResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	actions := expandActions(data.Actions)

	var ocsfFilters awstypes.OcsfFindingFilters
	if err := json.Unmarshal([]byte(data.CriteriaJSON.ValueString()), &ocsfFilters); err != nil {
		response.Diagnostics.AddError("invalid criteria_json", err.Error())
		return
	}

	ruleOrder := float32(data.RuleOrder.ValueFloat64())

	input := securityhub.CreateAutomationRuleV2Input{
		RuleName:    data.RuleName.ValueStringPointer(),
		Description: data.Description.ValueStringPointer(),
		RuleOrder:   &ruleOrder,
		Actions:     actions,
		Criteria:    &awstypes.CriteriaMemberOcsfFindingCriteria{Value: ocsfFilters},
		Tags:        getTagsIn(ctx),
	}

	if !data.RuleStatus.IsNull() && !data.RuleStatus.IsUnknown() {
		input.RuleStatus = awstypes.RuleStatusV2(data.RuleStatus.ValueString())
	}

	output, err := conn.CreateAutomationRuleV2(ctx, &input)

	if err != nil {
		response.Diagnostics.AddError("creating Security Hub V2 Automation Rule", err.Error())
		return
	}

	data.ARN = fwflex.StringToFramework(ctx, output.RuleArn)

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (r *automationRuleV2Resource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data automationRuleV2ResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	rule, err := findAutomationRuleV2ByARN(ctx, conn, data.ARN.ValueString())

	if retry.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		response.Diagnostics.AddError("reading Security Hub V2 Automation Rule", err.Error())
		return
	}

	data.ARN = fwflex.StringToFramework(ctx, rule.RuleArn)
	data.RuleName = fwflex.StringToFramework(ctx, rule.RuleName)
	data.Description = fwflex.StringToFramework(ctx, rule.Description)
	if rule.RuleOrder != nil {
		data.RuleOrder = types.Float64Value(float64(*rule.RuleOrder))
	}
	data.RuleStatus = types.StringValue(string(rule.RuleStatus))

	data.Actions = flattenActions(rule.Actions)

	// Serialize criteria back to JSON, extracting inner value from union type.
	if rule.Criteria != nil {
		var critBytes []byte
		switch v := rule.Criteria.(type) {
		case *awstypes.CriteriaMemberOcsfFindingCriteria:
			critBytes, _ = json.Marshal(v.Value)
		default:
			critBytes, _ = json.Marshal(rule.Criteria)
		}
		if critBytes != nil {
			var raw any
			if json.Unmarshal(critBytes, &raw) == nil {
				cleaned := stripV2Nulls(raw)
				if finalBytes, err2 := json.Marshal(cleaned); err2 == nil {
					data.CriteriaJSON = types.StringValue(string(finalBytes))
				}
			}
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *automationRuleV2Resource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var old, new automationRuleV2ResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &old)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	if !new.CriteriaJSON.Equal(old.CriteriaJSON) ||
		!new.Description.Equal(old.Description) ||
		!new.RuleName.Equal(old.RuleName) ||
		!new.RuleOrder.Equal(old.RuleOrder) ||
		!new.RuleStatus.Equal(old.RuleStatus) ||
		len(new.Actions) != len(old.Actions) ||
		actionsChanged(old.Actions, new.Actions) {
		actions := expandActions(new.Actions)

		var ocsfFilters awstypes.OcsfFindingFilters
		if err := json.Unmarshal([]byte(new.CriteriaJSON.ValueString()), &ocsfFilters); err != nil {
			response.Diagnostics.AddError("invalid criteria_json", err.Error())
			return
		}

		ruleOrder := float32(new.RuleOrder.ValueFloat64())

		input := securityhub.UpdateAutomationRuleV2Input{
			Identifier:  old.ARN.ValueStringPointer(),
			RuleName:    new.RuleName.ValueStringPointer(),
			Description: new.Description.ValueStringPointer(),
			RuleOrder:   &ruleOrder,
			Actions:     actions,
			Criteria:    &awstypes.CriteriaMemberOcsfFindingCriteria{Value: ocsfFilters},
		}

		if !new.RuleStatus.IsNull() && !new.RuleStatus.IsUnknown() {
			input.RuleStatus = awstypes.RuleStatusV2(new.RuleStatus.ValueString())
		}

		_, err := conn.UpdateAutomationRuleV2(ctx, &input)

		if err != nil {
			response.Diagnostics.AddError("updating Security Hub V2 Automation Rule", err.Error())
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *automationRuleV2Resource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data automationRuleV2ResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	input := securityhub.DeleteAutomationRuleV2Input{
		Identifier: data.ARN.ValueStringPointer(),
	}
	_, err := conn.DeleteAutomationRuleV2(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError("deleting Security Hub V2 Automation Rule", err.Error())
	}
}

func findAutomationRuleV2ByARN(ctx context.Context, conn *securityhub.Client, arn string) (*securityhub.GetAutomationRuleV2Output, error) {
	input := securityhub.GetAutomationRuleV2Input{
		Identifier: aws.String(arn),
	}
	output, err := conn.GetAutomationRuleV2(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return nil, &retry.NotFoundError{
			LastError: err,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, tfresource.NewEmptyResultError()
	}

	return output, nil
}

// stripV2Nulls recursively removes null values and empty strings from JSON-deserialized data.
func stripV2Nulls(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cleaned := make(map[string]any)
		for k, v2 := range val {
			if v2 == nil {
				continue
			}
			if s, ok := v2.(string); ok && s == "" {
				continue
			}
			result := stripV2Nulls(v2)
			if result != nil {
				cleaned[k] = result
			}
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	case []any:
		var cleaned []any
		for _, item := range val {
			result := stripV2Nulls(item)
			if result != nil {
				cleaned = append(cleaned, result)
			}
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	default:
		return v
	}
}

type automationRuleV2ResourceModel struct {
	framework.WithRegionModel
	ARN          types.String   `tfsdk:"arn"`
	Actions      []actionModel  `tfsdk:"action"`
	CriteriaJSON types.String   `tfsdk:"criteria_json"`
	Description  types.String   `tfsdk:"description"`
	RuleName     types.String   `tfsdk:"rule_name"`
	RuleOrder    types.Float64  `tfsdk:"rule_order"`
	RuleStatus   types.String   `tfsdk:"rule_status"`
	Tags         tftags.Map     `tfsdk:"tags"`
	TagsAll      tftags.Map     `tfsdk:"tags_all"`
}

type actionModel struct {
	Type                             types.String                        `tfsdk:"type"`
	FindingFieldsUpdate              []findingFieldsUpdateModel          `tfsdk:"finding_fields_update"`
	ExternalIntegrationConfiguration []externalIntegrationConfigModel    `tfsdk:"external_integration_configuration"`
}

type findingFieldsUpdateModel struct {
	Comment    types.String `tfsdk:"comment"`
	SeverityId types.Int32  `tfsdk:"severity_id"`
	StatusId   types.Int32  `tfsdk:"status_id"`
}

type externalIntegrationConfigModel struct {
	ConnectorArn types.String `tfsdk:"connector_arn"`
}

func expandActions(actions []actionModel) []awstypes.AutomationRulesActionV2 {
	result := make([]awstypes.AutomationRulesActionV2, len(actions))
	for i, a := range actions {
		result[i].Type = awstypes.AutomationRulesActionTypeV2(a.Type.ValueString())
		if len(a.FindingFieldsUpdate) > 0 {
			ffu := a.FindingFieldsUpdate[0]
			result[i].FindingFieldsUpdate = &awstypes.AutomationRulesFindingFieldsUpdateV2{}
			if !ffu.Comment.IsNull() && !ffu.Comment.IsUnknown() {
				result[i].FindingFieldsUpdate.Comment = ffu.Comment.ValueStringPointer()
			}
			if !ffu.SeverityId.IsNull() && !ffu.SeverityId.IsUnknown() {
				result[i].FindingFieldsUpdate.SeverityId = ffu.SeverityId.ValueInt32Pointer()
			}
			if !ffu.StatusId.IsNull() && !ffu.StatusId.IsUnknown() {
				result[i].FindingFieldsUpdate.StatusId = ffu.StatusId.ValueInt32Pointer()
			}
		}
		if len(a.ExternalIntegrationConfiguration) > 0 {
			eic := a.ExternalIntegrationConfiguration[0]
			result[i].ExternalIntegrationConfiguration = &awstypes.ExternalIntegrationConfiguration{
				ConnectorArn: eic.ConnectorArn.ValueStringPointer(),
			}
		}
	}
	return result
}

func flattenActions(actions []awstypes.AutomationRulesActionV2) []actionModel {
	result := make([]actionModel, len(actions))
	for i, a := range actions {
		result[i].Type = types.StringValue(string(a.Type))
		if a.FindingFieldsUpdate != nil {
			ffu := findingFieldsUpdateModel{}
			if a.FindingFieldsUpdate.Comment != nil {
				ffu.Comment = types.StringValue(*a.FindingFieldsUpdate.Comment)
			} else {
				ffu.Comment = types.StringNull()
			}
			if a.FindingFieldsUpdate.SeverityId != nil {
				ffu.SeverityId = types.Int32Value(*a.FindingFieldsUpdate.SeverityId)
			} else {
				ffu.SeverityId = types.Int32Null()
			}
			if a.FindingFieldsUpdate.StatusId != nil {
				ffu.StatusId = types.Int32Value(*a.FindingFieldsUpdate.StatusId)
			} else {
				ffu.StatusId = types.Int32Null()
			}
			result[i].FindingFieldsUpdate = []findingFieldsUpdateModel{ffu}
		}
		if a.ExternalIntegrationConfiguration != nil {
			eic := externalIntegrationConfigModel{}
			if a.ExternalIntegrationConfiguration.ConnectorArn != nil {
				eic.ConnectorArn = types.StringValue(*a.ExternalIntegrationConfiguration.ConnectorArn)
			}
			result[i].ExternalIntegrationConfiguration = []externalIntegrationConfigModel{eic}
		}
	}
	return result
}

func actionsChanged(old, new []actionModel) bool {
	if len(old) != len(new) {
		return true
	}
	for i := range old {
		if !old[i].Type.Equal(new[i].Type) {
			return true
		}
	}
	return false
}
