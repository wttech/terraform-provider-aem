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
	instanceID           string
	region               string
	client               *ssm.Client
	sessionId            *string
	context              context.Context
	commandOutputTimeout time.Duration
	commandWaitMax       time.Duration
	commandWaitMin       time.Duration
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
	if a.commandOutputTimeout == 0 {
		a.commandOutputTimeout = 5 * time.Hour
	}
	if a.commandWaitMin == 0 {
		a.commandWaitMin = 5 * time.Millisecond
	}
	if a.commandWaitMax == 0 {
		a.commandWaitMax = 5 * time.Second
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
	sessionIn := &ssm.StartSessionInput{Target: aws.String(a.instanceID)}
	sessionOut, err := client.StartSession(a.context, sessionIn)
	if err != nil {
		return fmt.Errorf("ssm: error starting session: %v", err)
	}

	a.client = client
	a.sessionId = sessionOut.SessionId

	return nil
}

func (a *AWSSSMConnection) Disconnect() error {
	sessionIn := &ssm.TerminateSessionInput{SessionId: a.sessionId}
	_, err := a.client.TerminateSession(a.context, sessionIn)
	if err != nil {
		return fmt.Errorf("ssm: error terminating session: %v", err)
	}

	return nil
}

func (a *AWSSSMConnection) Command(cmdLine []string) ([]byte, error) {
	command := strings.Join(cmdLine, " ")
	commandIn := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []string{a.instanceID},
		Parameters: map[string][]string{
			"commands": {command},
		},
	}
	runOut, err := a.client.SendCommand(a.context, commandIn)
	if err != nil {
		return nil, fmt.Errorf("ssm: error executing command: %v", err)
	}

	commandId := runOut.Command.CommandId
	invocationIn := &ssm.GetCommandInvocationInput{
		CommandId:  commandId,
		InstanceId: aws.String(a.instanceID),
	}
	var optFns []func(opt *ssm.CommandExecutedWaiterOptions)
	if a.commandWaitMax > 0 && a.commandWaitMin > 0 {
		optFns = []func(opt *ssm.CommandExecutedWaiterOptions){func(opt *ssm.CommandExecutedWaiterOptions) {
			opt.MinDelay = a.commandWaitMin
			opt.MaxDelay = a.commandWaitMax
		}}
	}

	waiter := ssm.NewCommandExecutedWaiter(a.client, optFns...)
	invocationOut, err := waiter.WaitForOutput(a.context, invocationIn, a.commandOutputTimeout)
	if err != nil {
		invocationOut, err = a.client.GetCommandInvocation(a.context, invocationIn)
		if invocationOut != nil {
			return nil, fmt.Errorf("ssm: error executing command: %v", *invocationOut.StandardErrorContent)
		} else if err != nil {
			return nil, fmt.Errorf("ssm: error executing command: %v", err)
		}
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
