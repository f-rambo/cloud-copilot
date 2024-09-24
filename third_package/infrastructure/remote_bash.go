package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

type RemoteBash struct {
	node      *biz.Node
	sshClient *ssh.Client
	log       *log.Helper
}

func NewRemoteBash(ctx context.Context, node *biz.Node, log *log.Helper) (*RemoteBash, error) {
	signer, err := ssh.ParsePrivateKey([]byte(node.PrivateKey))
	if err != nil {
		return nil, err
	}
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", node.InternalIP, node.SshPort), &ssh.ClientConfig{
		User:            node.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			// ssh.Password("your_password"),
			ssh.PublicKeys(signer),
		},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &RemoteBash{
		sshClient: sshClient,
		node:      node,
		log:       log,
	}, nil
}

func (s *RemoteBash) Close() {
	if s.sshClient != nil {
		s.sshClient.Close()
	}
}

func (s *RemoteBash) Run(command string) (stdout string, stderr string, err error) {
	s.log.Info(fmt.Sprintf("%s/%s run command: %s", s.node.Name, s.node.InternalIP, command))
	session, err := s.sshClient.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Set up pipes for stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run the command
	err = session.Run(command)
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		return stdout, stderr, fmt.Errorf("command execution failed: %v", err)
	}

	return stdout, stderr, nil
}

// run shell content
func (s *RemoteBash) RunShell(shell string) error {
	return s.RunWithLogging(fmt.Sprintf("bash -c '%s'", shell))
}

// run Command With Logging
func (s *RemoteBash) RunWithLogging(command string) error {
	s.log.Info(fmt.Sprintf("%s/%s run command: %s", s.node.Name, s.node.InternalIP, command))
	session, err := s.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Set up pipes for stdout and stderr
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	err = session.Start(command)
	if err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	// Use errgroup to wait for both goroutines to finish
	errgroup := new(errgroup.Group)

	// Function to read from a pipe and log output
	logOutput := func(pipe io.Reader, prefix string, logFunc func(args ...any)) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			logFunc(fmt.Sprintf("%s: %s", prefix, scanner.Text()))
		}
	}

	// Start goroutines to read and log stdout and stderr
	errgroup.Go(func() error {
		logOutput(stdout, "STDOUT", s.log.Info)
		return nil
	})
	errgroup.Go(func() error {
		logOutput(stderr, "STDERR", s.log.Error)
		return nil
	})

	// Wait for the command to finish
	err = session.Wait()
	if err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	// Wait for both goroutines to finish logging
	err = errgroup.Wait()
	if err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	return nil
}
