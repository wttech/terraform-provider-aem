package client

import (
	"fmt"
	"github.com/melbahja/goph"
)

type AWSSSMConnection struct {
	instanceId string
	region     string
}

func (a *AWSSSMConnection) Info() string {
	return fmt.Sprintf("ssm: instance='%s', region='%s'", a.instanceId, a.region)
}

func (a *AWSSSMConnection) User() string {
	//TODO implement me
	panic("implement me")
}

func (a *AWSSSMConnection) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (a *AWSSSMConnection) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (a *AWSSSMConnection) Command(cmdLine []string) (*goph.Cmd, error) {
	//TODO implement me
	panic("implement me")
}

func (a *AWSSSMConnection) CopyFile(localPath string, remotePath string) error {
	//TODO implement me
	panic("implement me")
}
