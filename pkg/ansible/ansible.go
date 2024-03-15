package ansible

import (
	"context"
	"fmt"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
)

type GoAnsiblePkg struct {
	logPrefix             string
	ansiblePlaybookBinary string
	LogChan               chan string
	cmdRunDir             string
	inventoryfile         string
	playbooks             []string
	env                   map[string]string
}

func NewGoAnsiblePkg() *GoAnsiblePkg {
	return &GoAnsiblePkg{
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
