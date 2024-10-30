package infrastructure

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/ssh"
)

type Bash struct {
	server     Server
	sshClient  *ssh.Client
	sshSession *ssh.Session
	log        *log.Helper
}

type Server struct {
	Name       string `json:"name,omitempty"`
	User       string `json:"user,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int32  `json:"port,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

func NewBash(server Server, log *log.Helper) *Bash {
	return &Bash{server: server, log: log}
}

func (s *Bash) connections() (*ssh.Session, error) {
	signer, err := ssh.ParsePrivateKey([]byte(s.server.PrivateKey))
	if err != nil {
		return nil, err
	}
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.server.Host, s.server.Port), &ssh.ClientConfig{
		User:            s.server.User,
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
	s.sshClient = sshClient
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	s.sshSession = session
	return session, nil
}

func (s *Bash) close() {
	if s.sshSession != nil {
		s.sshSession.Close()
	}
	if s.sshClient != nil {
		s.sshClient.Close()
	}
}

func (s *Bash) Run(command string, args ...string) (stdout string, err error) {
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	command = fmt.Sprintf("sudo %s", command)
	s.log.Info(fmt.Sprintf("%s/%s run command: %s", s.server.Name, s.server.Host, command))
	session, err := s.connections()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer s.close()

	// Set up pipes for stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run the command
	err = session.Run(command)
	stdout = stdoutBuf.String()
	stderr := stderrBuf.String()
	if stderr != "" {
		s.log.Warnf("command execution produced stderr: %s", stderr)
	}

	return stdout, nil
}

// RunWithLogging runs a command and logs its output
func (s *Bash) RunWithLogging(command string, args ...string) error {
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	command = fmt.Sprintf("sudo %s", command)
	s.log.Info(fmt.Sprintf("%s/%s run command: %s", s.server.Name, s.server.Host, command))
	session, err := s.connections()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer s.close()

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

	// Function to read from a pipe and log output
	logOutput := func(pipe io.Reader, prefix string, logFunc func(args ...any)) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			logFunc(fmt.Sprintf("%s: %s", prefix, scanner.Text()))
		}
	}

	// Start goroutines to read and log stdout and stderr
	go logOutput(stdout, "STDOUT", s.log.Info)
	go logOutput(stderr, "STDERR", s.log.Warn)

	// Wait for the command to finish
	if err := session.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func (s *Bash) GetUserHome() (string, error) {
	homePath, err := s.Run("echo", "$HOME")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(homePath), nil
}

func (s *Bash) GetRootHome() (string, error) {
	homePath, err := s.Run("grep '^root:' /etc/passwd | cut -d: -f6")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(homePath), nil
}
