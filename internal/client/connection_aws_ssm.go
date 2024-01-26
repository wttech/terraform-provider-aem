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
	instanceID    string
	region        string
	outputTimeout time.Duration
	client        *ssm.Client
	sessionId     *string
	context       context.Context
}

func (a *AWSSSMConnection) Info() string {
	region := a.region
	if region == "" {
		region = "<default>"
	}
	return fmt.Sprintf("ssm: instance_id='%s', region='%s'", a.instanceID, region)
}

func (a *AWSSSMConnection) User() string {
	out, err := a.Command([]string{"whoami"})
	if err != nil {
		panic(fmt.Sprintf("ssm: cannot determine connected user: %s", err))
	}
	return strings.TrimSpace(string(out))
}

func (a *AWSSSMConnection) Connect() error {
	if a.outputTimeout == 0 {
		a.outputTimeout = time.Hour
	}

	var optFns []func(*config.LoadOptions) error
	if a.region != "" {
		optFns = append(optFns, config.WithRegion(a.region))
	}

	cfg, err := config.LoadDefaultConfig(a.context, optFns...)
	if err != nil {
		return err
	}

	client := ssm.NewFromConfig(cfg)
	startSessionInput := &ssm.StartSessionInput{Target: aws.String(a.instanceID)}

	startSessionOutput, err := client.StartSession(a.context, startSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error starting session: %v", err)
	}

	a.client = client
	a.sessionId = startSessionOutput.SessionId

	return nil
}

func (a *AWSSSMConnection) Disconnect() error {
	// Disconnect from the session
	terminateSessionInput := &ssm.TerminateSessionInput{SessionId: a.sessionId}

	_, err := a.client.TerminateSession(a.context, terminateSessionInput)
	if err != nil {
		return fmt.Errorf("ssm: error terminating session: %v", err)
	}

	return nil
}

func (a *AWSSSMConnection) Command(cmdLine []string) ([]byte, error) {
	command := strings.Join(cmdLine, " ")
	runCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []string{a.instanceID},
		Parameters: map[string][]string{
			"commands": {command},
		},
	}
	runOut, err := a.client.SendCommand(a.context, runCommandInput)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	commandId := runOut.Command.CommandId
	invocationIn := &ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.instanceID),
	}
	waiter := ssm.NewCommandExecutedWaiter(a.client)
	_, err = waiter.WaitForOutput(a.context, invocationIn, a.outputTimeout)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	invocationOut, err := a.client.GetCommandInvocation(a.context, invocationIn)
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
