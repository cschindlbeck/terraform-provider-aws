// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package observabilityadmin

import (
	"context"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/observabilityadmin"
	awstypes "github.com/aws/aws-sdk-go-v2/service/observabilityadmin/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_observabilityadmin_telemetry_rule", name="Telemetry Rule")
// @Tags(identifierAttribute="rule_arn")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/observabilityadmin;observabilityadmin;observabilityadmin.GetTelemetryRuleOutput")
// @Testing(preCheck="testAccTelemetryRulePreCheck")
// @Testing(tagsTest=false)
// @Testing(generator=false)
func newTelemetryRuleResource(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &telemetryRuleResource{}

	r.SetDefaultCreateTimeout(5 * time.Minute)
	r.SetDefaultUpdateTimeout(5 * time.Minute)
	r.SetDefaultDeleteTimeout(5 * time.Minute)

	return r, nil
}

type telemetryRuleResource struct {
	framework.ResourceWithModel[telemetryRuleResourceModel]
	framework.WithTimeouts
}

func (r *telemetryRuleResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrID: framework.IDAttribute(),
			"rule_arn":   framework.ARNAttributeComputedOnly(),
			"rule_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
					stringvalidator.RegexMatches(regexache.MustCompile(`^[0-9A-Za-z\-_.#/]+$`), ""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
		Blocks: map[string]schema.Block{
			names.AttrRule: schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[telemetryRuleBlockModel](ctx),
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						names.AttrResourceType: schema.StringAttribute{
							CustomType: fwtypes.StringEnumType[awstypes.ResourceType](),
							Optional:   true,
						},
						"telemetry_type": schema.StringAttribute{
							CustomType: fwtypes.StringEnumType[awstypes.TelemetryType](),
							Required:   true,
						},
					},
				},
			},
			names.AttrTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *telemetryRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data telemetryRuleResourceModel
	smerr.AddEnrich(ctx, &response.Diagnostics, request.Plan.Get(ctx, &data))
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().ObservabilityAdminClient(ctx)

	ruleName := fwflex.StringValueFromFramework(ctx, data.RuleName)
	var input observabilityadmin.CreateTelemetryRuleInput
	smerr.AddEnrich(ctx, &response.Diagnostics, fwflex.Expand(ctx, data, &input))
	if response.Diagnostics.HasError() {
		return
	}

	// Additional fields.
	input.Tags = getTagsIn(ctx)

	output, err := conn.CreateTelemetryRule(ctx, &input)
	if err != nil {
		smerr.AddError(ctx, &response.Diagnostics, err, smerr.ID, ruleName)
		return
	}

	// Set values for unknowns.
	data.RuleARN = fwflex.StringToFramework(ctx, output.RuleArn)
	data.setID()

	smerr.AddEnrich(ctx, &response.Diagnostics, response.State.Set(ctx, data))
}

func (r *telemetryRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data telemetryRuleResourceModel
	smerr.AddEnrich(ctx, &response.Diagnostics, request.State.Get(ctx, &data))
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().ObservabilityAdminClient(ctx)

	ruleName := fwflex.StringValueFromFramework(ctx, data.RuleName)
	output, err := findTelemetryRuleStatus(ctx, conn, &observabilityadmin.GetTelemetryRuleInput{
		RuleIdentifier: aws.String(ruleName),
	})
	if retry.NotFound(err) {
		smerr.AddOne(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		smerr.AddError(ctx, &response.Diagnostics, err, smerr.ID, ruleName)
		return
	}

	// Set the ID and ARN from the output
	data.setID()
	data.RuleARN = fwflex.StringToFramework(ctx, output.RuleArn)

	// Manually set the rule block from the TelemetryRule nested object
	if output.TelemetryRule != nil {
		ruleBlock := telemetryRuleBlockModel{
			ResourceType:  fwtypes.StringEnumValue(output.TelemetryRule.ResourceType),
			TelemetryType: fwtypes.StringEnumValue(output.TelemetryRule.TelemetryType),
		}

		ruleList, diags := fwtypes.NewListNestedObjectValueOfPtr(ctx, &ruleBlock)
		smerr.AddEnrich(ctx, &response.Diagnostics, diags)
		if response.Diagnostics.HasError() {
			return
		}

		data.Rule = ruleList
	}

	smerr.AddEnrich(ctx, &response.Diagnostics, response.State.Set(ctx, &data))
}

func (r *telemetryRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var new, old telemetryRuleResourceModel
	smerr.AddEnrich(ctx, &response.Diagnostics, request.Plan.Get(ctx, &new))
	smerr.AddEnrich(ctx, &response.Diagnostics, request.State.Get(ctx, &old))
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().ObservabilityAdminClient(ctx)

	diff, d := fwflex.Diff(ctx, new, old)
	smerr.AddEnrich(ctx, &response.Diagnostics, d)
	if response.Diagnostics.HasError() {
		return
	}

	if diff.HasChanges() {
		ruleName := fwflex.StringValueFromFramework(ctx, new.RuleName)
		var input observabilityadmin.UpdateTelemetryRuleInput
		smerr.AddEnrich(ctx, &response.Diagnostics, fwflex.Expand(ctx, new, &input))
		if response.Diagnostics.HasError() {
			return
		}

		// Additional fields.
		input.RuleIdentifier = aws.String(ruleName)

		_, err := conn.UpdateTelemetryRule(ctx, &input)
		if err != nil {
			smerr.AddError(ctx, &response.Diagnostics, err, smerr.ID, ruleName)
			return
		}
	}

	smerr.AddEnrich(ctx, &response.Diagnostics, response.State.Set(ctx, &new))
}

func (r *telemetryRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data telemetryRuleResourceModel
	smerr.AddEnrich(ctx, &response.Diagnostics, request.State.Get(ctx, &data))
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().ObservabilityAdminClient(ctx)

	ruleName := fwflex.StringValueFromFramework(ctx, data.RuleName)
	var input observabilityadmin.DeleteTelemetryRuleInput
	input.RuleIdentifier = aws.String(ruleName)

	_, err := conn.DeleteTelemetryRule(ctx, &input)
	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}
	if err != nil {
		smerr.AddError(ctx, &response.Diagnostics, err, smerr.ID, ruleName)
		return
	}
}

func (r *telemetryRuleResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("rule_name"), request, response)
}

func findTelemetryRule(ctx context.Context, conn *observabilityadmin.Client, name string) (*awstypes.TelemetryRule, error) {
	input := observabilityadmin.GetTelemetryRuleInput{
		RuleIdentifier: aws.String(name),
	}

	output, err := findTelemetryRuleStatus(ctx, conn, &input)
	if err != nil {
		return nil, err
	}

	return output.TelemetryRule, nil
}

func findTelemetryRuleStatus(ctx context.Context, conn *observabilityadmin.Client, input *observabilityadmin.GetTelemetryRuleInput) (*observabilityadmin.GetTelemetryRuleOutput, error) {
	output, err := conn.GetTelemetryRule(ctx, input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return nil, &retry.NotFoundError{
			LastError: err,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.TelemetryRule == nil {
		return nil, tfresource.NewEmptyResultError()
	}

	return output, nil
}

type telemetryRuleResourceModel struct {
	framework.WithRegionModel
	ID       types.String                                             `tfsdk:"id"`
	Rule     fwtypes.ListNestedObjectValueOf[telemetryRuleBlockModel] `tfsdk:"rule"`
	RuleARN  types.String                                             `tfsdk:"rule_arn"`
	RuleName types.String                                             `tfsdk:"rule_name"`
	Tags     tftags.Map                                               `tfsdk:"tags"`
	TagsAll  tftags.Map                                               `tfsdk:"tags_all"`
	Timeouts timeouts.Value                                           `tfsdk:"timeouts"`
}

func (m *telemetryRuleResourceModel) setID() {
	m.ID = m.RuleName
}

type telemetryRuleBlockModel struct {
	ResourceType  fwtypes.StringEnum[awstypes.ResourceType]  `tfsdk:"resource_type"`
	TelemetryType fwtypes.StringEnum[awstypes.TelemetryType] `tfsdk:"telemetry_type"`
}

type telemetryDestinationConfigurationModel struct {
	CloudtrailParameters             fwtypes.ListNestedObjectValueOf[cloudtrailParametersModel]             `tfsdk:"cloudtrail_parameters"`
	DestinationPattern               types.String                                                           `tfsdk:"destination_pattern"`
	DestinationType                  fwtypes.StringEnum[awstypes.DestinationType]                           `tfsdk:"destination_type"`
	ELBLoadBalancerLoggingParameters fwtypes.ListNestedObjectValueOf[elbLoadBalancerLoggingParametersModel] `tfsdk:"elb_load_balancer_logging_parameters"`
	LogDeliveryParameters            fwtypes.ListNestedObjectValueOf[logDeliveryParametersModel]            `tfsdk:"log_delivery_parameters"`
	RetentionInDays                  types.Int32                                                            `tfsdk:"retention_in_days"`
	VPCFlowLogParameters             fwtypes.ListNestedObjectValueOf[vpcFlowLogParametersModel]             `tfsdk:"vpc_flow_log_parameters"`
	WAFLoggingParameters             fwtypes.ListNestedObjectValueOf[wafLoggingParametersModel]             `tfsdk:"waf_logging_parameters"`
}

type cloudtrailParametersModel struct {
	AdvancedEventSelectors fwtypes.ListNestedObjectValueOf[advancedEventSelectorModel] `tfsdk:"advanced_event_selectors"`
}

type advancedEventSelectorModel struct {
	FieldSelectors fwtypes.ListNestedObjectValueOf[advancedFieldSelectorModel] `tfsdk:"field_selectors"`
	Name           types.String                                                `tfsdk:"name"`
}

type advancedFieldSelectorModel struct {
	EndsWith      types.List   `tfsdk:"ends_with"`
	Equals        types.List   `tfsdk:"equals"`
	Field         types.String `tfsdk:"field"`
	NotEndsWith   types.List   `tfsdk:"not_ends_with"`
	NotEquals     types.List   `tfsdk:"not_equals"`
	NotStartsWith types.List   `tfsdk:"not_starts_with"`
	StartsWith    types.List   `tfsdk:"starts_with"`
}

type elbLoadBalancerLoggingParametersModel struct {
	FieldDelimiter types.String                              `tfsdk:"field_delimiter"`
	OutputFormat   fwtypes.StringEnum[awstypes.OutputFormat] `tfsdk:"output_format"`
}

type logDeliveryParametersModel struct {
	LogTypes fwtypes.ListOfStringEnum[awstypes.LogType] `tfsdk:"log_types"`
}

type vpcFlowLogParametersModel struct {
	LogFormat              types.String `tfsdk:"log_format"`
	MaxAggregationInterval types.Int32  `tfsdk:"max_aggregation_interval"`
	TrafficType            types.String `tfsdk:"traffic_type"`
}

type wafLoggingParametersModel struct {
	LoggingFilter  fwtypes.ListNestedObjectValueOf[loggingFilterModel] `tfsdk:"logging_filter"`
	LogType        fwtypes.StringEnum[awstypes.WAFLogType]             `tfsdk:"log_type"`
	RedactedFields fwtypes.ListNestedObjectValueOf[fieldToMatchModel]  `tfsdk:"redacted_fields"`
}

type loggingFilterModel struct {
	DefaultBehavior fwtypes.StringEnum[awstypes.FilterBehavior]  `tfsdk:"default_behavior"`
	Filters         fwtypes.ListNestedObjectValueOf[filterModel] `tfsdk:"filters"`
}

type filterModel struct {
	Behavior    fwtypes.StringEnum[awstypes.FilterBehavior]     `tfsdk:"behavior"`
	Conditions  fwtypes.ListNestedObjectValueOf[conditionModel] `tfsdk:"conditions"`
	Requirement fwtypes.StringEnum[awstypes.FilterRequirement]  `tfsdk:"requirement"`
}

type conditionModel struct {
	ActionCondition    fwtypes.ListNestedObjectValueOf[actionConditionModel]    `tfsdk:"action_condition"`
	LabelNameCondition fwtypes.ListNestedObjectValueOf[labelNameConditionModel] `tfsdk:"label_name_condition"`
}

type actionConditionModel struct {
	Action fwtypes.StringEnum[awstypes.Action] `tfsdk:"action"`
}

type labelNameConditionModel struct {
	LabelName types.String `tfsdk:"label_name"`
}

type fieldToMatchModel struct {
	Method       types.String                                       `tfsdk:"method"`
	QueryString  types.String                                       `tfsdk:"query_string"`
	SingleHeader fwtypes.ListNestedObjectValueOf[singleHeaderModel] `tfsdk:"single_header"`
	UriPath      types.String                                       `tfsdk:"uri_path"`
}

type singleHeaderModel struct {
	Name types.String `tfsdk:"name"`
}
