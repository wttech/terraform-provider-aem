package client

import (
	"context"
	"fmt"
	"github.com/melbahja/goph"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	typeName   string
	settings   map[string]string
	connection Connection

	Env    map[string]string
	EnvDir string // TODO this is more like tmp script dir
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

func (c Client) Run(cmdLine []string) (*goph.Cmd, error) {
	return c.connection.Command(cmdLine)
}

func (c Client) SetupEnv() error {
	file, err := os.CreateTemp(os.TempDir(), "tf-provider-aem-env-*.sh")
	path := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(path) }()
	if err != nil {
		return fmt.Errorf("cannot create temporary file for remote shell environment script: %w", err)
	}
	if _, err := file.WriteString(c.envScriptString()); err != nil {
		return fmt.Errorf("cannot write temporary file for remote shell environment script: %w", err)
	}
	if err := c.FileCopy(path, c.envScriptPath(), true); err != nil {
		return err
	}
	return nil
}

func (c Client) envScriptPath() string {
	return fmt.Sprintf("%s/env.sh", c.EnvDir)
}

func (c Client) envScriptString() string {
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	for name, value := range c.Env {
		escapedValue := strings.ReplaceAll(value, "\"", "\\\"")
		escapedValue = strings.ReplaceAll(escapedValue, "$", "\\$")
		sb.WriteString(fmt.Sprintf("export %s=\"%s\"\n", name, escapedValue))
	}
	return sb.String()
}

func (c Client) RunShellWithEnv(cmd string) ([]byte, error) {
	return c.RunShell(fmt.Sprintf("source %s && %s", c.envScriptPath(), cmd))
}

func (c Client) RunShellScriptWithEnv(dir string, cmdScript string) ([]byte, error) {
	file, err := os.CreateTemp(os.TempDir(), "tf-provider-aem-script-*.sh")
	path := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(path) }()
	if err != nil {
		return nil, fmt.Errorf("cannot create temporary file for remote shell script: %w", err)
	}
	if _, err := file.WriteString(cmdScript); err != nil {
		return nil, fmt.Errorf("cannot write temporary file for remote shell script: %w", err)
	}
	remotePath := fmt.Sprintf("%s/%s", c.EnvDir, filepath.Base(file.Name()))
	defer func() { _ = c.FileDelete(remotePath) }()
	if err := c.FileCopy(path, remotePath, true); err != nil {
		return nil, err
	}
	return c.RunShellWithEnv(fmt.Sprintf("cd %s && sh %s", dir, remotePath))
}

func (c Client) RunShell(cmd string) ([]byte, error) {
	cmdObj, err := c.connection.Command([]string{"sh", "-c", "\"" + cmd + "\""})
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
	_, err := c.RunShell(fmt.Sprintf("mkdir -p %s", path))
	if err != nil {
		return fmt.Errorf("cannot ensure directory '%s': %w", path, err)
	}
	return nil
}

func (c Client) FileExists(path string) (bool, error) {
	out, err := c.RunShell(fmt.Sprintf("test -f %s && echo '0' || echo '1'", path))
	if err != nil {
		return false, fmt.Errorf("cannot check if file exists '%s': %w", path, err)
	}
	return strings.TrimSpace(string(out)) == "0", nil
}

func (c Client) FileMove(oldPath string, newPath string) error {
	if err := c.DirEnsure(filepath.Dir(newPath)); err != nil {
		return err
	}
	if _, err := c.RunShell(fmt.Sprintf("mv %s %s", oldPath, newPath)); err != nil {
		return fmt.Errorf("cannot move file '%s' to '%s': %w", oldPath, newPath, err)
	}
	return nil
}

func (c Client) DirExists(path string) (bool, error) {
	out, err := c.RunShell(fmt.Sprintf("test -d %s && echo '0' || echo '1'", path))
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
	if _, err := c.RunShell(fmt.Sprintf("rm -rf %s", path)); err != nil {
		return fmt.Errorf("cannot delete file '%s': %w", path, err)
	}
	return nil
}

// TODO seems that if file exists it is not skipping copying file
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
	remoteTmpPath := fmt.Sprintf("%s.tmp", remotePath)
	defer func() {
		_ = c.FileDelete(remoteTmpPath)
	}()
	if err := c.connection.CopyFile(localPath, remoteTmpPath); err != nil {
		return err
	}
	if err := c.FileMove(remoteTmpPath, remotePath); err != nil {
		return err
	}
	return nil
}
