package client

type Connection interface {
	Connect() error
	Disconnect() error
	Run(cmd string) ([]byte, error)
	CopyFile(localPath string, remotePath string) error
}
