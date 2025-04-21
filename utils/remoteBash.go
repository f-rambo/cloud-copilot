package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type RemoteBash struct {
	server     Server
	shellDir   string
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

func NewRemoteBash(server Server, shellDir string, log *log.Helper) *RemoteBash {
	return &RemoteBash{server: server, shellDir: shellDir, log: log}
}

func (s *RemoteBash) connections() (*ssh.Session, error) {
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

func (s *RemoteBash) close() {
	if s.sshSession != nil {
		s.sshSession.Close()
	}
	if s.sshClient != nil {
		s.sshClient.Close()
	}
}

func (s *RemoteBash) Run(command string, args ...string) (stdout string, err error) {
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
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
func (s *RemoteBash) RunWithLogging(command string, args ...string) error {
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
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

func (s *RemoteBash) SftpFile(localFile, remoteFile string) error {
	_, err := s.connections()
	if err != nil {
		return err
	}
	defer s.close()
	sftpClient, err := sftp.NewClient(s.sshClient)
	if err != nil {
		return errors.Wrap(err, "failed to create sftp client")
	}
	defer sftpClient.Close()
	srcFile, err := os.Open(localFile)
	if err != nil {
		return errors.Wrap(err, "failed to open local file")
	}
	defer srcFile.Close()
	dstFile, err := sftpClient.Create(remoteFile)
	if err != nil {
		return errors.Wrap(err, "failed to create destination file")
	}
	defer dstFile.Close()
	bytesCopied, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		return errors.Wrap(err, "failed to copy file")
	}
	s.log.Infof("Copied %d bytes from %s to %s", bytesCopied, localFile, remoteFile)
	return nil
}

func (s *RemoteBash) GetUserHome() (string, error) {
	homePath, err := s.Run("echo", "$HOME")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(homePath), nil
}

func (s *RemoteBash) GetRootHome() (string, error) {
	homePath, err := s.Run("grep '^root:' /etc/passwd | cut -d: -f6")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(homePath), nil
}

func (s *RemoteBash) ExecShellLogging(shellName string, args ...string) error {
	userHome, err := s.GetUserHome()
	if err != nil {
		return err
	}
	execShellPath := filepath.Join(userHome, s.shellDir, shellName)
	localShellPath := filepath.Join(s.shellDir, shellName)
	_, err = s.Run(fmt.Sprintf("mkdir -p %s", filepath.Join(userHome, s.shellDir)))
	if err != nil {
		return err
	}
	shellExists, err := s.Run("test -f " + execShellPath + " && echo 1 || echo 0")
	if err != nil {
		return err
	}
	if strings.TrimSpace(shellExists) == "0" {
		s.log.Info(fmt.Sprintf("shell %s not exists, copy from %s", execShellPath, localShellPath))
		if err := s.SftpFile(localShellPath, execShellPath); err != nil {
			return err
		}
	}
	return s.RunWithLogging(fmt.Sprintf("sudo bash %s", execShellPath), args...)
}

func (s *RemoteBash) ExecShell(shellName string, args ...string) (stdout string, err error) {
	userHome, err := s.GetUserHome()
	if err != nil {
		return "", err
	}
	execShellPath := filepath.Join(userHome, s.shellDir, shellName)
	localShellPath := filepath.Join(s.shellDir, shellName)
	_, err = s.Run(fmt.Sprintf("mkdir -p %s", filepath.Join(userHome, s.shellDir)))
	if err != nil {
		return "", err
	}
	shellExists, err := s.Run("test -f " + execShellPath + " && echo 1 || echo 0")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(shellExists) == "0" {
		s.log.Info(fmt.Sprintf("shell %s not exists, copy from %s", execShellPath, localShellPath))
		if err := s.SftpFile(localShellPath, execShellPath); err != nil {
			return "", err
		}
	}
	return s.Run(fmt.Sprintf("sudo bash %s", execShellPath), args...)
}

func (s *RemoteBash) SftpDirectory(localDir, remoteDir string) error {
	_, err := s.connections()
	if err != nil {
		return err
	}
	defer s.close()

	sftpClient, err := sftp.NewClient(s.sshClient)
	if err != nil {
		return errors.Wrap(err, "failed to create sftp client")
	}
	defer sftpClient.Close()

	// 确保远程目录存在
	err = sftpClient.MkdirAll(remoteDir)
	if err != nil {
		return errors.Wrap(err, "failed to create remote directory")
	}

	// 遍历本地目录
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localDir, localPath)
		if err != nil {
			return errors.Wrap(err, "failed to get relative path")
		}

		// 构建远程路径
		remotePath := filepath.Join(remoteDir, relPath)

		if info.IsDir() {
			// 如果是目录，在远程创建对应的目录
			return sftpClient.MkdirAll(remotePath)
		}

		// 如果是文件，复制文件内容
		srcFile, err := os.Open(localPath)
		if err != nil {
			return errors.Wrap(err, "failed to open local file")
		}
		defer srcFile.Close()

		dstFile, err := sftpClient.Create(remotePath)
		if err != nil {
			return errors.Wrap(err, "failed to create remote file")
		}
		defer dstFile.Close()

		bytesCopied, err := dstFile.ReadFrom(srcFile)
		if err != nil {
			return errors.Wrap(err, "failed to copy file")
		}

		s.log.Infof("Copied %d bytes from %s to %s", bytesCopied, localPath, remotePath)
		return nil
	})
}
