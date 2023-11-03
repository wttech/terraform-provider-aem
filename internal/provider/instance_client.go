package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

type InstanceClient ClientContext[InstanceResourceModel]

func (ic *InstanceClient) DataDir() string {
	return ic.data.Compose.DataDir.ValueString()
}

func (ic *InstanceClient) Close() error {
	return ic.cl.Disconnect()
}

// TODO chown data dir to ssh user or 'aem' user (create him maybe)
func (ic *InstanceClient) prepareDataDir() error {
	/* TODO to avoid re-uploading library files (probably temporary)
	if _, err := ic.cl.RunShell(fmt.Sprintf("rm -fr %s", ic.DataDir())); err != nil {
		return fmt.Errorf("cannot clean up AEM data directory: %w", err)
	}
	*/
	if _, err := ic.cl.RunShell(fmt.Sprintf("mkdir -p %s", ic.DataDir())); err != nil {
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
		out, err := ic.cl.RunShellWithEnv(fmt.Sprintf("cd %s && curl -s 'https://raw.githubusercontent.com/wttech/aemc/main/pkg/project/common/aemw' -o 'aemw'", ic.DataDir()))
		tflog.Info(ic.ctx, string(out))
		if err != nil {
			return fmt.Errorf("cannot download AEM Compose CLI wrapper: %w", err)
		}
	}
	return nil
}

func (ic *InstanceClient) copyConfigFile() error {
	configFile := ic.data.Compose.ConfigFile.ValueString()
	if err := ic.cl.FileCopy(configFile, fmt.Sprintf("%s/aem/default/etc/aem.yml", ic.DataDir()), true); err != nil {
		return fmt.Errorf("unable to copy AEM configuration file: %w", err)
	}
	return nil
}

func (ic *InstanceClient) copyLibraryDir() error {
	localLibDir := ic.data.Compose.LibDir.ValueString()
	remoteLibDir := fmt.Sprintf("%s/aem/home/lib", ic.DataDir())
	if err := ic.cl.DirCopy(localLibDir, remoteLibDir, false); err != nil {
		return fmt.Errorf("unable to copy AEM library dir: %w", err)
	}
	return nil
}

func (ic *InstanceClient) create() error {
	tflog.Info(ic.ctx, "Creating AEM instance(s)")

	textOut, err := ic.cl.RunShellWithEnv(fmt.Sprintf("cd %s && sh aemw instance create", ic.DataDir()))
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
	textOut, err := ic.cl.RunShellWithEnv(fmt.Sprintf("cd %s && sh aemw instance launch", ic.DataDir()))
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
	textOut, err := ic.cl.RunShellWithEnv(fmt.Sprintf("cd %s && sh aemw instance terminate", ic.DataDir()))
	if err != nil {
		return fmt.Errorf("unable to terminate AEM instance: %w", err)
	}

	textStr := string(textOut) // TODO how about streaming it line by line to tflog ;)
	tflog.Info(ic.ctx, "Terminated AEM instance(s)")
	tflog.Info(ic.ctx, textStr) // TODO consider checking 'changed' flag here if needed

	return nil
}

func (ic *InstanceClient) deleteDataDir() error {
	if _, err := ic.cl.RunShell(fmt.Sprintf("rm -fr %s", ic.DataDir())); err != nil {
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
	yamlBytes, err := ic.cl.RunShellWithEnv(fmt.Sprintf("cd %s && sh aemw instance status --output-format yaml", ic.DataDir()))
	if err != nil {
		return status, err
	}
	if err := yaml.Unmarshal(yamlBytes, &status); err != nil {
		return status, fmt.Errorf("unable to parse AEM instance status: %w", err)
	}
	return status, nil
}
