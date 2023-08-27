package client

import (
	"io"
)

type AWSSSMClient struct {
	InstanceID string
	Region     string
}

func (c AWSSSMClient) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (c AWSSSMClient) Invoke(args []string, input io.Reader) (io.ReadCloser, error) {
	//TODO implement me
	panic("implement me")
}

func (c AWSSSMClient) Disconnect() error {
	//TODO implement me
	panic("implement me")
}
