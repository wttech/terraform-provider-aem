package client

import (
	"fmt"
	"github.com/spf13/cast"
)

func (c ClientManager) Make(typeName string, settings map[string]string) (*Client, error) {
	connection, err := c.connection(typeName, settings)
	if err != nil {
		return nil, err
	}
	return &Client{
		typeName:   typeName,
		settings:   settings,
		connection: connection,

		Env: map[string]string{},
	}, nil
}

func (c ClientManager) Use(typeName string, settings map[string]string, callback func(c Client) error) error {
	client, err := c.Make(typeName, settings)
	if err != nil {
		return err
	}
	return client.Use(callback)
}

func (c ClientManager) connection(typeName string, settings map[string]string) (Connection, error) {
	switch typeName {
	case "ssh":
		return &SSHConnection{
			host:                 settings["host"],
			user:                 settings["user"],
			privateKey:           settings["private_key"],
			privateKeyPassphrase: settings["private_key_passphrase"],
			port:                 cast.ToInt(settings["port"]),
			secure:               cast.ToBool(settings["secure"]),
		}, nil
	case "aws-ssm":
		return &AWSSSMConnection{
			user:       settings["user"],
			instanceId: settings["instance_id"],
			region:     settings["region"],
		}, nil
	}
	return nil, fmt.Errorf("unknown AEM client type: %s", typeName)
}

type ClientManager struct{}

var ClientManagerDefault = &ClientManager{}
