package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/wttech/terraform-provider-aem/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ExampleResource{}
var _ resource.ResourceWithImportState = &ExampleResource{}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

// InstanceResource defines the resource implementation.
type InstanceResource struct {
	client *client.Client
}

// InstanceResourceModel describes the resource data model.
type InstanceResourceModel struct {
	Config struct {
		File    types.String `tfsdk:"file"`
		DataDir types.String `tfsdk:"data_dir"`
		LibDir  types.String `tfsdk:"lib_dir"`
	} `tfsdk:"config"`
	Client struct {
		Type     types.String `tfsdk:"type"`
		Settings types.Map    `tfsdk:"settings"`
	} `tfsdk:"client"`
}

func (r *InstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "AEM Instance resource",
		Blocks: map[string]schema.Block{
			"config": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"file": schema.StringAttribute{
						MarkdownDescription: "Path to the AEM configuration file",
						Computed:            true,
						Default:             stringdefault.StaticString("aem.yml"),
					},
					"data_dir": schema.StringAttribute{
						MarkdownDescription: "Root path which AEM instance(s) will be stored",
						Computed:            true,
						Default:             stringdefault.StaticString("/data/aemc"),
					},
					"lib_dir": schema.StringAttribute{
						MarkdownDescription: "Path to the AEM library directory",
						Optional:            true,
					},
				},
			},
			"client": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "Type of connection to use to connect to the machine on which AEM instance will be running",
						Required:            true,
					},
					"settings": schema.MapAttribute{
						MarkdownDescription: "Settings for the connection type",
						ElementType:         types.StringType,
						Required:            true,
					},
				},
			},
		},
	}
}

func (r *InstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientManager, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ClientManager, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = clientManager
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstanceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "creating AEM instance resource")

	tflog.Trace(ctx, "connecting to AEM instance machine")
	conn, err := r.client.Connect(data.Client.Type.String(), map[string]string{})
	if err != nil {
		resp.Diagnostics.AddError("AEM instance error", fmt.Sprintf("Unable to connect AEM instance machine, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "connected to AEM instance machine")
	defer conn.Disconnect()

	tflog.Trace(ctx, "creating AEM instance resource")

	if err := conn.CopyFile(data.Config.File.String(), fmt.Sprintf("%s/aem/default/etc/aem.yml", data.Config.DataDir.String())); err != nil {
		resp.Diagnostics.AddError("AEM instance error", fmt.Sprintf("Unable to copy AEM configuration file, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "launching AEM instance(s)")
	yml, err := conn.Invoke([]string{"sh", "aemw", "instance", "launch"}, []byte{})
	if err != nil {
		resp.Diagnostics.AddError("AEM instance error", fmt.Sprintf("Unable to launch AEM instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "launched AEM instance(s)")
	tflog.Trace(ctx, string(yml)) // TODO parse output; add it as data to the state; consider checking 'changed' flag from AEMCLI

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	// data.Id = types.StringValue("example-id")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created AEM instance resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstanceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO ... read the resource

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data InstanceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO ... update the resource

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data InstanceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO ... delete the resource
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
