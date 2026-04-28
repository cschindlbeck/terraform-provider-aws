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
			"actions_json": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded list of actions (max 1). Supports FINDING_FIELDS_UPDATE and EXTERNAL_INTEGRATION.",
			},
			"criteria_json": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded OCSF finding criteria for the rule.",
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
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

	var actions []awstypes.AutomationRulesActionV2
	if err := json.Unmarshal([]byte(data.ActionsJSON.ValueString()), &actions); err != nil {
		response.Diagnostics.AddError("invalid actions_json", err.Error())
		return
	}

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

	// Serialize actions back to JSON, stripping null fields.
	if rule.Actions != nil {
		actBytes, err := json.Marshal(rule.Actions)
		if err == nil {
			var raw []map[string]any
			if json.Unmarshal(actBytes, &raw) == nil {
				for i := range raw {
					for k, v := range raw[i] {
						if v == nil {
							delete(raw[i], k)
						}
					}
				}
				if cleaned, err2 := json.Marshal(raw); err2 == nil {
					data.ActionsJSON = types.StringValue(string(cleaned))
				}
			}
		}
	}

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

	if !new.ActionsJSON.Equal(old.ActionsJSON) ||
		!new.CriteriaJSON.Equal(old.CriteriaJSON) ||
		!new.Description.Equal(old.Description) ||
		!new.RuleName.Equal(old.RuleName) ||
		!new.RuleOrder.Equal(old.RuleOrder) ||
		!new.RuleStatus.Equal(old.RuleStatus) {
		var actions []awstypes.AutomationRulesActionV2
		if err := json.Unmarshal([]byte(new.ActionsJSON.ValueString()), &actions); err != nil {
			response.Diagnostics.AddError("invalid actions_json", err.Error())
			return
		}

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
	ARN          types.String  `tfsdk:"arn"`
	ActionsJSON  types.String  `tfsdk:"actions_json"`
	CriteriaJSON types.String  `tfsdk:"criteria_json"`
	Description  types.String  `tfsdk:"description"`
	RuleName     types.String  `tfsdk:"rule_name"`
	RuleOrder    types.Float64 `tfsdk:"rule_order"`
	RuleStatus   types.String  `tfsdk:"rule_status"`
	Tags         tftags.Map    `tfsdk:"tags"`
	TagsAll      tftags.Map    `tfsdk:"tags_all"`
}
