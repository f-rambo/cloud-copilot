package utils

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type Bash struct {
	log *log.Helper
}

func NewBash(log *log.Helper) *Bash {
	return &Bash{
		log: log,
	}
}

func (b *Bash) RunCommand(command string, args ...string) (output string, err error) {
	b.log.Infof("exec command: %s %s", command, strings.Join(args, " "))

	cmd := exec.Command(command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		return "", errors.Wrapf(err, "command failed: %s\nstdout: %s\nstderr: %s", command, stdoutStr, stderrStr)
	}

	if stderrStr != "" {
		log.Warnf("command wrote to stderr: %s", stderrStr)
	}

	return stdoutStr, nil
}

func (b *Bash) RunCommandWithLogging(command string, args ...string) error {
	b.log.Infof("exec command: %s %s", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Info(scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Warn(scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "command failed")
	}
	return nil
}
