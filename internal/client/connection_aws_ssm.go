package client

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/melbahja/goph"
	"os"
	"strings"
)

type AWSSSMConnection struct {
	instanceId string
	region     string
	ssmClient  *ssm.SSM
	sessionId  *string
}

func (a *AWSSSMConnection) Info() string {
	return fmt.Sprintf("ssm: instance='%s', region='%s'", a.instanceId, a.region)
}

func (a *AWSSSMConnection) User() string {
	return "aem" // does not impact the connection, used as default user for systemd only
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

func (a *AWSSSMConnection) Command(cmdLine []string) (*goph.Cmd, error) {
	// Execute command on the remote instance
	runCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []*string{aws.String(a.instanceId)},
		Parameters: map[string][]*string{
			"commands": aws.StringSlice(cmdLine),
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

	// Transform the SSM command output into a goph.Cmd structure
	parts := strings.Fields(*getCommandOutput.StandardOutputContent)
	if len(parts) < 2 {
		return nil, fmt.Errorf("ssm: unexpected command output format")
	}

	gophCommand := goph.Cmd{
		Path: parts[0],
		Args: parts[1:],
		Env:  os.Environ(),
	}

	return &gophCommand, nil
}

func (a *AWSSSMConnection) CopyFile(localPath string, remotePath string) error {
	// Upload file to the remote instance using SSM Parameter Store
	fileContent, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("ssm: error reading local file: %v", err)
	}

	putParameterInput := &ssm.PutParameterInput{
		Name:      aws.String(remotePath),
		Value:     aws.String(string(fileContent)),
		Type:      aws.String("SecureString"),
		Overwrite: aws.Bool(true),
	}

	_, err = a.ssmClient.PutParameter(putParameterInput)
	if err != nil {
		return fmt.Errorf("ssm: error uploading file to the instance: %v", err)
	}

	return nil
}
