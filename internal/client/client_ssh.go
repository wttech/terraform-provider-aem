package client

import "io"

type SSHClient struct {
	Host       string
	User       string
	PrivateKey []byte
	Port       int
}

func (S SSHClient) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (S SSHClient) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (S SSHClient) Invoke(args []string, input []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (S SSHClient) Call(args []string, input io.ReadCloser) (io.ReadCloser, error) {
	//TODO implement me
	panic("implement me")
}

func (S SSHClient) CopyFile(localPath string, remotePath string) error {
	//TODO implement me
	panic("implement me")
}

func (S SSHClient) WriteFile(file io.ReadCloser, remotePath string) error {
	//TODO implement me
	panic("implement me")
}
