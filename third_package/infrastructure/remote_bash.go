package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/ssh"
)

type RemoteBash struct {
	server    Server
	sshClient *ssh.Client
	log       *log.Helper
}

type Server struct {
	Name       string `json:"name,omitempty"`
	User       string `json:"user,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int32  `json:"port,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

func NewRemoteBash(ctx context.Context, server Server, log *log.Helper) (*RemoteBash, error) {
	signer, err := ssh.ParsePrivateKey([]byte(server.PrivateKey))
	if err != nil {
		return nil, err
	}
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port), &ssh.ClientConfig{
		User:            server.User,
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
		server:    server,
		log:       log,
	}, nil
}

func (s *RemoteBash) Close() {
	if s.sshClient != nil {
		s.sshClient.Close()
	}
}
func (s *RemoteBash) Run(command string, args ...string) (stdout string, err error) {
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	s.log.Info(fmt.Sprintf("%s/%s run command: %s", s.server.Name, s.server.Host, command))
	session, err := s.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up pipes for stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run the command
	err = session.Run(command)
	stdout = stdoutBuf.String()
	stderr := stderrBuf.String()

	if err != nil {
		return stdout, fmt.Errorf("command execution failed: %w, stdout: %s, stderr: %s", err, stdout, stderr)
	}
	if stderr != "" {
		return stdout, fmt.Errorf("command execution produced stderr: %s", stderr)
	}

	return stdout, nil
}

// RunWithLogging runs a command and logs its output
func (s *RemoteBash) RunWithLogging(command string, args ...string) error {
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	s.log.Info(fmt.Sprintf("%s/%s run command: %s", s.server.Name, s.server.Host, command))
	session, err := s.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up pipes for stdout and stderr
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := session.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var stderrOutput bytes.Buffer
	var stdoutOutput bytes.Buffer

	// Function to read from a pipe and log output
	logOutput := func(pipe io.Reader, prefix string, logFunc func(args ...any), output *bytes.Buffer) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			logFunc(fmt.Sprintf("%s: %s", prefix, line))
			output.WriteString(line + "\n")
		}
	}

	// Start goroutines to read and log stdout and stderr
	go logOutput(stdout, "STDOUT", s.log.Info, &stdoutOutput)
	go logOutput(stderr, "STDERR", s.log.Error, &stderrOutput)

	// Wait for the command to finish
	err = session.Wait()

	if err != nil {
		return fmt.Errorf("command execution failed: %w, stdout: %s, stderr: %s", err, stdoutOutput.String(), stderrOutput.String())
	}

	if stderrOutput.Len() > 0 {
		return fmt.Errorf("command wrote to stderr: %s", stderrOutput.String())
	}

	return nil
}
