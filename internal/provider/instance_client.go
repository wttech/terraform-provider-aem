package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/wttech/terraform-provider-aem/internal/utils"
	"gopkg.in/yaml.v3"
)

const (
	ServiceName = "aem"
)

type InstanceClient ClientContext[InstanceResourceModel]

func (ic *InstanceClient) Close() error {
	return ic.cl.Disconnect()
}

func (ic *InstanceClient) dataDir() string {
	return ic.data.System.DataDir.ValueString()
}

func (ic *InstanceClient) prepareWorkDir() error {
	return ic.cl.DirEnsure(ic.cl.WorkDir)
}

func (ic *InstanceClient) prepareDataDir() error {
	return ic.cl.DirEnsure(ic.dataDir())
}

func (ic *InstanceClient) installComposeCLI() error {
	if !ic.data.Compose.Download.ValueBool() {
		tflog.Info(ic.ctx, "Skipping AEM Compose CLI wrapper download. It is expected to be alternatively installed under the data directory.")
		return nil
	}
	exists, err := ic.cl.FileExists(fmt.Sprintf("%s/aemw", ic.dataDir()))
	if err != nil {
		return fmt.Errorf("cannot check if AEM Compose CLI wrapper is installed: %w", err)
	}
	if !exists {
		tflog.Info(ic.ctx, "Downloading AEM Compose CLI wrapper")
		out, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && curl -s 'https://raw.githubusercontent.com/wttech/aemc/main/pkg/project/common/aemw' -o 'aemw'", ic.dataDir()))
		tflog.Info(ic.ctx, string(out))
		if err != nil {
			return fmt.Errorf("cannot download AEM Compose CLI wrapper: %w", err)
		}
		tflog.Info(ic.ctx, "Downloaded AEM Compose CLI wrapper")
	}
	return nil
}

func (ic *InstanceClient) writeConfigFile() error {
	configYAML := ic.data.Compose.Config.ValueString()
	if err := ic.cl.FileWrite(fmt.Sprintf("%s/aem/default/etc/aem.yml", ic.dataDir()), configYAML); err != nil {
		return fmt.Errorf("unable to copy AEM configuration file: %w", err)
	}
	return nil
}

func (ic *InstanceClient) copyFiles() error {
	var filesMap map[string]string
	ic.data.Files.ElementsAs(ic.ctx, &filesMap, true)
	for localPath, remotePath := range filesMap {
		if err := ic.cl.PathCopy(localPath, remotePath, true); err != nil {
			return fmt.Errorf("unable to copy path '%s' to '%s': %w", localPath, remotePath, err)
		}
	}
	return nil
}

func (ic *InstanceClient) create() error {
	tflog.Info(ic.ctx, "Creating AEM instance(s)")
	if err := ic.configureService(); err != nil {
		return err
	}
	if err := ic.saveEnv(); err != nil {
		return err
	}
	if err := ic.runHook("create", ic.data.Compose.Create.ValueString(), ic.dataDir()); err != nil {
		return err
	}
	tflog.Info(ic.ctx, "Created AEM instance(s)")
	return nil
}

func (ic *InstanceClient) saveEnv() error {
	envFile := fmt.Sprintf("/etc/profile.d/%s.sh", ServiceName)
	var envMap map[string]string
	ic.data.System.Env.ElementsAs(ic.ctx, &envMap, true)

	ic.cl.Sudo = true
	defer func() { ic.cl.Sudo = false }()

	if err := ic.cl.FileWrite(envFile, utils.EnvToScript(envMap)); err != nil {
		return fmt.Errorf("unable to write AEM environment variables file '%s': %w", envFile, err)
	}
	return nil
}

func (ic *InstanceClient) configureService() error {
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", ServiceName)
	serviceTemplated, err := utils.TemplateString(ic.data.System.Service.ValueString(), map[string]string{
		"DATA_DIR":     ic.dataDir(),
		"SERVICE_USER": ic.serviceUser(),
	})
	if err != nil {
		return fmt.Errorf("unable to template AEM system service definition: %w", err)
	}

	ic.cl.Sudo = true
	defer func() { ic.cl.Sudo = false }()

	if err := ic.cl.FileWrite(serviceFile, serviceTemplated); err != nil {
		return fmt.Errorf("unable to write AEM system service definition '%s': %w", serviceFile, err)
	}
	if err := ic.runServiceAction("enable"); err != nil {
		return err
	}
	return nil
}

func (ic *InstanceClient) runServiceAction(action string) error {
	ic.cl.Sudo = true
	defer func() { ic.cl.Sudo = false }()

	outBytes, err := ic.cl.RunShellCommand(fmt.Sprintf("systemctl %s %s.service", action, ServiceName))
	if err != nil {
		return fmt.Errorf("unable to perform AEM system service action '%s': %w", action, err)
	}
	outText := string(outBytes)
	tflog.Info(ic.ctx, outText)
	return nil
}

func (ic *InstanceClient) serviceUser() string {
	user := ic.data.System.User.ValueString()
	if user == "" {
		user = ic.cl.Connection().User()
	}
	return user
}

func (ic *InstanceClient) launch() error {
	tflog.Info(ic.ctx, "Launching AEM instance(s)")
	if err := ic.runServiceAction("start"); err != nil {
		return err
	}
	if err := ic.runHook("launch", ic.data.Compose.Launch.ValueString(), ic.dataDir()); err != nil {
		return err
	}
	tflog.Info(ic.ctx, "Launched AEM instance(s)")
	return nil
}

// TODO consider using "delete --kill"
func (ic *InstanceClient) terminate() error {
	tflog.Info(ic.ctx, "Terminating AEM instance(s)")
	if err := ic.runServiceAction("stop"); err != nil {
		return err
	}
	if err := ic.runHook("delete", ic.data.Compose.Delete.ValueString(), ic.dataDir()); err != nil {
		return err
	}
	tflog.Info(ic.ctx, "Terminated AEM instance(s)")
	return nil
}

func (ic *InstanceClient) deleteDataDir() error {
	if _, err := ic.cl.RunShellPurely(fmt.Sprintf("rm -fr %s", ic.dataDir())); err != nil {
		return fmt.Errorf("cannot delete AEM data directory: %w", err)
	}
	return nil
}

type InstanceStatus struct {
	Data struct {
		Instances []struct {
			ID           string   `yaml:"id"`
			URL          string   `yaml:"url"`
			AemVersion   string   `yaml:"aem_version"`
			Attributes   []string `yaml:"attributes"`
			RunModes     []string `yaml:"run_modes"`
			HealthChecks []string `yaml:"health_checks"`
			Dir          string   `yaml:"dir"`
		} `yaml:"instances"`
	}
}

func (ic *InstanceClient) ReadStatus() (InstanceStatus, error) {
	var status InstanceStatus
	yamlBytes, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && sh aemw instance status --output-format yaml", ic.dataDir()))
	if err != nil {
		return status, err
	}
	if err := yaml.Unmarshal(yamlBytes, &status); err != nil {
		return status, fmt.Errorf("unable to parse AEM instance status: %w", err)
	}
	return status, nil
}

func (ic *InstanceClient) bootstrap() error {
	return ic.runHook("bootstrap", ic.data.System.Bootstrap.ValueString(), ".")
}

func (ic *InstanceClient) runHook(name, cmdScript, dir string) error {
	if cmdScript == "" {
		return nil
	}

	tflog.Info(ic.ctx, fmt.Sprintf("Executing instance hook '%s'", name))

	textOut, err := ic.cl.RunShellScript(name, cmdScript, dir)
	if err != nil {
		return fmt.Errorf("unable to execute hook '%s' properly: %w", name, err)
	}
	textStr := string(textOut) // TODO how about streaming it line by line to tflog ;)

	tflog.Info(ic.ctx, fmt.Sprintf("Executed instance hook '%s'", name))
	tflog.Info(ic.ctx, textStr)

	return nil
}
