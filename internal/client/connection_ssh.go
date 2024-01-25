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

	host                 string
	user                 string
	passphrase           string
	privateKey           string
	privateKeyPassphrase string
	port                 int
	secure               bool
}

func (s *SSHConnection) Connect() error {
	if s.host == "" {
		return fmt.Errorf("ssh: host is required")
	}
	if s.user == "" {
		return fmt.Errorf("ssh: user is required")
	}
	if s.privateKey == "" {
		return fmt.Errorf("ssh: private key is required")
	}
	if s.port == 0 {
		s.port = 22
	}
	var (
		signer ssh.Signer
		err    error
	)
	if s.passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(s.privateKey), []byte(s.passphrase))
		if err != nil {
			return fmt.Errorf("ssh: cannot parse private key with passphrase: %w", err)
		}
	} else {
		signer, err = ssh.ParsePrivateKey([]byte(s.privateKey))
		if err != nil {
			return fmt.Errorf("ssh: cannot parse private key: %w", err)
		}
	}
	var callback ssh.HostKeyCallback
	if s.secure {
		callback = ssh.FixedHostKey(signer.PublicKey())
	} else {
		callback = ssh.InsecureIgnoreHostKey()
	}
	client, err := goph.NewConn(&goph.Config{
		User:     s.user,
		Addr:     s.host,
		Port:     cast.ToUint(s.port),
		Auth:     goph.Auth{ssh.PublicKeys(signer)},
		Timeout:  goph.DefaultTimeout,
		Callback: callback,
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

func (s *SSHConnection) Command(cmdLine []string) ([]byte, error) {
	name, args := s.splitCommandLine(cmdLine)
	cmd, err := s.client.Command(name, args...)
	if err != nil {
		return nil, fmt.Errorf("ssh: cannot create command '%s' for host '%s': %w", strings.Join(cmdLine, " "), s.host, err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) > 0 {
		return nil, fmt.Errorf("ssh: cannot run command '%s': %w\n\n%s", cmd, err, string(out))
	} else if err != nil {
		return nil, fmt.Errorf("ssh: cannot run command '%s': %w", cmd, err)
	}
	return out, nil
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
