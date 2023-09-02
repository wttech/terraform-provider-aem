package client

import (
	"fmt"
	"github.com/melbahja/goph"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	typeName   string
	settings   map[string]string
	connection Connection

	Env []string
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

func (c Client) Disconnect() error {
	return c.connection.Disconnect()
}

func (c Client) Connection() Connection {
	return c.connection
}

func (c Client) Run(cmdLine []string) (*goph.Cmd, error) {
	return c.connection.Command(cmdLine)
}

func (c Client) RunShell(cmd string) ([]byte, error) {
	cmdObj, err := c.Command([]string{"sh", "-c", cmd}) // TODO run pass output env vars
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

func (c Client) Command(cmdLine []string) (*goph.Cmd, error) {
	cmd, err := c.connection.Command(cmdLine)
	if err != nil {
		return nil, err
	}
	cmd.Env = c.Env
	return cmd, err
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
		return false, err
	}
	return strings.TrimSpace(string(out)) == "1", nil
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

func (c Client) DirCopy(localPath string, remotePath string, override bool) error {
	if err := c.DirEnsure(remotePath); err != nil {
		return err
	}
	dir, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("cannot list files to copy in directory '%s': %w", localPath, err)
	}
	for _, file := range dir {
		if file.IsDir() {
			if err := c.DirCopy(filepath.Join(localPath, file.Name()), filepath.Join(remotePath, file.Name()), override); err != nil {
				return err
			}
		} else {
			if err := c.FileCopy(filepath.Join(localPath, file.Name()), filepath.Join(remotePath, file.Name()), override); err != nil {
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
