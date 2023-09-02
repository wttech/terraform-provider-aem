package client

import (
	"github.com/melbahja/goph"
)

type Connection interface {
	Connect() error
	Disconnect() error
	Command(cmdLine []string) (*goph.Cmd, error)
	CopyFile(localPath string, remotePath string) error
}
