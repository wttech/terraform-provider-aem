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
	Context    context.Context
}

func (a *AWSSSMConnection) Info() string {
	return fmt.Sprintf("ssm: instance_id='%s', region='%s'", a.InstanceID, a.Region)
}

func (a *AWSSSMConnection) User() string {
	out, _ := a.Command([]string{"whoami"})
	return strings.TrimSpace(string(out))
}

func (a *AWSSSMConnection) Connect() error {
	// Specify the AWS region
	a.Context = context.Background()
	cfg, err := config.LoadDefaultConfig(a.Context, config.WithRegion(a.Region))
	if err != nil {
		return err
	}

	// Create an SSM client
	client := ssm.NewFromConfig(cfg)
	startSessionInput := &ssm.StartSessionInput{Target: aws.String(a.InstanceID)}

	startSessionOutput, err := client.StartSession(a.Context, startSessionInput)
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

	_, err := a.Client.TerminateSession(a.Context, terminateSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error terminating session: %v", err)
	}

	return nil
}

func (a *AWSSSMConnection) Command(cmdLine []string) ([]byte, error) {
	// Execute command on the remote instance
	command := strings.Join(cmdLine, " ")
	runCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []string{a.InstanceID},
		Parameters: map[string][]string{
			"commands": {command},
		},
	}

	runCommandOutput, err := a.Client.SendCommand(a.Context, runCommandInput)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	commandId := runCommandOutput.Command.CommandId

	commandInvocationInput := &ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.InstanceID),
	}

	waiter := ssm.NewCommandExecutedWaiter(a.Client)
	_, err = waiter.WaitForOutput(a.Context, commandInvocationInput, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	getCommandOutput, err := a.Client.GetCommandInvocation(a.Context, commandInvocationInput)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	return []byte(*getCommandOutput.StandardOutputContent), nil
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
