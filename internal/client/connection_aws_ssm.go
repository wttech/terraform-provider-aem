package client

type AWSSSMConnection struct {
	InstanceID string
	Region     string
}

func (A AWSSSMConnection) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) Run(cmd string) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMConnection) CopyFile(localPath string, remotePath string) error {
	//TODO implement me
	panic("implement me")
}
