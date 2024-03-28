package ansible

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

type GoAnsiblePkg struct {
	c                     *conf.Bootstrap
	logPrefix             string
	ansiblePlaybookBinary string
	LogChan               chan string
	cmdRunDir             string
	inventoryfile         string
	playbooks             []string
	env                   map[string]string
}

func NewGoAnsiblePkg(c *conf.Bootstrap) *GoAnsiblePkg {
	return &GoAnsiblePkg{
		c:                     c,
		ansiblePlaybookBinary: playbook.DefaultAnsiblePlaybookBinary,
	}
}

func (a *GoAnsiblePkg) Write(p []byte) (n int, err error) {
	a.LogChan <- string(p)
	return len(p), nil
}

func (a *GoAnsiblePkg) SetLogChan(logchan chan string) *GoAnsiblePkg {
	a.LogChan = logchan
	return a
}

func (a *GoAnsiblePkg) SetLogPrefix(logPrefix string) *GoAnsiblePkg {
	a.logPrefix = logPrefix
	return a
}

func (a *GoAnsiblePkg) SetAnsiblePlaybookBinary(ansiblePlaybookBinary string) *GoAnsiblePkg {
	a.ansiblePlaybookBinary = ansiblePlaybookBinary
	return a
}

func (a *GoAnsiblePkg) SetCmdRunDir(cmdRunDir string) *GoAnsiblePkg {
	a.cmdRunDir = cmdRunDir
	return a
}

func (a *GoAnsiblePkg) SetInventoryFile(inventoryfile string) *GoAnsiblePkg {
	a.inventoryfile = inventoryfile
	return a
}

func (a *GoAnsiblePkg) SetPlaybooks(playbooks []string) *GoAnsiblePkg {
	a.playbooks = playbooks
	return a
}

func (a *GoAnsiblePkg) SetEnv(key, val string) *GoAnsiblePkg {
	if a.env == nil {
		a.env = make(map[string]string)
	}
	a.env[key] = val
	return a
}

func (a *GoAnsiblePkg) SetEnvMap(env map[string]string) *GoAnsiblePkg {
	a.env = env
	return a
}

func (a *GoAnsiblePkg) ExecPlayBooks(ctx context.Context) error {
	if a.cmdRunDir == "" {
		return fmt.Errorf("cmdRunDir path is required")
	}
	if a.inventoryfile == "" {
		return fmt.Errorf("inventory file is required")
	}
	if len(a.playbooks) == 0 {
		return fmt.Errorf("playbooks is required")
	}
	if a.LogChan == nil {
		return fmt.Errorf("log channel is required")
	}
	envExecute := []execute.ExecuteOptions{
		execute.WithCmdRunDir(a.cmdRunDir),
		execute.WithWrite(a),
		execute.WithWriteError(a),
		execute.WithTransformers(
			results.Prepend(a.logPrefix),
		),
		execute.WithEnvVar("ANSIBLE_FORCE_COLOR", "true"),
	}
	for k, v := range a.env {
		envExecute = append(envExecute, execute.WithEnvVar(k, v))
	}
	playbook := &playbook.AnsiblePlaybookCmd{
		Binary:    a.ansiblePlaybookBinary,
		Playbooks: a.playbooks,
		Options: &playbook.AnsiblePlaybookOptions{
			Inventory: a.inventoryfile,
		},
		PrivilegeEscalationOptions: &options.AnsiblePrivilegeEscalationOptions{
			Become:       true,
			BecomeMethod: "sudo",
			BecomeUser:   "root",
		},
		Exec: execute.NewDefaultExecute(envExecute...),
	}
	return playbook.Run(ctx)
}

type Server struct {
	ID       string
	Ip       string
	Username string
	Role     string
}

// param servers: list of servers to generate inventory file
// result: inventory file content
func GenerateInventoryFile(servers []Server) string {
	// node1 ansible_host=95.54.0.12 ip=10.3.0.1 ansible_user=username etcd_member_name=etcd1
	inventory := `
# ## Configure 'ip' variable to bind kubernetes services on a
# ## different ip than the default iface
# ## We should set etcd_member_name for etcd cluster. The node that is not a etcd member do not need to set the value, or can set the empty string value.
[all]
`
	etcdNum := 1
	for _, server := range servers {
		etcdName := ""
		if server.Role == "master" {
			etcdName = fmt.Sprintf("etcd_member_name=etcd%d", etcdNum)
			etcdNum++
		}
		inventory += fmt.Sprintf("%s ansible_host=%s ip=%s ansible_user=%s %s\n",
			server.ID, server.Ip, server.Ip, server.Username, etcdName)
	}

	inventory += `
# ## configure a bastion host if your nodes are not directly reachable
# [bastion]
# bastion ansible_host=x.x.x.x ansible_user=some_user
`
	inventory += `
# ## configure masters
[kube_control_plane]
`
	for _, server := range servers {
		if server.Role == "master" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
# ## configure etcd
[etcd]
`
	for _, server := range servers {
		if server.Role == "master" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
# ## configure nodes
[kube_node]
`
	for _, server := range servers {
		if server.Role == "worker" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
[calico_rr]
`
	inventory += `
# ## configure k8s cluster using kubeadm
[k8s_cluster:children]
kube_control_plane
kube_node
calico_rr
`
	return inventory
}

type ServerInit struct {
	Tasks []ServerInitTask `yaml:"tasks"`
}

type ServerInitTask struct {
	Name     string `yaml:"name,omitempty"`
	Shell    string `yaml:"shell,omitempty"`
	Register string `yaml:"register,omitempty"`
	When     string `yaml:"when,omitempty"`
	IgnErr   string `yaml:"ignore_errors,omitempty"`
	Command  string `yaml:"command,omitempty"`
}

func (a *GoAnsiblePkg) GenerateServerInitPlaybook() (string, error) {
	serverInitPlaybookContent := `---
`
	serverInit := ServerInit{}
	yamlStr, err := yaml.Marshal(a.c.Serverinit)
	if err != nil {
		return "", err
	}
	err = yaml.Unmarshal(yamlStr, &serverInit)
	if err != nil {
		return "", err
	}
	data := []map[string]any{
		{
			"hosts": "all",
			"tasks": serverInit.Tasks,
		},
	}
	yamlStr, err = yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	serverInitPlaybookContent = serverInitPlaybookContent + string(yamlStr)
	return serverInitPlaybookContent, nil
}

type AnsibleCfg struct {
	SSHConnection map[string]any `json:"ssh_connection"`
	Defaults      map[string]any `json:"defaults"`
	Inventory     map[string]any `json:"inventory"`
}

func (a *GoAnsiblePkg) GenerateAnsibleCfg() (string, error) {
	fString := func(k string, v any) string {
		val := cast.ToString(v)
		if val == "" {
			return ""
		}
		val2 := strings.ToUpper(val[:1]) + val[1:]
		if val2 == "False" || val2 == "True" {
			return fmt.Sprintf("\n%s=%s", k, val2)
		}
		return fmt.Sprintf("\n%s = %s", k, cast.ToString(v))
	}

	ansibleCfg := AnsibleCfg{}
	ansibleCfgJson, err := json.Marshal(a.c.Ansible)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(ansibleCfgJson, &ansibleCfg)
	if err != nil {
		return "", err
	}
	ansibleCfgContent := `
[ssh_connection]`

	for k, v := range ansibleCfg.SSHConnection {
		ansibleCfgContent += fString(k, v)
	}
	ansibleCfgContent += `

[defaults]`
	for k, v := range ansibleCfg.Defaults {
		ansibleCfgContent += fString(k, v)
	}

	ansibleCfgContent += `

[inventory]`
	for k, v := range ansibleCfg.Inventory {
		ansibleCfgContent += fString(k, v)
	}
	return ansibleCfgContent, nil
}
