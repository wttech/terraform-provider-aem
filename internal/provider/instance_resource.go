package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/spf13/cast"
	"github.com/wttech/terraform-provider-aem/internal/client"
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
	plannedModel := r.newModel()

	// Read Terraform planned data into the model
	diags.Append(plan.Get(ctx, &plannedModel)...)
	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "Started setting up AEM instance resource")

	ic, err := r.client(ctx, plannedModel, cast.ToDuration(plannedModel.Client.ActionTimeout.ValueString()))
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
			diags.AddError("Unable to bootstrap AEM instance machine", fmt.Sprintf("%s", err))
			return
		}
	}
	if err := ic.copyFiles(); err != nil {
		diags.AddError("Unable to copy AEM instance files", fmt.Sprintf("%s", err))
		return
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

	diags.Append(r.fillModelWithStatus(ctx, &plannedModel, status)...)

	// Save data into Terraform state
	diags.Append(state.Set(ctx, &plannedModel)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	model := r.newModel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ic, err := r.client(ctx, model, cast.ToDuration(model.Client.StateTimeout.ValueString()))
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

	ic, err := r.client(ctx, model, cast.ToDuration(model.Client.StateTimeout.ValueString()))
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
	resp.Diagnostics.AddError("Import state error", "Not supported for AEM instance resource")
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
