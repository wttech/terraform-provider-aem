package client

import (
	"fmt"
	"os/exec"
	"strings"
)

type LocalConnection struct {
}

func (a *LocalConnection) Info() string {
	return "local environment"
}

func (a *LocalConnection) User() string {
	return ""
}

func (a *LocalConnection) Connect() error {
	return nil
}

func (a *LocalConnection) Disconnect() error {
	return nil
}

func (a *LocalConnection) Command(cmdLine []string) ([]byte, error) {
	var alterCmdLine []string
	for _, cmdElem := range cmdLine {
		alterCmdLine = append(alterCmdLine, strings.Trim(cmdElem, `"`))
	}
	cmd := exec.Command(alterCmdLine[0], alterCmdLine[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("local: error executing command: %v", err)
	}
	return output, nil
}

func (a *LocalConnection) CopyFile(localPath string, remotePath string) error {
	cmd := fmt.Sprintf("cp %s %s", localPath, remotePath)
	cmdLine := []string{"sh", "-c", "\"" + cmd + "\""}
	_, err := a.Command(cmdLine)
	return err
}
