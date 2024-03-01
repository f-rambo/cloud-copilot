package ansible

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
)

type GoAnsiblePkg struct {
	logPrefix             string
	log                   *ExecLog
	logErr                *ExecLogErr
	ansiblePlaybookBinary string
}

func NewGoAnsiblePkg(ansiblePlaybookBinary string, logPrefixs ...string) *GoAnsiblePkg {
	logPrefix := strings.Join(logPrefixs, "")
	if ansiblePlaybookBinary == "" {
		ansiblePlaybookBinary = playbook.DefaultAnsiblePlaybookBinary
	}
	return &GoAnsiblePkg{
		logPrefix:             logPrefix,
		log:                   new(ExecLog),
		logErr:                new(ExecLogErr),
		ansiblePlaybookBinary: ansiblePlaybookBinary,
	}
}

func (a *GoAnsiblePkg) execPlayBooks(ctx context.Context, inventoryfile string, playbooks []string) error {
	playbook := &playbook.AnsiblePlaybookCmd{
		Binary:    a.ansiblePlaybookBinary,
		Playbooks: playbooks,
		Options: &playbook.AnsiblePlaybookOptions{
			Inventory: inventoryfile,
		},
		PrivilegeEscalationOptions: &options.AnsiblePrivilegeEscalationOptions{
			Become:       true,
			BecomeMethod: "sudo",
			BecomeUser:   "root",
		},
		Exec: execute.NewDefaultExecute(
			execute.WithCmdRunDir(filepath.Dir(inventoryfile)),
			execute.WithEnvVar("ANSIBLE_FORCE_COLOR", "true"),
			execute.WithWrite(a.log),
			execute.WithWriteError(a.logErr),
			execute.WithShowDuration(),
			execute.WithTransformers(
				results.Prepend(a.logPrefix),
			),
		),
	}
	return playbook.Run(ctx)
}

type ExecLog struct{}

func (l *ExecLog) Write(p []byte) (n int, err error) {
	// todo log path
	fmt.Println(string(p))
	return len(p), nil
}

type ExecLogErr struct{}

func (l *ExecLogErr) Write(p []byte) (n int, err error) {
	// todo log path
	fmt.Println(string(p))
	return len(p), nil
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
	inventory := `
	# ## Configure 'ip' variable to bind kubernetes services on a
	# ## different ip than the default iface
	# ## We should set etcd_member_name for etcd cluster. The node that is not a etcd member do not need to set the value, or can set the empty string value.
	[all]
	# node1 ansible_host=95.54.0.12 ip=10.3.0.1 ansible_user=username etcd_member_name=etcd1
	`
	for _, server := range servers {
		inventory += fmt.Sprintf("%s ansible_host=%s ansible_user=%s\n", server.ID, server.Ip, server.Username)
	}

	inventory += `


	`

	inventory += `
	# ## configure a bastion host if your nodes are not directly reachable
	# [bastion]
	# bastion ansible_host=x.x.x.x ansible_user=some_user
	`
	inventory += `
	# ## configure masters
	[kube_control_plane]
	# node1
	# node2
	# node3
	`

	for _, server := range servers {
		if server.Role == "master" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
	# ## configure etcd
	[etcd]
	# node1
	# node2
	# node3
	`
	for _, server := range servers {
		if server.Role == "master" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
	# ## configure nodes
	[kube_node]
	# node4
	# node5
	# node6
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

type Kubespray struct {
	packagePath string
}

func NewKubespray(c *conf.Resource) (*Kubespray, error) {
	k := &Kubespray{}
	// 检查文件是否存储
	k.packagePath = c.GetClusterPath() + "kubespray"
	if utils.IsFileExist(k.packagePath) {
		return k, nil
	}
	// 下载kubespray
	fileName := "kubespray.tar.gz"
	err := utils.DownloadFile(c.GetKubesprayUrl(), c.GetClusterPath(), fileName)
	if err != nil {
		return nil, err
	}
	// 解压kubespray
	err = utils.Decompress(c.GetClusterPath()+fileName, c.GetClusterPath())
	if err != nil {
		return nil, err
	}
	version := ""
	re := regexp.MustCompile(`v(\d+\.\d+\.\d+)`)
	match := re.FindStringSubmatch(c.GetKubesprayUrl())
	if len(match) > 1 {
		version = match[1]
	} else {
		return nil, fmt.Errorf("kubespray version not found")
	}
	// 重命名kubespray
	err = utils.RenameFile(c.GetClusterPath()+"kubespray-"+version, k.packagePath)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *Kubespray) GetDefaultClusterConfig(ctx context.Context) (string, error) {
	defaultClusterConfig := k.packagePath + "/inventory/sample/group_vars/all/all.yml"
	fileData, err := utils.ReadFile(defaultClusterConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) GetDefaultClusterAddons(ctx context.Context) (string, error) {
	defaultClusterAddons := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/addons.yml"
	fileData, err := utils.ReadFile(defaultClusterAddons)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}
