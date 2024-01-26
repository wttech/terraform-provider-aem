package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"os"
	"strings"
	"time"
)

type AWSSSMConnection struct {
	InstanceID string
	Region     string
	Client     *ssm.Client
	SessionId  *string

	context context.Context
}

func (a *AWSSSMConnection) Info() string {
	return fmt.Sprintf("ssm: instance_id='%s', region='%s'", a.InstanceID, a.Region)
}

func (a *AWSSSMConnection) User() string {
	out, err := a.Command([]string{"whoami"})
	if err != nil {
		panic(fmt.Sprintf("ssm: cannot determine connected user: %s", err))
	}
	return strings.TrimSpace(string(out))
}

func (a *AWSSSMConnection) Connect() error {
	var optFns []func(*config.LoadOptions) error
	if a.Region != "" {
		optFns = append(optFns, config.WithRegion(a.Region))
	}

	cfg, err := config.LoadDefaultConfig(a.context, optFns...)
	if err != nil {
		return err
	}

	client := ssm.NewFromConfig(cfg)
	startSessionInput := &ssm.StartSessionInput{Target: aws.String(a.InstanceID)}

	startSessionOutput, err := client.StartSession(a.context, startSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error starting session: %v", err)
	}

	a.Client = client
	a.SessionId = startSessionOutput.SessionId

	return nil
}

func (a *AWSSSMConnection) Disconnect() error {
	// Disconnect from the session
	terminateSessionInput := &ssm.TerminateSessionInput{SessionId: a.SessionId}

	_, err := a.Client.TerminateSession(a.context, terminateSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error terminating session: %v", err)
	}

	return nil
}

func (a *AWSSSMConnection) Command(cmdLine []string) ([]byte, error) {
	command := strings.Join(cmdLine, " ")
	runCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []string{a.InstanceID},
		Parameters: map[string][]string{
			"commands": {command},
		},
	}
	runOut, err := a.Client.SendCommand(a.context, runCommandInput)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	commandId := runOut.Command.CommandId
	invocationIn := &ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.InstanceID),
	}
	waiter := ssm.NewCommandExecutedWaiter(a.Client)
	_, err = waiter.WaitForOutput(a.context, invocationIn, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	invocationOut, err := a.Client.GetCommandInvocation(a.context, invocationIn)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	return []byte(*invocationOut.StandardOutputContent), nil
}

func (a *AWSSSMConnection) CopyFile(localPath string, remotePath string) error {
	fileContent, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("ssm: error reading local file: %v", err)
	}
	encodedContent := base64.StdEncoding.EncodeToString(fileContent)

	cmd := fmt.Sprintf("echo -n %s | base64 -d > %s", encodedContent, remotePath)
	_, err = a.Command([]string{cmd})
	return err
}
