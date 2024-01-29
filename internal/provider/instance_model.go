package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/wttech/terraform-provider-aem/internal/provider/instance"
)

type InstanceResourceModel struct {
	Client struct {
		Type          types.String `tfsdk:"type"`
		Settings      types.Map    `tfsdk:"settings"`
		Credentials   types.Map    `tfsdk:"credentials"`
		ActionTimeout types.String `tfsdk:"action_timeout"`
		StateTimeout  types.String `tfsdk:"state_timeout"`
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
		Download  types.Bool     `tfsdk:"download"`
		Version   types.String   `tfsdk:"version"`
		Config    types.String   `tfsdk:"config"`
		Create    InstanceScript `tfsdk:"create"`
		Configure InstanceScript `tfsdk:"configure"`
		Delete    InstanceScript `tfsdk:"delete"`
	} `tfsdk:"compose"`
	Instances types.List `tfsdk:"instances"`
}

type InstanceScript struct {
	Inline types.List   `tfsdk:"inline"`
	Script types.String `tfsdk:"script"`
}

// workaround for https://github.com/hashicorp/terraform-plugin-framework/issues/777
func instanceScriptSchemaDefault(inline []string, script *string) defaults.Object {
	return objectdefault.StaticValue(types.ObjectValueMust(
		map[string]attr.Type{
			"inline": types.ListType{ElemType: types.StringType},
			"script": types.StringType,
		},
		map[string]attr.Value{
			"inline": instanceScriptSchemaInlineValue(inline),
			"script": types.StringPointerValue(script),
		},
	))
}

func instanceScriptSchemaInlineValue(inline []string) basetypes.ListValue {
	var result basetypes.ListValue
	if inline == nil {
		result = types.ListNull(types.StringType)
	} else {
		vals := make([]attr.Value, len(inline))
		for i, v := range inline {
			vals[i] = types.StringValue(v)
		}
		result = types.ListValueMust(types.StringType, vals)
	}
	return result
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
		MarkdownDescription: instance.DescriptionMD,
		Blocks: map[string]schema.Block{
			"client": schema.SingleNestedBlock{
				MarkdownDescription: "Connection settings used to access the machine on which the AEM instance will be running.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "Type of connection to use to connect to the machine on which AEM instance will be running.",
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
					"action_timeout": schema.StringAttribute{
						MarkdownDescription: "Used when trying to connect to the AEM instance machine (often right after creating it). Need to be enough long because various types of connections (like AWS SSM or SSH) may need some time to boot up the agent.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("10m"),
					},
					"state_timeout": schema.StringAttribute{
						MarkdownDescription: "Used when reading the AEM instance state when determining the plan.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("30s"),
					},
				},
			},
			"system": schema.SingleNestedBlock{
				MarkdownDescription: "Operating system configuration for the machine on which AEM instance will be running.",
				Attributes: map[string]schema.Attribute{
					"bootstrap": schema.SingleNestedAttribute{
						MarkdownDescription: "Script executed once upon instance connection, often for mounting on VM data volumes from attached disks (e.g., AWS EBS, Azure Disk Storage). This script runs only once, even during instance recreation, as changes are typically persistent and system-wide. If re-execution is needed, it is recommended to set up a new machine.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								MarkdownDescription: "Inline shell commands to be executed",
								ElementType:         types.StringType,
								Optional:            true,
							},
							"script": schema.StringAttribute{
								MarkdownDescription: "Multiline shell script to be executed",
								Optional:            true,
							},
						},
					},
					"data_dir": schema.StringAttribute{
						MarkdownDescription: "Remote root path in which AEM Compose files and unpacked AEM instances will be stored.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("/mnt/aemc"),
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"work_dir": schema.StringAttribute{
						MarkdownDescription: "Remote root path where provider-related files will be stored.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("/tmp/aemc"),
					},
					"service_config": schema.StringAttribute{
						MarkdownDescription: "Contents of the AEM system service definition file (systemd).",
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
						MarkdownDescription: "Environment variables for AEM instances.",
						ElementType:         types.StringType,
						Computed:            true,
						Optional:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
					},
				},
			},
			"compose": schema.SingleNestedBlock{
				MarkdownDescription: "AEM Compose CLI configuration. See [documentation](https://github.com/wttech/aemc#configuration).",
				Attributes: map[string]schema.Attribute{
					"download": schema.BoolAttribute{
						MarkdownDescription: "Toggle automatic AEM Compose CLI wrapper download. If set to false, assume the wrapper is present in the data directory.",
						Computed:            true,
						Optional:            true,
						Default:             booldefault.StaticBool(true),
					},
					"version": schema.StringAttribute{
						MarkdownDescription: "Version of AEM Compose tool to use on remote machine.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString("1.6.12"),
					},
					"config": schema.StringAttribute{
						MarkdownDescription: "Contents of the AEM Compose YML configuration file.",
						Computed:            true,
						Optional:            true,
						Default:             stringdefault.StaticString(instance.ConfigYML),
					},
					"create": schema.SingleNestedAttribute{
						MarkdownDescription: "Script(s) for creating an instance or restoring it from a backup. Typically customized to provide AEM library files (quickstart.jar, license.properties, service packs) from alternative sources (e.g., AWS S3, Azure Blob Storage). Instance recreation is forced if changed.",
						Optional:            true,
						Computed:            true,
						Default:             instanceScriptSchemaDefault(instance.CreateScriptInline, nil),
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								MarkdownDescription: "Inline shell commands to be executed",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Default:             listdefault.StaticValue(instanceScriptSchemaInlineValue(instance.CreateScriptInline)),
								PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
							},
							"script": schema.StringAttribute{
								MarkdownDescription: "Multiline shell script to be executed",
								Optional:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
					"configure": schema.SingleNestedAttribute{
						MarkdownDescription: "Script(s) for configuring a launched instance. Must be idempotent as it is executed always when changed. Typically used for installing AEM service packs, setting up replication agents, etc.",
						Optional:            true,
						Computed:            true,
						Default:             instanceScriptSchemaDefault(instance.LaunchScriptInline, nil),
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								MarkdownDescription: "Inline shell commands to be executed",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Default:             listdefault.StaticValue(instanceScriptSchemaInlineValue(instance.LaunchScriptInline)),
							},
							"script": schema.StringAttribute{
								MarkdownDescription: "Multiline shell script to be executed",
								Optional:            true,
							},
						},
					},
					"delete": schema.SingleNestedAttribute{
						MarkdownDescription: "Script(s) for deleting a stopped instance.",
						Optional:            true,
						Computed:            true,
						Default:             instanceScriptSchemaDefault(instance.DeleteScriptInline, nil),
						Attributes: map[string]schema.Attribute{
							"inline": schema.ListAttribute{
								MarkdownDescription: "Inline shell commands to be executed",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Default:             listdefault.StaticValue(instanceScriptSchemaInlineValue(instance.DeleteScriptInline)),
							},
							"script": schema.StringAttribute{
								MarkdownDescription: "Multiline shell script to be executed",
								Optional:            true,
							},
						},
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"files": schema.MapAttribute{
				MarkdownDescription: "Files or directories to be copied into the machine.",
				ElementType:         types.StringType,
				Computed:            true,
				Optional:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"instances": schema.ListNestedAttribute{
				MarkdownDescription: "Current state of the configured AEM instances.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier of AEM instance defined in the configuration.",
							Computed:            true,
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "The machine-internal HTTP URL address used for communication with the AEM instance.",
							Computed:            true,
						},
						"aem_version": schema.StringAttribute{
							MarkdownDescription: "Version of the AEM instance. Reflects service pack installations.",
							Computed:            true,
						},
						"attributes": schema.ListAttribute{
							MarkdownDescription: "A brief description of the state details for a specific AEM instance. Possible states include 'created', 'uncreated', 'running', 'unreachable', 'up-to-date', and 'out-of-date'.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"run_modes": schema.ListAttribute{
							MarkdownDescription: "A list of run modes for a specific AEM instance.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"dir": schema.StringAttribute{
							MarkdownDescription: "Remote path in which AEM instance is stored.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
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
