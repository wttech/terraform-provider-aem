package client

import (
	"io"
)

type Client struct{}

var ClientDefault = &Client{}

type ClientConnection struct{}

func (c Client) Connect(typeName string, settings map[string]string) (*ClientConnection, error) {
	return &ClientConnection{}, nil
}

func (c *ClientConnection) Disconnect() error {
	return nil
}

func (c *ClientConnection) Invoke(args []string, input []byte) ([]byte, error) {
	return nil, nil
}

func (c *ClientConnection) Call(args []string, input io.ReadCloser) (io.ReadCloser, error) {
	return nil, nil
}

func (c *ClientConnection) CopyFile(localPath string, remotePath string) error {
	return nil
}

func (c *ClientConnection) WriteFile(file io.ReadCloser, remotePath string) error {
	return nil
}
