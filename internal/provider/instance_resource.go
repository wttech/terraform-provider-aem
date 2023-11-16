package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/wttech/terraform-provider-aem/internal/client"
	"github.com/wttech/terraform-provider-aem/internal/provider/instance"
	"golang.org/x/exp/maps"
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
		Type        types.String `tfsdk:"type"`
		Settings    types.Map    `tfsdk:"settings"`
		Credentials types.Map    `tfsdk:"credentials"`
	} `tfsdk:"client"`
	Files  types.Map `tfsdk:"files"`
	System struct {
		DataDir       types.String   `tfsdk:"data_dir"`
		WorkDir       types.String   `tfsdk:"work_dir"`
		Env           types.Map      `tfsdk:"env"`
		ServiceConfig types.String   `tfsdk:"service_config"`
		User          types.String   `tfsdk:"user"`
		Bootstrap     InstanceScript `tfsdk:"bootstrap"`
	} `tfsdk:"system"`
	Compose struct {
		Download types.Bool     `tfsdk:"download"`
		Version  types.String   `tfsdk:"version"`
		Config   types.String   `tfsdk:"config"`
		Create   InstanceScript `tfsdk:"create"`
		Launch   InstanceScript `tfsdk:"launch"`
		Delete   InstanceScript `tfsdk:"delete"`
	} `tfsdk:"compose"`
	Instances types.List `tfsdk:"instances"`
}

type InstanceScript struct {
	Inline types.List   `tfsdk:"inline"`
	Script types.String `tfsdk:"script"`
}

func instanceScriptSchemaDefault(inline []string, script string) defaults.Object {
	var inlineValue basetypes.ListValue
	if inline == nil {
		inlineValue = types.ListNull(types.StringType)
	} else {
		inlineValues := make([]attr.Value, len(inline))
		for i, v := range inline {
			inlineValues[i] = types.StringValue(v)
		}
		inlineValue = types.ListValueMust(types.StringType, inlineValues)
	}
	return objectdefault.StaticValue(types.ObjectValueMust(
		map[string]attr.Type{
			"inline": types.ListType{ElemType: types.StringType},
			"script": types.StringType,
		},
		map[string]attr.Value{
			"inline": inlineValue,
			"script": types.StringValue(script),
		},
	))
}

type InstanceStatusItemModel struct {
	ID         types.String `tfsdk:"id"`
	URL        types.String `tfsdk:"url"`
	AemVersion types.String `tfsdk:"aem_version"`
	Dir        types.String `tfsdk:"dir"`
	Attributes types.List   `tfsdk:"attributes"`
	RunModes   types.List   `tfsdk:"run_modes"`
}

// fix for https://github.com/hashicorp/terraform-plugin-framework/issues/713
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
					"credentials": schema.MapAttribute{
						MarkdownDescription: "Credentials for the connection type",
						ElementType:         types.StringType,
						Optional:            true,
						Sensitive:           true,
					},
				},
			},
			"system": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"bootstrap": schema.SingleNestedAttribute{
						MarkdownDescription: "Script executed once after connecting to the instance. Typically used for: providing AEM library files (quickstart.jar, license.properties, service packs), mounting data volume, etc. Forces instance recreation if changed.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								Optional:      true,
								ElementType:   types.StringType,
								PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
							},
							"script": schema.StringAttribute{
								Optional:      true,
								PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
					"data_dir": schema.StringAttribute{
						MarkdownDescription: "Remote root path in which AEM Compose files and unpacked instances will be stored",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("/mnt/aemc"),
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"work_dir": schema.StringAttribute{
						MarkdownDescription: "Remote root path in which AEM Compose TF provider temporary files will be stored",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("/tmp/aemc"),
					},
					"service_config": schema.StringAttribute{
						MarkdownDescription: "Contents of the AEM 'systemd' service definition file",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(instance.ServiceConf),
					},
					"user": schema.StringAttribute{
						MarkdownDescription: "System user under which AEM instance will be running. By default, the same as the user used to connect to the machine.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
					"env": schema.MapAttribute{
						MarkdownDescription: "Environment variables for AEM instances",
						ElementType:         types.StringType,
						Computed:            true,
						Optional:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
					},
				},
			},
			"compose": schema.SingleNestedBlock{
				MarkdownDescription: "AEM Compose CLI configuration",
				Attributes: map[string]schema.Attribute{
					"download": schema.BoolAttribute{
						MarkdownDescription: "Toggle automatic AEM Compose CLI wrapper download. If set to false, assume the wrapper is present in the data directory.",
						Required:            true,
					},
					"version": schema.StringAttribute{
						MarkdownDescription: "Version of AEM Compose tool to use on remote AEM machine.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("1.5.8"),
					},
					"config": schema.StringAttribute{
						MarkdownDescription: "Contents o f the AEM Compose YML configuration file.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString(instance.ConfigYML),
					},
					"create": schema.SingleNestedAttribute{
						MarkdownDescription: "Creates the instance or restores from backup. Forces instance recreation if changed.",
						Optional:            true,
						Computed:            true,
						Default:             instanceScriptSchemaDefault(nil, instance.CreateScript),
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								Optional:      true,
								ElementType:   types.StringType,
								PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
							},
							"script": schema.StringAttribute{
								Optional:      true,
								Computed:      true,
								Default:       stringdefault.StaticString(instance.CreateScript),
								PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
					"launch": schema.SingleNestedAttribute{
						MarkdownDescription: "Configures launched instance. Must be idempotent as it is executed always when changed. Typically used for setting up replication agents, installing service packs, etc.",
						Optional:            true,
						Computed:            true,
						Default:             instanceScriptSchemaDefault(nil, instance.LaunchScript),
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"script": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Default:  stringdefault.StaticString(instance.LaunchScript),
							},
						},
					},
					"delete": schema.SingleNestedAttribute{
						MarkdownDescription: "Deletes the instance.",
						Optional:            true,
						Computed:            true,
						Default:             instanceScriptSchemaDefault(nil, instance.DeleteScript),
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"script": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Default:  stringdefault.StaticString(instance.DeleteScript),
							},
						},
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

	if err := ic.copyFiles(); err != nil {
		diags.AddError("Unable to copy AEM instance files", fmt.Sprintf("%s", err))
		return
	}
	if create {
		if err := ic.bootstrap(); err != nil {
			diags.AddError("Unable to bootstrap AEM instance machine", fmt.Sprintf("%s", err))
			return
		}
	}
	if err := ic.prepareWorkDir(); err != nil {
		diags.AddError("Unable to prepare AEM work directory", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.prepareDataDir(); err != nil {
		diags.AddError("Unable to prepare AEM data directory", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.installComposeCLI(); err != nil {
		diags.AddError("Unable to install AEM Compose CLI", fmt.Sprintf("%s", err))
		return
	}
	if err := ic.writeConfigFile(); err != nil {
		diags.AddError("Unable to write AEM configuration file", fmt.Sprintf("%s", err))
		return
	}
	if create {
		if err := ic.create(); err != nil {
			diags.AddError("Unable to create AEM instance", fmt.Sprintf("%s", err))
			return
		}
	}
	if err := ic.launch(); err != nil {
		diags.AddError("Unable to launch AEM instance", fmt.Sprintf("%s", err))
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
		tflog.Info(ctx, "Cannot read AEM instance state as it is not possible to connect	 at the moment. Possible reasons: machine IP change is in progress, machine is not yet created or booting up, etc.")
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

	cl, err := r.clientManager.Make(typeName, r.clientSettings(ctx, model))
	if err != nil {
		return nil, err
	}

	if err := cl.ConnectWithRetry(timeout, func() { tflog.Info(ctx, "Awaiting connection to AEM instance machine") }); err != nil {
		return nil, err
	}

	cl.Env["AEM_CLI_VERSION"] = model.Compose.Version.ValueString()
	cl.Env["AEM_OUTPUT_LOG_MODE"] = "both"
	cl.WorkDir = model.System.WorkDir.ValueString()

	if err := cl.SetupEnv(); err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Connected to AEM instance machine using %s", cl.Connection().Info()))
	return &InstanceClient{cl, ctx, model}, nil
}

func (r *InstanceResource) clientSettings(ctx context.Context, model InstanceResourceModel) map[string]string {
	var settings map[string]string
	model.Client.Settings.ElementsAs(ctx, &settings, true)
	var credentials map[string]string
	model.Client.Credentials.ElementsAs(ctx, &credentials, true)

	combined := map[string]string{}
	maps.Copy(combined, credentials)
	maps.Copy(combined, settings)
	return combined
}
