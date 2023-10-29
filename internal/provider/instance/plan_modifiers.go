package instance

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wttech/terraform-provider-aem/internal/utils"
)

func ConfigFileChecksumPlanModifier() planmodifier.String {
	return &configFileChecksumPlanModifier{}
}

type configFileChecksumPlanModifier struct{}

func (m *configFileChecksumPlanModifier) Description(ctx context.Context) string {
	return "Updates AEM configuration file checksum when contents change."
}

func (m *configFileChecksumPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *configFileChecksumPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var configFile types.String
	diags := req.Plan.GetAttribute(ctx, path.Root("compose").AtName("config_file"), &configFile)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !configFile.IsNull() {
		configFilePath := configFile.ValueString()
		checksum, err := utils.HashFileMD5(configFilePath)
		if err != nil {
			resp.Diagnostics.AddError("Unable to calculate checksum of AEM configuration file", fmt.Sprintf("path '%s', error: %s", configFilePath, err))
			return
		}
		resp.PlanValue = types.StringValue(checksum)
	}
}
