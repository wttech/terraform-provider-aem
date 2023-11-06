package client

import "github.com/melbahja/goph"

type AWSSSMConnection struct {
	InstanceID string
	Region     string
}

func (A AWSSSMConnection) Info() string {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) Command(cmdLine []string) (*goph.Cmd, error) {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) CopyFile(localPath string, remotePath string) error {
	//TODO implement me
	panic("implement me")
}
