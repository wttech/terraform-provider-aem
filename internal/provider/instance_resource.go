package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

type InstanceResource struct {
	clientManager *client.ClientManager
}

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
	Instances types.List `tfsdk:"instances"`
}

type InstanceStatusItemModel struct {
	ID         types.String `tfsdk:"id"`
	URL        types.String `tfsdk:"url"`
	AemVersion types.String `tfsdk:"aem_version"`
	Dir        types.String `tfsdk:"dir"`
	Attributes types.List   `tfsdk:"attributes"`
	RunModes   types.List   `tfsdk:"run_modes"`
}

func (o InstanceStatusItemModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":          types.StringType,
		"url":         types.StringType,
		"aem_version": types.StringType,
		"dir":         types.StringType,
		"attributes":  types.ListType{ElemType: types.StringType},
		"run_modes":   types.ListType{ElemType: types.StringType},
	}
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

		Attributes: map[string]schema.Attribute{
			"instances": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"url": schema.StringAttribute{
							Computed: true,
						},
						"aem_version": schema.StringAttribute{
							Computed: true,
						},
						"attributes": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"run_modes": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"dir": schema.StringAttribute{
							Computed: true,
						},
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
	model := r.defaultModel()

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating AEM instance resource")

	ic, err := r.Client(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to AEM instance", fmt.Sprintf("%s", err))
		return
	}
	defer func(ic *InstanceClient) {
		err := ic.Close()
		if err != nil {
			resp.Diagnostics.AddWarning("Unable to disconnect from AEM instance", fmt.Sprintf("%s", err))
		}
	}(ic)

	if err := ic.prepareDataDir(); err != nil {
		resp.Diagnostics.AddError("Unable to prepare AEM data directory", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.installComposeWrapper(); err != nil {
		resp.Diagnostics.AddError("Unable to install AEM Compose CLI", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.copyConfigFile(); err != nil {
		resp.Diagnostics.AddError("Unable to copy AEM configuration file", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.copyLibraryDir(); err != nil {
		resp.Diagnostics.AddError("Unable to copy AEM library dir", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.create(); err != nil {
		resp.Diagnostics.AddError("Unable to create AEM instance", fmt.Sprintf("%s", err))
		return
	}
	/* TODO systemd and stuff for later
	if err := ic.launch(); err != nil {
		resp.Diagnostics.AddError("Unable to launch AEM instance", fmt.Sprintf("%s", err))
		return
	}
	*/

	tflog.Info(ctx, "Created AEM instance resource")

	status, err := ic.ReadStatus()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read AEM instance data", fmt.Sprintf("%s", err))
		return
	}

	resp.Diagnostics.Append(r.fillModelWithStatus(ctx, &model, status)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *InstanceResource) defaultModel() InstanceResourceModel {
	return InstanceResourceModel{}
}

func (r *InstanceResource) fillModelWithStatus(ctx context.Context, model *InstanceResourceModel, status InstanceStatus) diag.Diagnostics {
	var allDiags diag.Diagnostics

	instances := make([]InstanceStatusItemModel, len(status.Data.Instances))
	for i, instance := range status.Data.Instances {
		attributeList, diags := types.ListValueFrom(ctx, types.StringType, instance.Attributes)
		allDiags.Append(diags...)
		runModeList, diags := types.ListValueFrom(ctx, types.StringType, instance.RunModes)
		allDiags.Append(diags...)

		instances[i] = InstanceStatusItemModel{
			ID:         types.StringValue(instance.ID),
			URL:        types.StringValue(instance.URL),
			AemVersion: types.StringValue(instance.AemVersion),
			Dir:        types.StringValue(instance.Dir),
			Attributes: attributeList,
			RunModes:   runModeList,
		}
	}
	instanceList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: InstanceStatusItemModel{}.attrTypes()}, instances)
	allDiags.Append(diags...)
	model.Instances = instanceList

	return allDiags
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	model := r.defaultModel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ic, err := r.Client(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to AEM instance", fmt.Sprintf("%s", err))
		return
	}
	defer func(ic *InstanceClient) {
		err := ic.Close()
		if err != nil {
			resp.Diagnostics.AddWarning("Unable to disconnect from AEM instance", fmt.Sprintf("%s", err))
		}
	}(ic)

	status, err := ic.ReadStatus()
	if err != nil { //
		resp.Diagnostics.AddError("Unable to read AEM instance data", fmt.Sprintf("%s", err))
		return
	}

	resp.Diagnostics.Append(r.fillModelWithStatus(ctx, &model, status)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
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

func (r *InstanceResource) Client(ctx context.Context, data InstanceResourceModel) (*InstanceClient, error) {
	tflog.Info(ctx, "Connecting to AEM instance machine")

	typeName := data.Client.Type.ValueString()
	var settings map[string]string
	data.Client.Settings.ElementsAs(ctx, &settings, true)

	cl, err := r.clientManager.Make(typeName, settings)
	if err != nil {
		return nil, err
	}

	if err := cl.ConnectWithRetry(func() { tflog.Info(ctx, "Awaiting connection to AEM instance machine") }); err != nil {
		return nil, err
	}

	cl.Env["AEM_CLI_VERSION"] = data.Compose.Version.ValueString()
	cl.EnvDir = "/tmp" // TODO make configurable; or just in user home dir './' ?

	if err := cl.SetupEnv(); err != nil {
		return nil, err
	}

	tflog.Info(ctx, "Connected to AEM instance machine")
	return &InstanceClient{cl, ctx, data}, nil
}
