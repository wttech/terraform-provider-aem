package client

type Connection interface {
	Info() string
	User() string
	Connect() error
	Disconnect() error
	Command(cmdLine []string) ([]byte, error)
	CopyFile(localPath string, remotePath string) error
}
