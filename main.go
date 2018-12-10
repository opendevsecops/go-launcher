package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"os"
	"os/exec"
	"syscall"
)

const ERR_NO_ARGS = 1
const ERR_CMD_START = 2
const ERR_CMD_WAIT = 3
const ERR_AWS_S3_UPLOAD = 4
const ERR_AWS_SNS_PUBLISH = 5

func exit(code int) {
	os.Exit(code)
}

func exitExt(code int, message string) {
	if os.Getenv("DEBUG") != "" {
		print(fmt.Sprintf("[x] %s", message))
	}

	exit(code)
}

func exitErr(code int, err error) {
	exitExt(code, err.Error())
}

func exitErrExt(code int, err error, message string) {
	exitExt(code, fmt.Sprintf("%s: %s", message, err.Error()))
}

func execCommand(command string, args ...string) (int, error) {
	cmd := exec.Command(command, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); nil != err {
		return ERR_CMD_START, err
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
		}

		return ERR_CMD_WAIT, err
	}

	return 0, nil
}

func main() {
	if len(os.Args) < 2 {
		exitExt(ERR_NO_ARGS, "Not enough command line arguments")
	}

	if os.Getenv("PROGRESS_SNS_ARN") != "" {
		svc := sns.New(session.New())

		pub := func(step int, steps int) {
			params := &sns.PublishInput{
				Message:  aws.String(fmt.Sprintf(`{"step": %d, "steps": %d, "status": ""}`, step, steps)),
				TopicArn: aws.String(os.Getenv("PROGRESS_SNS_ARN")),
			}

			if _, err := svc.Publish(params); err != nil {
				exitErrExt(ERR_AWS_SNS_PUBLISH, err, "Cannot publish SNS progress")
			}
		}

		pub(0, 1)

		defer pub(1, 1)
	}

	code, err := execCommand(os.Args[1], os.Args[2:]...)

	if err != nil {
		exitErr(code, err)
	} else {
		exit(code)
	}
}
