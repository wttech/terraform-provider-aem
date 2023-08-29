package client

import "io"

type AWSSSMClient struct {
	InstanceID string
	Region     string
}

func (A AWSSSMClient) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMClient) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMClient) Invoke(args []string, input []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMClient) Call(args []string, input io.ReadCloser) (io.ReadCloser, error) {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMClient) CopyFile(localPath string, remotePath string) error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMClient) WriteFile(file io.ReadCloser, remotePath string) error {
	//TODO implement me
	panic("implement me")
}
