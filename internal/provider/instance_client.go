package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

type InstanceClient ClientContext[InstanceResourceModel]

func (ic *InstanceClient) DataDir() string {
	return ic.data.System.DataDir.ValueString()
}

func (ic *InstanceClient) Close() error {
	return ic.cl.Disconnect()
}

// TODO chown data dir to ssh user or 'aem' user (create him maybe)
func (ic *InstanceClient) prepareDataDir() error {
	if _, err := ic.cl.RunShellPurely(fmt.Sprintf("mkdir -p %s", ic.DataDir())); err != nil {
		return fmt.Errorf("cannot create AEM data directory: %w", err)
	}
	return nil
}

func (ic *InstanceClient) installComposeWrapper() error {
	exists, err := ic.cl.FileExists(fmt.Sprintf("%s/aemw", ic.DataDir()))
	if err != nil {
		return fmt.Errorf("cannot check if AEM Compose CLI wrapper is installed: %w", err)
	}
	if !exists {
		out, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && curl -s 'https://raw.githubusercontent.com/wttech/aemc/main/pkg/project/common/aemw' -o 'aemw'", ic.DataDir()))
		tflog.Info(ic.ctx, string(out))
		if err != nil {
			return fmt.Errorf("cannot download AEM Compose CLI wrapper: %w", err)
		}
	}
	return nil
}

func (ic *InstanceClient) writeConfigFile() error {
	configYAML := ic.data.Compose.Config.ValueString()
	if err := ic.cl.FileWrite(fmt.Sprintf("%s/aem/default/etc/aem.yml", ic.DataDir()), configYAML); err != nil {
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

	textOut, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && sh aemw instance create", ic.DataDir()))
	if err != nil {
		return fmt.Errorf("unable to create AEM instance: %w", err)
	}

	textStr := string(textOut) // TODO how about streaming it line by line to tflog ;)
	tflog.Info(ic.ctx, "Created AEM instance(s)")
	tflog.Info(ic.ctx, textStr) // TODO consider checking 'changed' flag here if needed

	return nil
}

func (ic *InstanceClient) launch() error {
	tflog.Info(ic.ctx, "Launching AEM instance(s)")

	// TODO register systemd service instead and start it
	textOut, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && sh aemw instance launch", ic.DataDir()))
	if err != nil {
		return fmt.Errorf("unable to launch AEM instance: %w", err)
	}

	textStr := string(textOut) // TODO how about streaming it line by line to tflog ;)
	tflog.Info(ic.ctx, "Launched AEM instance(s)")
	tflog.Info(ic.ctx, textStr) // TODO consider checking 'changed' flag here if needed

	return nil
}

// TODO consider using "delete --kill"
func (ic *InstanceClient) terminate() error {
	tflog.Info(ic.ctx, "Terminating AEM instance(s)")

	// TODO use systemd service instead and stop it
	textOut, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && sh aemw instance terminate", ic.DataDir()))
	if err != nil {
		return fmt.Errorf("unable to terminate AEM instance: %w", err)
	}

	textStr := string(textOut) // TODO how about streaming it line by line to tflog ;)
	tflog.Info(ic.ctx, "Terminated AEM instance(s)")
	tflog.Info(ic.ctx, textStr) // TODO consider checking 'changed' flag here if needed

	return nil
}

func (ic *InstanceClient) deleteDataDir() error {
	if _, err := ic.cl.RunShellPurely(fmt.Sprintf("rm -fr %s", ic.DataDir())); err != nil {
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
	yamlBytes, err := ic.cl.RunShellCommand(fmt.Sprintf("cd %s && sh aemw instance status --output-format yaml", ic.DataDir()))
	if err != nil {
		return status, err
	}
	if err := yaml.Unmarshal(yamlBytes, &status); err != nil {
		return status, fmt.Errorf("unable to parse AEM instance status: %w", err)
	}
	return status, nil
}

// TODO when create fails this could be run twice; how about protecting it with lock?
func (ic *InstanceClient) runBootstrapHook() error {
	return ic.runHook("bootstrap", ic.data.System.Bootstrap.ValueString(), ".")
}

// TODO when create fails this could be run twice; how about protecting it with lock?
func (ic *InstanceClient) runInitHook() error {
	return ic.runHook("init", ic.data.Compose.Init.ValueString(), ic.DataDir())
}

func (ic *InstanceClient) runLaunchHook() error {
	return ic.runHook("launch", ic.data.Compose.Launch.ValueString(), ic.DataDir())
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
