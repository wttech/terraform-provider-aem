package client

import (
	"context"
	"fmt"
	"github.com/melbahja/goph"
	"github.com/wttech/terraform-provider-aem/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	typeName   string
	settings   map[string]string
	connection Connection

	Env     map[string]string
	WorkDir string
	Sudo    bool
}

func (c Client) TypeName() string {
	return c.typeName
}

func (c Client) Use(callback func(c Client) error) error {
	if err := c.Connect(); err != nil {
		return err
	}
	if err := callback(c); err != nil {
		return err
	}
	if err := c.Disconnect(); err != nil {
		return err
	}
	return nil
}

func (c Client) Connect() error {
	return c.connection.Connect()
}

func (c Client) ConnectWithRetry(timeout time.Duration, callback func()) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	var err error
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cannot connect - awaiting timeout reached '%s': %w", timeout, err)
		default:
			if err = c.Connect(); err == nil {
				return nil
			}
			time.Sleep(3 * time.Second)
			callback()
		}
	}
}

func (c Client) Disconnect() error {
	return c.connection.Disconnect()
}

func (c Client) Connection() Connection {
	return c.connection
}

func (c Client) Command(cmdLine []string) (*goph.Cmd, error) {
	return c.connection.Command(cmdLine)
}

func (c Client) SetupEnv() error {
	if err := c.FileWrite(c.envScriptPath(), c.envScriptString()); err != nil {
		return fmt.Errorf("cannot setup environment script: %w", err)
	}
	return nil
}

func (c Client) envScriptPath() string {
	return fmt.Sprintf("%s/env.sh", c.WorkDir)
}

func (c Client) envScriptString() string {
	return utils.EnvToScript(c.Env)
}

func (c Client) RunShellScript(cmdName string, cmdScript string, dir string) ([]byte, error) {
	remotePath := fmt.Sprintf("%s/%s.sh", c.WorkDir, cmdName)
	if err := c.FileWrite(remotePath, cmdScript); err != nil {
		return nil, fmt.Errorf("cannot write temporary script at remote path '%s': %w", remotePath, err)
	}
	defer func() { _ = c.FileDelete(remotePath) }()
	return c.RunShellCommand(fmt.Sprintf("sh %s", remotePath), dir)
}

func (c Client) RunShellCommand(cmd string, dir string) ([]byte, error) {
	if dir == "" || dir == "." {
		return c.RunShellPurely(fmt.Sprintf("source %s && %s", c.envScriptPath(), cmd))
	}
	return c.RunShellPurely(fmt.Sprintf("source %s && cd %s && %s", c.envScriptPath(), dir, cmd))
}

func (c Client) RunShellPurely(cmd string) ([]byte, error) {
	var cmdLine []string
	if c.Sudo {
		cmdLine = []string{"sudo", "sh", "-c", "\"" + cmd + "\""}
	} else {
		cmdLine = []string{"sh", "-c", "\"" + cmd + "\""}
	}
	cmdObj, err := c.connection.Command(cmdLine)
	if err != nil {
		return nil, fmt.Errorf("cannot create command '%s': %w", cmd, err)
	}
	out, err := cmdObj.CombinedOutput()
	if err != nil {
		if len(out) > 0 { // TODO rethink error handling
			return nil, fmt.Errorf("cannot run command '%s': %w\n\n%s", cmdObj, err, string(out))
		}
		return nil, err
	}
	return out, nil
}

func (c Client) DirEnsure(path string) error {
	_, err := c.RunShellPurely(fmt.Sprintf("mkdir -p %s", path))
	if err != nil {
		return fmt.Errorf("cannot ensure directory '%s': %w", path, err)
	}
	return nil
}

func (c Client) FileExists(path string) (bool, error) {
	out, err := c.RunShellPurely(fmt.Sprintf("test -f %s && echo '0' || echo '1'", path))
	if err != nil {
		return false, fmt.Errorf("cannot check if file exists '%s': %w", path, err)
	}
	return strings.TrimSpace(string(out)) == "0", nil
}

func (c Client) FileMove(oldPath string, newPath string) error {
	if err := c.DirEnsure(filepath.Dir(newPath)); err != nil {
		return err
	}
	if _, err := c.RunShellPurely(fmt.Sprintf("mv %s %s", oldPath, newPath)); err != nil {
		return fmt.Errorf("cannot move file '%s' to '%s': %w", oldPath, newPath, err)
	}
	return nil
}

func (c Client) FileMakeExecutable(path string) error {
	_, err := c.RunShellPurely(fmt.Sprintf("chmod +x %s", path))
	if err != nil {
		return fmt.Errorf("cannot make file executable '%s': %w", path, err)
	}
	return nil
}

func (c Client) DirExists(path string) (bool, error) {
	out, err := c.RunShellPurely(fmt.Sprintf("test -d %s && echo '0' || echo '1'", path))
	if err != nil {
		return false, fmt.Errorf("cannot check if directory exists '%s': %w", path, err)
	}
	return strings.TrimSpace(string(out)) == "0", nil
}

func (c Client) DirCopy(localPath string, remotePath string, override bool) error {
	if err := c.DirEnsure(remotePath); err != nil {
		return err
	}
	dir, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("cannot list files to copy in directory '%s': %w", localPath, err)
	}
	for _, file := range dir {
		localSubPath := filepath.Join(localPath, file.Name())
		remoteSubPath := filepath.Join(remotePath, file.Name())
		if file.IsDir() {
			if err := c.DirCopy(localSubPath, remoteSubPath, override); err != nil {
				return err
			}
		} else {
			if err := c.FileCopy(localSubPath, remoteSubPath, override); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c Client) FileDelete(path string) error {
	if _, err := c.RunShellPurely(fmt.Sprintf("rm -rf %s", path)); err != nil {
		return fmt.Errorf("cannot delete file '%s': %w", path, err)
	}
	return nil
}

func (c Client) FileCopy(localPath string, remotePath string, override bool) error {
	if !override {
		exists, err := c.FileExists(remotePath)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
	}
	if err := c.DirEnsure(filepath.Dir(remotePath)); err != nil {
		return err
	}
	var remoteTmpPath string
	if c.Sudo { // assume that work dir is writable without sudo for uploading time
		remoteTmpPath = fmt.Sprintf("%s/%s.tmp", c.WorkDir, filepath.Base(remotePath))
	} else {
		remoteTmpPath = fmt.Sprintf("%s.tmp", remotePath)
	}
	err := c.FileDelete(remoteTmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = c.FileDelete(remoteTmpPath) }()
	if err := c.connection.CopyFile(localPath, remoteTmpPath); err != nil {
		return err
	}
	if err := c.FileMove(remoteTmpPath, remotePath); err != nil {
		return err
	}
	return nil
}

func (c Client) PathCopy(localPath string, remotePath string, override bool) error {
	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("cannot stat path '%s': %w", localPath, err)
	}
	if stat.IsDir() {
		return c.DirCopy(localPath, remotePath, override)
	}
	return c.FileCopy(localPath, remotePath, override)
}

func (c Client) FileWrite(remotePath string, text string) error {
	file, err := os.CreateTemp(os.TempDir(), "tf-provider-aem-*.tmp")
	path := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(path) }()
	if err != nil {
		return fmt.Errorf("cannot create local writable temporary file to be copied to remote path '%s': %w", remotePath, err)
	}
	if _, err := file.WriteString(text); err != nil {
		return fmt.Errorf("cannot write text to local temporary file to be copied to remote path '%s': %w", remotePath, err)
	}
	if err := c.FileCopy(path, remotePath, true); err != nil {
		return err
	}
	return nil
}
