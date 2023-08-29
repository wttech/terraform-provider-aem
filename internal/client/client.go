package client

import (
	"fmt"
	"github.com/spf13/cast"
)

type ClientManager struct{}

var ClientManagerDefault = &ClientManager{}

type Client interface {
	Connect() error
	Disconnect() error
	Run(cmd string) ([]byte, error)
	CopyFile(localPath string, remotePath string) error
}

func (c ClientManager) Make(typeName string, settings map[string]string) (Client, error) {
	switch typeName {
	case "ssh":
		return &SSHClient{
			host:           settings["host"],
			user:           settings["user"],
			privateKeyFile: settings["private_key_file"],
			port:           cast.ToInt(settings["port"]),
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
