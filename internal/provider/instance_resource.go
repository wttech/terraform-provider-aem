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
var _ resource.Resource = &InstanceResource{}
var _ resource.ResourceWithImportState = &InstanceResource{}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

// InstanceResource defines the resource implementation.

type InstanceResource struct {
	clientManager *client.ClientManager
}

// InstanceResourceModel describes the resource data model.
type InstanceResourceModel struct {
	Client struct {
		Type     types.String `tfsdk:"type"`
		Settings types.Map    `tfsdk:"settings"`
	} `tfsdk:"client"`
	Compose struct {
		DataDir    types.String `tfsdk:"data_dir"`
		Version    types.String `tfsdk:"version"`
		ConfigFile types.String `tfsdk:"config_file"`
		LibDir     types.String `tfsdk:"lib_dir"`
		InstanceId types.String `tfsdk:"instance_id"`
	} `tfsdk:"compose"`
}

func (r *InstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "AEM Instance resource",
		Blocks: map[string]schema.Block{
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
			"compose": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"data_dir": schema.StringAttribute{
						MarkdownDescription: "Remote path in which AEM Compose data will be stored",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("/mnt/aemc"),
					},
					"version": schema.StringAttribute{
						MarkdownDescription: "Version of AEM Compose tool to use on remote AEM machine",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("1.4.1"),
					},
					"config_file": schema.StringAttribute{
						MarkdownDescription: "Local path to the AEM configuration file",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("aem/default/etc/aem.yml"),
					},
					"lib_dir": schema.StringAttribute{
						MarkdownDescription: "Local path to directory from which AEM library files will be copied to the remote AEM machine",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("aem/home/lib"),
					},
					"instance_id": schema.StringAttribute{
						MarkdownDescription: "ID of the AEM instance to use (one of the instances defined in the configuration file)",
						Optional:            true,
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

	clientManager, ok := req.ProviderData.(*client.ClientManager)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ClientManager, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.clientManager = clientManager
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstanceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating AEM instance resource")

	tflog.Info(ctx, "Connecting to AEM instance machine")

	typeName := data.Client.Type.ValueString()
	var settings map[string]string
	data.Client.Settings.ElementsAs(ctx, &settings, true)

	cl, err := r.clientManager.Make(typeName, settings)
	if err != nil {
		resp.Diagnostics.AddError("Unable to determine AEM instance client", fmt.Sprintf("%s", err))
		return
	}
	if err := cl.Connect(); err != nil {
		resp.Diagnostics.AddError("Unable to connect to AEM instance machine", fmt.Sprintf("%s", err))
		return
	}
	tflog.Info(ctx, "Connected to AEM instance machine")
	defer func(client client.Client) {
		err := client.Disconnect()
		if err != nil {
			resp.Diagnostics.AddWarning("Unable to disconnect from AEM instance machine", fmt.Sprintf("%s", err))
		}
	}(cl)

	tflog.Info(ctx, "Creating AEM instance resource")

	dataDir := data.Compose.DataDir.ValueString()

	if _, err := cl.Run(fmt.Sprintf("rm -fr %s", dataDir)); err != nil {
		resp.Diagnostics.AddError("Cannot clean up AEM data directory", fmt.Sprintf("%s", err))
		return
	}

	if _, err := cl.Run(fmt.Sprintf("mkdir -p %s", dataDir)); err != nil {
		resp.Diagnostics.AddError("Cannot create AEM data directory", fmt.Sprintf("%s", err))
		return
	}

	// TODO chown data dir to ssh user or aem user (create him maybe)
	// TODO run with context and env vars for setting AEMC version

	out, err := cl.Run(fmt.Sprintf("cd %s && curl -s https://raw.githubusercontent.com/wttech/aemc/main/project-init.sh | sh", dataDir))
	tflog.Info(ctx, string(out))
	if err != nil {
		resp.Diagnostics.AddError("Unable to install AEM Compose CLI", fmt.Sprintf("%s", err))
		return
	}

	configFile := data.Compose.ConfigFile.ValueString()
	if err := cl.CopyFile(configFile, fmt.Sprintf("%s/aem/default/etc/aem.yml", dataDir)); err != nil {
		resp.Diagnostics.AddError("Unable to copy AEM configuration file", fmt.Sprintf("%s", err))
		return
	}

	// TODO copy lib dir

	tflog.Info(ctx, "Launching AEM instance(s)")

	// TODO register systemd service instead and start it
	ymlBytes, err := cl.Run(fmt.Sprintf("cd %s && sh aemw instance launch --output-format yml", dataDir))
	if err != nil {
		resp.Diagnostics.AddError("Unable to launch AEM instance", fmt.Sprintf("%s", err))
		return
	}
	yml := string(ymlBytes) // TODO parse it and add to state

	tflog.Info(ctx, "Launched AEM instance(s)")
	tflog.Info(ctx, string(yml)) // TODO parse output; add it as data to the state; consider checking 'changed' flag from AEMCLI

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	// data.Id = types.StringValue("example-id")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Info(ctx, "Created AEM instance resource")

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
