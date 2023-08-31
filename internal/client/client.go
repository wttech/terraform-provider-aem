package client

import (
	"fmt"
	"path/filepath"
)

type Client struct {
	typeName   string
	settings   map[string]string
	connection Connection
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

func (c Client) Run(cmd string) ([]byte, error) {
	return c.connection.Run(cmd)
}

func (c Client) RunWithErrOut(cmd string) ([]byte, error) {
	out, err := c.connection.Run(cmd)
	if err != nil {
		if len(out) > 0 { // TODO rethink error handling
			return nil, fmt.Errorf("cannot run command '%s': %w\n\n%s", cmd, err, string(out))
		}
		return nil, err
	}
	return out, nil
}

func (c Client) CopyFile(localPath string, remotePath string) error {
	dir := filepath.Dir(remotePath)
	if _, err := c.connection.Run(fmt.Sprintf("mkdir -p %s", dir)); err != nil {
		return fmt.Errorf("SSH: cannot ensure directory '%s' before copying file: %w", dir, err)
	}
	return c.connection.CopyFile(localPath, remotePath)
}
