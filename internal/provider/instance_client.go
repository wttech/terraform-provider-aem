package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/wttech/terraform-provider-aem/internal/utils"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	"time"
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
		out, err := ic.cl.RunShellCommand("curl -s 'https://raw.githubusercontent.com/wttech/aemc/main/pkg/project/common/aemw' -o 'aemw'", ic.dataDir())
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
	if err := ic.saveProfileScript(); err != nil {
		return err
	}
	if err := ic.runScript("create", ic.data.Compose.Create, ic.dataDir()); err != nil {
		return err
	}
	tflog.Info(ic.ctx, "Created AEM instance(s)")
	return nil
}

func (ic *InstanceClient) serviceName() string {
	if ic.data.System.ServiceName.ValueString() != "" {
		return ic.data.System.ServiceName.ValueString()
	}
	return ServiceName
}

func (ic *InstanceClient) saveProfileScript() error {
	envFile := fmt.Sprintf("/etc/profile.d/%s.sh", ic.serviceName())

	var systemEnvMap map[string]string
	ic.data.System.Env.ElementsAs(ic.ctx, &systemEnvMap, true)

	envMap := map[string]string{}
	maps.Copy(envMap, ic.cl.Env)
	maps.Copy(envMap, systemEnvMap)

	ic.cl.Sudo = true
	defer func() { ic.cl.Sudo = false }()

	if err := ic.cl.FileWrite(envFile, utils.EnvToScript(envMap)); err != nil {
		return fmt.Errorf("unable to write AEM environment variables file '%s': %w", envFile, err)
	}
	return nil
}

func (ic *InstanceClient) configureService() error {
	if !ic.data.System.ServiceEnabled.ValueBool() {
		return nil
	}

	user := ic.data.System.User.ValueString()
	if user == "" {
		user = ic.cl.Connection().User()
	}
	vars := map[string]string{
		"DATA_DIR": ic.dataDir(),
		"USER":     user,
	}

	ic.cl.Sudo = true
	defer func() { ic.cl.Sudo = false }()

	serviceTemplated, err := utils.TemplateString(ic.data.System.ServiceConfig.ValueString(), vars)
	if err != nil {
		return fmt.Errorf("unable to template AEM system service definition: %w", err)
	}
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", ic.serviceName())
	if err := ic.cl.FileWrite(serviceFile, serviceTemplated); err != nil {
		return fmt.Errorf("unable to write AEM system service definition '%s': %w", serviceFile, err)
	}

	if err := ic.runServiceAction("enable"); err != nil {
		return err
	}
	return nil
}

func (ic *InstanceClient) runServiceAction(action string) error {
	if !ic.data.System.ServiceEnabled.ValueBool() {
		return nil
	}

	ic.cl.Sudo = true
	defer func() { ic.cl.Sudo = false }()

	outBytes, err := ic.cl.RunShellCommand(fmt.Sprintf("systemctl %s %s.service", action, ic.serviceName()), ".")
	if err != nil {
		return fmt.Errorf("unable to perform AEM system service action '%s': %w", action, err)
	}
	outText := string(outBytes)
	tflog.Info(ic.ctx, outText)
	return nil
}

func (ic *InstanceClient) launch() error {
	tflog.Info(ic.ctx, "Launching AEM instance(s)")
	if err := ic.runServiceAction("start"); err != nil {
		return err
	}
	if err := ic.applyConfig(); err != nil {
		return err
	}
	if err := ic.runScript("configure", ic.data.Compose.Configure, ic.dataDir()); err != nil {
		return err
	}
	tflog.Info(ic.ctx, "Launched AEM instance(s)")
	return nil
}

func (ic *InstanceClient) applyConfig() error {
	tflog.Info(ic.ctx, "Applying AEM instance configuration")
	outBytes, err := ic.cl.RunShellCommand("sh aemw instance launch", ic.dataDir())
	if err != nil {
		return fmt.Errorf("unable to apply AEM instance configuration: %w", err)
	}
	outText := string(outBytes)
	tflog.Info(ic.ctx, outText)
	tflog.Info(ic.ctx, "Applied AEM instance configuration")
	return nil
}

func (ic *InstanceClient) terminate() error {
	tflog.Info(ic.ctx, "Terminating AEM instance(s)")
	if err := ic.runServiceAction("stop"); err != nil {
		return err
	}
	if err := ic.runScript("delete", ic.data.Compose.Delete, ic.dataDir()); err != nil {
		return err
	}
	tflog.Info(ic.ctx, "Terminated AEM instance(s)")
	return nil
}

func (ic *InstanceClient) deleteDataDir() error {
	if err := ic.cl.PathDelete(ic.dataDir()); err != nil {
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
	yamlBytes, err := ic.cl.RunShellCommand("sh aemw instance status --output-format yaml", ic.dataDir())
	if err != nil {
		return status, err
	}
	if err := yaml.Unmarshal(yamlBytes, &status); err != nil {
		return status, fmt.Errorf("unable to parse AEM instance status: %w", err)
	}
	return status, nil
}

func (ic *InstanceClient) bootstrap() error {
	return ic.doActionOnce("bootstrap", ic.cl.WorkDir, func() error {
		return ic.runScript("bootstrap", ic.data.System.Bootstrap, ".")
	})
}

func (ic *InstanceClient) runScript(name string, script InstanceScript, dir string) error {
	scriptCmd := script.Script.ValueString()
	inlineCmds := []string{}
	diags := script.Inline.ElementsAs(ic.ctx, &inlineCmds, true)
	if diags.HasError() {
		return fmt.Errorf("unable to parse script '%s' properly: %s", name, diags)
	}

	if scriptCmd != "" {
		if err := ic.runScriptMultiline(name, scriptCmd, dir); err != nil {
			return err
		}
	}
	if len(inlineCmds) > 0 {
		if err := ic.runScriptInline(name, inlineCmds, dir); err != nil {
			return err
		}
	}

	return nil
}

func (ic *InstanceClient) runScriptInline(name string, inlineCmds []string, dir string) error {
	for i, cmd := range inlineCmds {
		tflog.Info(ic.ctx, fmt.Sprintf("Executing command '%s' of script '%s' (%d/%d)", cmd, name, i+1, len(inlineCmds)))
		textOut, err := ic.cl.RunShellScript(name, cmd, dir)
		if err != nil {
			return fmt.Errorf("unable to execute command '%s' of script '%s' properly: %w", cmd, name, err)
		}
		textStr := string(textOut)
		tflog.Info(ic.ctx, fmt.Sprintf("Executed command '%s' of script '%s' (%d/%d)", cmd, name, i+1, len(inlineCmds)))
		tflog.Info(ic.ctx, textStr)
	}
	return nil
}

func (ic *InstanceClient) runScriptMultiline(name string, scriptCmd string, dir string) error {
	tflog.Info(ic.ctx, fmt.Sprintf("Executing instance script '%s'", name))
	textOut, err := ic.cl.RunShellScript(name, scriptCmd, dir)
	if err != nil {
		return fmt.Errorf("unable to execute script '%s' properly: %w", name, err)
	}
	textStr := string(textOut)
	tflog.Info(ic.ctx, fmt.Sprintf("Executed instance script '%s'", name))
	tflog.Info(ic.ctx, textStr)
	return nil
}

func (ic *InstanceClient) doActionOnce(name string, lockDir string, action func() error) error {
	lock := fmt.Sprintf("%s/provider/%s.lock", lockDir, name)
	exists, err := ic.cl.FileExists(lock)
	if err != nil {
		return fmt.Errorf("cannot read lock file '%s': %w", lock, err)
	}
	if exists {
		tflog.Info(ic.ctx, fmt.Sprintf("Skipping AEM instance action '%s' (lock file already exists '%s')", name, lock))
		return nil
	}
	if err := action(); err != nil {
		return err
	}
	if err := ic.cl.FileWrite(lock, time.Now().String()); err != nil {
		return fmt.Errorf("cannot save lock file '%s': %w", lock, err)
	}
	return nil
}
