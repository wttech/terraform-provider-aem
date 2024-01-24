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
	user       string
	instanceId string
	region     string
	ssmClient  *ssm.Client
	sessionId  *string
}

func (a *AWSSSMConnection) Info() string {
	return fmt.Sprintf("ssm: instance='%s', region='%s'", a.instanceId, a.region)
}

func (a *AWSSSMConnection) User() string {
	return a.user
}

func (a *AWSSSMConnection) Connect() error {
	// Specify the AWS region
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(a.region))
	if err != nil {
		return err
	}

	// Create an SSM client
	ssmClient := ssm.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("ssm: error creating AWS session: %v", err)
	}

	startSessionInput := &ssm.StartSessionInput{
		Target: aws.String(a.instanceId),
	}

	startSessionOutput, err := ssmClient.StartSession(context.Background(), startSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error starting session: %v", err)
	}

	a.ssmClient = ssmClient
	a.sessionId = startSessionOutput.SessionId

	return nil
}

func (a *AWSSSMConnection) Disconnect() error {
	// Disconnect from the session
	terminateSessionInput := &ssm.TerminateSessionInput{
		SessionId: a.sessionId,
	}

	_, err := a.ssmClient.TerminateSession(context.Background(), terminateSessionInput)
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
		InstanceIds:  []string{a.instanceId},
		Parameters: map[string][]string{
			"commands": {command},
		},
	}

	runCommandOutput, err := a.ssmClient.SendCommand(context.Background(), runCommandInput)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	commandId := runCommandOutput.Command.CommandId

	commandInvocationInput := &ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.instanceId),
	}

	waiter := ssm.NewCommandExecutedWaiter(a.ssmClient)
	_, err = waiter.WaitForOutput(context.Background(), commandInvocationInput, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	getCommandOutput, err := a.ssmClient.GetCommandInvocation(context.Background(), commandInvocationInput)
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
	cmdLine := []string{"sh", "-c", "\"" + cmd + "\""}
	_, err = a.Command(cmdLine)
	return err
}
