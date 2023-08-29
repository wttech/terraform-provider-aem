package client

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

func (A AWSSSMClient) Run(cmd string) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (A AWSSSMClient) CopyFile(localPath string, remotePath string) error {
	//TODO implement me
	panic("implement me")
}
