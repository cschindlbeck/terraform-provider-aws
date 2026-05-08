// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package securityhub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	awstypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_securityhub_connector_v2", name="Connector V2")
// @ArnIdentity
// @Tags(identifierAttribute="arn")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/securityhub;securityhub;securityhub.GetConnectorV2Output")
// @Testing(serialize=true)
// @Testing(tagsTest=false)
// @Testing(hasNoPreExistingResource=true)
// @Testing(generator=false)
func newConnectorV2Resource(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &connectorV2Resource{}, nil
}

type connectorV2Resource struct {
	framework.ResourceWithModel[connectorV2ResourceModel]
	framework.WithImportByIdentity
}

func (r *connectorV2Resource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"auth_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connector_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connector_status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrDescription: schema.StringAttribute{
				Optional: true,
			},
			names.AttrKMSKeyARN: schema.StringAttribute{
				CustomType: fwtypes.ARNType,
				Optional:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrName: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_json": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
	}
}

func (r *connectorV2Resource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data connectorV2ResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	var providerCfg awstypes.JiraCloudProviderConfiguration
	if err := json.Unmarshal([]byte(data.ProviderJSON.ValueString()), &providerCfg); err != nil {
		response.Diagnostics.AddError("creating Security Hub V2 Connector", "invalid provider_json: "+err.Error())
		return
	}

	name := fwflex.StringValueFromFramework(ctx, data.Name)
	input := securityhub.CreateConnectorV2Input{
		Name:     data.Name.ValueStringPointer(),
		Provider: &awstypes.ProviderConfigurationMemberJiraCloud{Value: providerCfg},
		Tags:     getTagsIn(ctx),
	}

	if !data.Description.IsNull() {
		input.Description = data.Description.ValueStringPointer()
	}

	if !data.KmsKeyARN.IsNull() {
		input.KmsKeyArn = data.KmsKeyARN.ValueStringPointer()
	}

	output, err := conn.CreateConnectorV2(ctx, &input)

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("creating Security Hub V2 Connector (%s)", name), err.Error())
		return
	}

	// Set values for unknowns.
	response.Diagnostics.Append(fwflex.Flatten(ctx, output, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (r *connectorV2Resource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data connectorV2ResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	// On import, only ARN is set. Extract connector_id from the ARN resource path.
	connectorID := data.ConnectorID.ValueString()
	if connectorID == "" {
		if parts := strings.Split(data.ConnectorARN.ValueString(), "/"); len(parts) > 1 {
			connectorID = parts[len(parts)-1]
		}
	}

	output, err := findConnectorV2ByID(ctx, conn, connectorID)

	if retry.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		response.Diagnostics.AddError("reading Security Hub V2 Connector", err.Error())
		return
	}

	// Set attributes for import.
	response.Diagnostics.Append(fwflex.Flatten(ctx, output, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *connectorV2Resource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var old, new connectorV2ResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &old)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	if !new.Description.Equal(old.Description) {
		input := securityhub.UpdateConnectorV2Input{
			ConnectorId: old.ConnectorID.ValueStringPointer(),
		}

		if !new.Description.IsNull() {
			input.Description = new.Description.ValueStringPointer()
		}

		_, err := conn.UpdateConnectorV2(ctx, &input)

		if err != nil {
			response.Diagnostics.AddError("updating Security Hub V2 Connector", err.Error())
			return
		}
	}

	new.ConnectorStatus = old.ConnectorStatus
	new.AuthURL = old.AuthURL

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *connectorV2Resource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data connectorV2ResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	input := securityhub.DeleteConnectorV2Input{
		ConnectorId: data.ConnectorID.ValueStringPointer(),
	}
	_, err := conn.DeleteConnectorV2(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError("deleting Security Hub V2 Connector", err.Error())
	}
}

func findConnectorV2ByID(ctx context.Context, conn *securityhub.Client, connectorID string) (*securityhub.GetConnectorV2Output, error) {
	input := securityhub.GetConnectorV2Input{
		ConnectorId: &connectorID,
	}
	output, err := conn.GetConnectorV2(ctx, &input)

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

type connectorV2ResourceModel struct {
	framework.WithRegionModel
	AuthURL         types.String `tfsdk:"auth_url"`
	ConnectorARN    types.String `tfsdk:"arn"`
	ConnectorID     types.String `tfsdk:"connector_id"`
	ConnectorStatus types.String `tfsdk:"connector_status"`
	Description     types.String `tfsdk:"description"`
	KmsKeyARN       fwtypes.ARN  `tfsdk:"kms_key_arn"`
	Name            types.String `tfsdk:"name"`
	ProviderJSON    types.String `tfsdk:"provider_json"`
	Tags            tftags.Map   `tfsdk:"tags"`
	TagsAll         tftags.Map   `tfsdk:"tags_all"`
}
