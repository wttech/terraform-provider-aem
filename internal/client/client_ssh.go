package client

import (
	"fmt"
	"github.com/melbahja/goph"
	"github.com/spf13/cast"
	"golang.org/x/crypto/ssh"
	"path/filepath"
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
	// TODO loop until establishment of connection
	client, err := goph.NewConn(&goph.Config{
		User:     s.user,
		Addr:     s.host,
		Port:     cast.ToUint(s.port),
		Auth:     auth,
		Timeout:  goph.DefaultTimeout,
		Callback: ssh.InsecureIgnoreHostKey(), // TODO make it secure by default
	})
	if err != nil {
		return fmt.Errorf("SSH: cannot connect to host '%s': %w", s.host, err)
	}
	s.client = client
	return nil
}

func (s *SSHClient) Disconnect() error {
	if s.client == nil {
		return nil
	}
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("SSH: cannot disconnect from host '%s': %w", s.host, err)
	}
	return nil
}

func (s *SSHClient) Run(cmd string) ([]byte, error) {
	out, err := s.client.Run(cmd)
	if err != nil {
		if len(out) > 0 { // TODO rethink error handling
			return nil, fmt.Errorf("SSH: cannot run command '%s' on host '%s': %w\n\n%s", cmd, s.host, err, string(out))
		}
		return nil, fmt.Errorf("SSH: cannot run command '%s' on host '%s': %w", cmd, s.host, err)
	}
	return out, nil
}

func (s *SSHClient) CopyFile(localPath string, remotePath string) error {
	dir := filepath.Dir(remotePath)
	if _, err := s.client.Run(fmt.Sprintf("mkdir -p %s", dir)); err != nil {
		return fmt.Errorf("SSH: cannot ensure directory '%s' before copying file: %w", dir, err)
	}
	if err := s.client.Upload(localPath, remotePath); err != nil {
		return fmt.Errorf("SSH: cannot copy local file '%s' to remote path '%s' on host '%s': %w", localPath, remotePath, s.host, err)
	}
	return nil
}
