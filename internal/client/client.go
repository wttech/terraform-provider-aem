package client

import (
	"fmt"
	"github.com/spf13/cast"
	"io"
)

type ClientManager struct{}

var ClientManagerDefault = &ClientManager{}

type Client interface {
	Connect() error
	Disconnect() error
	Invoke(args []string, input []byte) ([]byte, error)
	Call(args []string, input io.ReadCloser) (io.ReadCloser, error)
	CopyFile(localPath string, remotePath string) error
	WriteFile(file io.ReadCloser, remotePath string) error
}

func (c ClientManager) Make(typeName string, settings map[string]string) (Client, error) {
	switch typeName {
	case "ssh":
		return &SSHClient{
			Host:       settings["host"],
			User:       settings["user"],
			PrivateKey: []byte(settings["private_key"]),
			Port:       cast.ToInt(settings["port"]),
		}, nil
	case "aws-ssm":
		return &AWSSSMClient{
			InstanceID: settings["instance_id"],
			Region:     settings["region"],
		}, nil
	default:
		return nil, fmt.Errorf("unknown AEM client type: %s", typeName)
	}
}
