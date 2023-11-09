package client

import (
	"fmt"
	"github.com/melbahja/goph"
	"github.com/spf13/cast"
	"golang.org/x/crypto/ssh"
	"strings"
)

type SSHConnection struct {
	client *goph.Client

	host           string
	user           string
	passphrase     string
	privateKeyFile string
	port           int
}

func (s *SSHConnection) Connect() error {
	auth, err := goph.Key(s.privateKeyFile, s.passphrase)
	if err != nil {
		return fmt.Errorf("ssh: cannot get auth using private key '%s': %w", s.privateKeyFile, err)
	}
	client, err := goph.NewConn(&goph.Config{
		User:     s.user,
		Addr:     s.host,
		Port:     cast.ToUint(s.port),
		Auth:     auth,
		Timeout:  goph.DefaultTimeout,
		Callback: ssh.InsecureIgnoreHostKey(), // TODO make it secure by default
	})
	if err != nil {
		return fmt.Errorf("ssh: cannot connect to host '%s': %w", s.host, err)
	}
	s.client = client
	return nil
}

func (s *SSHConnection) Info() string {
	return fmt.Sprintf("ssh: host='%s', user='%s', port='%d'", s.host, s.user, s.port)
}

func (s *SSHConnection) User() string {
	return s.user
}

func (s *SSHConnection) Disconnect() error {
	if s.client == nil {
		return nil
	}
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("ssh: cannot disconnect from host '%s': %w", s.host, err)
	}
	return nil
}

func (s *SSHConnection) Command(cmdLine []string) (*goph.Cmd, error) {
	name, args := s.splitCommandLine(cmdLine)
	cmd, err := s.client.Command(name, args...)
	if err != nil {
		return nil, fmt.Errorf("ssh: cannot create command '%s' for host '%s': %w", strings.Join(cmdLine, " "), s.host, err)
	}
	return cmd, nil
}

func (s *SSHConnection) splitCommandLine(cmdLine []string) (string, []string) {
	name := cmdLine[0]
	var args []string
	if len(cmdLine) > 1 {
		args = cmdLine[1:]
	}
	return name, args
}

func (s *SSHConnection) CopyFile(localPath string, remotePath string) error {
	if err := s.client.Upload(localPath, remotePath); err != nil {
		return fmt.Errorf("ssh: cannot copy local file '%s' to remote path '%s' on host '%s': %w", localPath, remotePath, s.host, err)
	}
	return nil
}
