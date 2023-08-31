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
	"os"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &InstanceResource{}
var _ resource.ResourceWithImportState = &InstanceResource{}

type InstanceCreateContext ClientCreateContext[InstanceResourceModel]

func (ic InstanceCreateContext) DataDir() string {
	return ic.data.Compose.DataDir.ValueString()
}

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

	ic := InstanceCreateContext{cl, ctx, data, req, resp}

	tflog.Info(ctx, "Connected to AEM instance machine")
	defer func(client *client.Client) {
		err := client.Disconnect()
		if err != nil {
			resp.Diagnostics.AddWarning("Unable to disconnect from AEM instance machine", fmt.Sprintf("%s", err))
		}
	}(cl)

	tflog.Info(ctx, "Creating AEM instance resource")

	if !r.prepareDataDir(ic) {
		return
	}
	if !r.installComposeCLI(ic) {
		return
	}
	if !r.copyConfigFile(ic) {
		return
	}
	if !r.copyLibraryDir(ic) {
		return
	}
	if !r.launch(ic) {
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	// data.Id = types.StringValue("example-id")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Info(ctx, "Created AEM instance resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// TODO chown data dir to ssh user or aem user (create him maybe)
func (r *InstanceResource) prepareDataDir(ic InstanceCreateContext) bool {
	if _, err := ic.cl.Run(fmt.Sprintf("rm -fr %s", ic.DataDir())); err != nil {
		ic.resp.Diagnostics.AddError("Cannot clean up AEM data directory", fmt.Sprintf("%s", err))
		return false
	}

	if _, err := ic.cl.Run(fmt.Sprintf("mkdir -p %s", ic.DataDir())); err != nil {
		ic.resp.Diagnostics.AddError("Cannot create AEM data directory", fmt.Sprintf("%s", err))
		return false
	}
	return true
}

// TODO run with context and env vars for setting AEMC version
func (r *InstanceResource) installComposeCLI(ic InstanceCreateContext) bool {
	out, err := ic.cl.Run(fmt.Sprintf("cd %s && curl -s https://raw.githubusercontent.com/wttech/aemc/main/project-init.sh | sh", ic.DataDir()))
	tflog.Info(ic.ctx, string(out))
	if err != nil {
		ic.resp.Diagnostics.AddError("Unable to install AEM Compose CLI", fmt.Sprintf("%s", err))
		return false
	}
	return true
}

func (r *InstanceResource) copyConfigFile(ic InstanceCreateContext) bool {
	configFile := ic.data.Compose.ConfigFile.ValueString()
	if err := ic.cl.FileCopy(configFile, fmt.Sprintf("%s/aem/default/etc/aem.yml", ic.DataDir()), true); err != nil {
		ic.resp.Diagnostics.AddError("Unable to copy AEM configuration file", fmt.Sprintf("%s", err))
		return false
	}
	return true
}

func (r *InstanceResource) copyLibraryDir(ic InstanceCreateContext) bool {
	libDir := ic.data.Compose.LibDir.ValueString()
	libFiles, err := os.ReadDir(libDir)
	if err != nil {
		ic.resp.Diagnostics.AddError("Unable to read files in AEM library directory", fmt.Sprintf("%s", err))
		return false
	}
	for _, libFile := range libFiles {
		if err := ic.cl.FileCopy(fmt.Sprintf("%s/%s", libDir, libFile.Name()), fmt.Sprintf("%s/aem/home/lib/%s", ic.DataDir(), libFile.Name()), false); err != nil {
			ic.resp.Diagnostics.AddError("Unable to copy AEM library file", fmt.Sprintf("file path '%s/%s': %s", libDir, libFile.Name(), err))
			return false
		}
	}
	return true
}

func (r *InstanceResource) launch(ic InstanceCreateContext) bool {
	tflog.Info(ic.ctx, "Launching AEM instance(s)")

	// TODO register systemd service instead and start it
	// TODO set 'ENV TERM=xterm' ; without it AEM is unpacked wrongly; check it in AEMC?
	ymlBytes, err := ic.cl.Run(fmt.Sprintf("cd %s && sh aemw instance launch --output-format yml", ic.DataDir()))
	if err != nil {
		ic.resp.Diagnostics.AddError("Unable to launch AEM instance", fmt.Sprintf("%s", err))
		return false
	}
	yml := string(ymlBytes) // TODO parse it and add to state

	tflog.Info(ic.ctx, "Launched AEM instance(s)")
	tflog.Info(ic.ctx, yml) // TODO parse output; add it as data to the state; consider checking 'changed' flag from AEMCLI
	return true
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstanceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

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
