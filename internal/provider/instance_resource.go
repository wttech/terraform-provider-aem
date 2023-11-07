package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/wttech/terraform-provider-aem/internal/client"
	"github.com/wttech/terraform-provider-aem/internal/provider/instance"
	"time"
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
	Files  types.Map `tfsdk:"files"`
	System struct {
		DataDir   types.String `tfsdk:"data_dir"`
		Env       types.Map    `tfsdk:"env"`
		Service   types.String `tfsdk:"service"`
		Bootstrap types.String `tfsdk:"bootstrap"`
	} `tfsdk:"system"`
	Compose struct {
		Version types.String `tfsdk:"version"`
		Config  types.String `tfsdk:"config"`
		Init    types.String `tfsdk:"initialize"`
		Launch  types.String `tfsdk:"provision"`
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
			"system": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"bootstrap": schema.StringAttribute{
						MarkdownDescription: "Script executed once after connecting to the instance. Typically used for: providing AEM library files (quickstart.jar, license.properties, service packs), mounting data volume, etc. Forces instance recreation if changed.",
						Optional:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"data_dir": schema.StringAttribute{
						MarkdownDescription: "Remote root path in which AEM Compose files and unpacked instances will be stored",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("/mnt/aemc"),
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"service": schema.StringAttribute{
						MarkdownDescription: "Contents of the 'systemd' service configuration file",
						Optional:            true,
						Default:             stringdefault.StaticString(instance.ServiceTemplate),
					},
					"env": schema.MapAttribute{ // TODO handle it
						MarkdownDescription: "Environment variables for AEM instances",
						ElementType:         types.StringType,
						Computed:            true,
						Optional:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
					},
				},
			},
			"compose": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"version": schema.StringAttribute{
						MarkdownDescription: "Version of AEM Compose tool to use on remote AEM machine",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("1.4.1"),
					},
					"config": schema.StringAttribute{
						MarkdownDescription: "Contents of the AEM Compose YML configuration file",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString(instance.ConfigYML),
					},
					"init": schema.StringAttribute{
						MarkdownDescription: "Script executed once after initializing AEM Compose but before launching the instance. Forces instance recreation if changed. Can be used for restoring instances from backup.",
						Optional:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"launch": schema.StringAttribute{
						MarkdownDescription: "Script executed when the instance is launched. Must be idempotent as it is executed always when changed. Typically used for setting up replication agents, installing service packs, etc.",
						Optional:            true,
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"files": schema.MapAttribute{ // TODO handle it, instead of copying lib dir
				MarkdownDescription: "Files or directories to be copied into the machine",
				ElementType:         types.StringType,
				Computed:            true,
				Optional:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
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
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplaceIf(func(ctx context.Context, request planmodifier.ListRequest, response *listplanmodifier.RequiresReplaceIfFuncResponse) {
						// TODO check if: [1] list is not empty; [2] the same instances are still created; [3] dirs have not changed
						// response.RequiresReplace = true
					}, "If the value of this attribute changes, Terraform will destroy and recreate the resource.", "If the value of this attribute changes, Terraform will destroy and recreate the resource."),
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
	r.createOrUpdate(ctx, &req.Plan, &resp.Diagnostics, &resp.State, true)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.createOrUpdate(ctx, &req.Plan, &resp.Diagnostics, &resp.State, false)
}

func (r *InstanceResource) createOrUpdate(ctx context.Context, plan *tfsdk.Plan, diags *diag.Diagnostics, state *tfsdk.State, create bool) {
	model := r.newModel()

	// Read Terraform plan data into the model
	diags.Append(plan.Get(ctx, &model)...)
	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "Started setting up AEM instance resource")

	ic, err := r.client(ctx, model, time.Minute*5)
	if err != nil {
		diags.AddError("Unable to connect to AEM instance", fmt.Sprintf("%s", err))
		return
	}
	defer func(ic *InstanceClient) {
		err := ic.Close()
		if err != nil {
			diags.AddWarning("Unable to disconnect from AEM instance", fmt.Sprintf("%s", err))
		}
	}(ic)

	if create {
		if err := ic.bootstrap(); err != nil {
			diags.AddError("Unable to bootstrap AEM machine", fmt.Sprintf("%s", err))
			return
		}
	}
	if err := ic.prepareDataDir(); err != nil {
		diags.AddError("Unable to prepare AEM data directory", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.installComposeWrapper(); err != nil {
		diags.AddError("Unable to install AEM Compose CLI", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.copyConfigFile(); err != nil {
		diags.AddError("Unable to copy AEM configuration file", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.copyLibraryDir(); err != nil {
		diags.AddError("Unable to copy AEM library dir", fmt.Sprintf("%s", err))
		return
	}
	if create {
		if err := ic.initialize(); err != nil {
			diags.AddError("Unable to initialize AEM instance", fmt.Sprintf("%s", err))
			return
		}
	}
	if err := ic.create(); err != nil {
		diags.AddError("Unable to create AEM instance", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.launch(); err != nil {
		diags.AddError("Unable to launch AEM instance", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.provision(); err != nil {
		diags.AddError("Unable to provision AEM instance", fmt.Sprintf("%s", err))
		return
	}

	tflog.Info(ctx, "Finished setting up AEM instance resource")

	status, err := ic.ReadStatus()
	if err != nil {
		diags.AddError("Unable to read AEM instance status", fmt.Sprintf("%s", err))
		return
	}

	diags.Append(r.fillModelWithStatus(ctx, &model, status)...)

	// Save data into Terraform state
	diags.Append(state.Set(ctx, &model)...)
}

func (r *InstanceResource) newModel() InstanceResourceModel {
	model := InstanceResourceModel{}
	model.Instances = types.ListValueMust(types.ObjectType{AttrTypes: InstanceStatusItemModel{}.attrTypes()}, []attr.Value{})
	return model
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
	model := r.newModel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ic, err := r.client(ctx, model, time.Second*15)
	if err != nil {
		tflog.Info(ctx, "Cannot read AEM instance state as it is not possible to connect at the moment. Possible reasons: machine IP change is in progress, machine is not yet created or booting up, etc.")
	} else {
		defer func(ic *InstanceClient) {
			err := ic.Close()
			if err != nil {
				resp.Diagnostics.AddWarning("Unable to disconnect from AEM instance", fmt.Sprintf("%s", err))
			}
		}(ic)

		status, err := ic.ReadStatus()
		if err != nil { //
			resp.Diagnostics.AddError("Unable to read AEM instance status", fmt.Sprintf("%s", err))
			return
		}

		resp.Diagnostics.Append(r.fillModelWithStatus(ctx, &model, status)...)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	model := r.newModel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Started deleting AEM instance resource")

	ic, err := r.client(ctx, model, time.Minute*5)
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

	if err := ic.terminate(); err != nil {
		resp.Diagnostics.AddError("Unable to terminate AEM instance", fmt.Sprintf("%s", err))
		return
	}

	if err := ic.deleteDataDir(); err != nil {
		resp.Diagnostics.AddError("Unable to delete AEM data directory", fmt.Sprintf("%s", err))
		return
	}

	tflog.Info(ctx, "Finished deleting AEM instance resource")
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// TODO implement it properly
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *InstanceResource) client(ctx context.Context, model InstanceResourceModel, timeout time.Duration) (*InstanceClient, error) {
	typeName := model.Client.Type.ValueString()
	tflog.Info(ctx, fmt.Sprintf("Connecting to AEM instance machine using %s", typeName))

	var settings map[string]string
	model.Client.Settings.ElementsAs(ctx, &settings, true)

	cl, err := r.clientManager.Make(typeName, settings)
	if err != nil {
		return nil, err
	}

	if err := cl.ConnectWithRetry(timeout, func() { tflog.Info(ctx, "Awaiting connection to AEM instance machine") }); err != nil {
		return nil, err
	}

	cl.Env["AEM_CLI_VERSION"] = model.Compose.Version.ValueString()
	cl.Env["AEM_OUTPUT_LOG_MODE"] = "both"
	cl.EnvDir = "/tmp" // TODO make configurable; or just in user home dir './' ?

	if err := cl.SetupEnv(); err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Connected to AEM instance machine using %s", cl.Connection().Info()))
	return &InstanceClient{cl, ctx, model}, nil
}
