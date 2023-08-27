package client

import (
	"io"
)

type SSHClient struct {
	Host       string
	User       string
	PrivateKey []byte
	Port       int
}

func (c SSHClient) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (c SSHClient) Invoke(args []string, input io.Reader) (io.ReadCloser, error) {
	//TODO implement me
	panic("implement me")
}

func (c SSHClient) Disconnect() error {
	//TODO implement me
	panic("implement me")
}
