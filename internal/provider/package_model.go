package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wttech/terraform-provider-aem/internal/provider/pkg"
)

func (r *PackageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: pkg.DescriptionMD,
		Attributes: map[string]schema.Attribute{
			"file": schema.StringAttribute{
				MarkdownDescription: "AEM package file path to deploy on the AEM instance.",
				Required:            true,
			},
		},
	}
}

type PackageResourceModel struct {
	File types.String `tfsdk:"file"`
}

/* TODO ...
func (r *PackageResource) newModel() PackageResourceModel {
	model := PackageResourceModel{}
	return model
}
*/
