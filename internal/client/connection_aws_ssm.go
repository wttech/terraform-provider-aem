package client

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"os"
	"strings"
)

type AWSSSMConnection struct {
	user       string
	instanceId string
	region     string
	ssmClient  *ssm.SSM
	sessionId  *string
}

func (a *AWSSSMConnection) Info() string {
	return fmt.Sprintf("ssm: instance='%s', region='%s'", a.instanceId, a.region)
}

func (a *AWSSSMConnection) User() string {
	return a.user
}
func (a *AWSSSMConnection) Connect() error {
	// Create an AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(a.region),
	})
	if err != nil {
		return fmt.Errorf("ssm: error creating AWS session: %v", err)
	}

	// Connect to AWS instance using SSM
	ssmClient := ssm.New(sess)
	startSessionInput := &ssm.StartSessionInput{
		Target: aws.String(a.instanceId),
	}

	startSessionOutput, err := ssmClient.StartSession(startSessionInput)
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

	_, err := a.ssmClient.TerminateSession(terminateSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error terminating session: %v", err)
	}

	return nil
}

func (a *AWSSSMConnection) Command(cmdLine []string) ([]byte, error) {
	// Execute command on the remote instance
	command := aws.String(strings.Join(cmdLine, " "))
	runCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []*string{aws.String(a.instanceId)},
		Parameters: map[string][]*string{
			"commands": {command},
		},
	}

	runCommandOutput, err := a.ssmClient.SendCommand(runCommandInput)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	commandId := runCommandOutput.Command.CommandId

	// Wait for command completion
	err = a.ssmClient.WaitUntilCommandExecuted(&ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.instanceId),
	})
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	getCommandOutput, err := a.ssmClient.GetCommandInvocation(&ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.instanceId),
	})
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

	command := fmt.Sprintf("echo -n %s | base64 -d > %s", encodedContent, remotePath)
	_, err = a.Command(strings.Split(command, " "))
	return err
}
