package client

import (
	"fmt"
	"github.com/melbahja/goph"
)

type SSHClient struct {
	client *goph.Client

	host           string
	user           string
	passphrase     string
	privateKeyFile string
	port           int
}

func (s *SSHClient) Connect() error {
	auth, err := goph.Key(s.privateKeyFile, s.passphrase)
	if err != nil {
		return fmt.Errorf("SSH: cannot get auth using private key '%s': %w", s.privateKeyFile, err)
	}
	client, err := goph.New(s.user, s.host, auth)
	if err != nil {
		return fmt.Errorf("SSH: cannot connect to host '%s': %w", s.host, err)
	}
	s.client = client
	return nil
}

func (s *SSHClient) Disconnect() error {
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("SSH: cannot disconnect from host '%s': %w", s.host, err)
	}
	return nil
}

func (s *SSHClient) Run(cmd string) ([]byte, error) {
	out, err := s.client.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("SSH: cannot run command '%s' on host '%s': %w", cmd, s.host, err)
	}
	return out, nil
}

func (s *SSHClient) CopyFile(localPath string, remotePath string) error {
	err := s.client.Upload(localPath, remotePath)
	if err != nil {
		return fmt.Errorf("SSH: cannot local file '%s' to remote path '%s' on host '%s': %w", localPath, remotePath, s.host, err)
	}
	return nil
}
