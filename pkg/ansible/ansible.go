package ansible

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/execute/result/transformer"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

type GoAnsiblePkg struct {
	c                     *conf.Bootstrap
	logPrefix             string
	ansiblePlaybookBinary string
	LogChan               chan string
	cmdRunDir             string
	inventory             string
	playbooks             []string
	env                   map[string]string
	servers               []Server
}

type Server struct {
	ID       string
	Ip       string
	Username string
	Role     string
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

func (a *GoAnsiblePkg) SetInventory(inventory string) *GoAnsiblePkg {
	a.inventory = inventory
	return a
}

func (a *GoAnsiblePkg) SetPlaybooks(playbooks ...string) *GoAnsiblePkg {
	a.playbooks = playbooks
	return a
}

func (a *GoAnsiblePkg) SetEnv(key, val string) *GoAnsiblePkg {
	if a.env == nil {
		a.env = make(map[string]string)
		a.env["ANSIBLE_FORCE_COLOR"] = "true"
	}
	a.env[key] = val
	return a
}

func (a *GoAnsiblePkg) SetServers(servers ...Server) *GoAnsiblePkg {
	a.servers = servers
	return a
}

func (a *GoAnsiblePkg) SetEnvMap(env map[string]string) *GoAnsiblePkg {
	for k, v := range env {
		a.SetEnv(k, v)
	}
	return a
}

func (a *GoAnsiblePkg) ExecPlayBooks(ctx context.Context) error {
	if a.cmdRunDir == "" {
		return errors.New("cmdRunDir is required")
	}
	if a.inventory == "" {
		return errors.New("inventory is required")
	}
	if len(a.playbooks) == 0 {
		return errors.New("playbooks is required")
	}
	if a.LogChan == nil {
		return errors.New("log channel is required")
	}
	if len(a.servers) == 0 {
		return errors.New("servers is required")
	}
	err := a.generateAnsibleCfg()
	if err != nil {
		return err
	}
	err = a.generateInventoryFile()
	if err != nil {
		return err
	}
	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		Inventory:    a.inventory,
		Become:       true,
		BecomeMethod: "sudo",
		BecomeUser:   "root",
	}
	playbookCmd := playbook.NewAnsiblePlaybookCmd(
		playbook.WithPlaybooks(a.playbooks...),
		playbook.WithPlaybookOptions(ansiblePlaybookOptions),
		playbook.WithBinary(a.ansiblePlaybookBinary),
		playbook.WithPlaybookOptions(ansiblePlaybookOptions),
	)

	exec := execute.NewDefaultExecute(
		execute.WithCmd(playbookCmd),
		execute.WithCmdRunDir(a.cmdRunDir),
		execute.WithErrorEnrich(playbook.NewAnsiblePlaybookErrorEnrich()),
		execute.WithWrite(a),
		execute.WithWriteError(a),
		execute.WithEnvVars(a.env),
		execute.WithTransformers(
			transformer.Prepend(a.logPrefix),
		),
	)
	return exec.Execute(ctx)
}

func (a *GoAnsiblePkg) generateAnsibleCfg() error {
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
	type AnsibleCfg struct {
		SSHConnection map[string]any `json:"ssh_connection"`
		Defaults      map[string]any `json:"defaults"`
		Inventory     map[string]any `json:"inventory"`
	}
	ansibleCfg := AnsibleCfg{}
	ansibleCfgJson, err := json.Marshal(a.c.Ansible)
	if err != nil {
		return err
	}
	err = json.Unmarshal(ansibleCfgJson, &ansibleCfg)
	if err != nil {
		return err
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
	// write ansible.cfg
	return os.WriteFile(strings.Join([]string{a.cmdRunDir, "ansible.cfg"}, "/"), []byte(ansibleCfgContent), 0644)
}

func (a *GoAnsiblePkg) generateInventoryFile() error {
	// node1 ansible_host=95.54.0.12 ip=10.3.0.1 ansible_user=username etcd_member_name=etcd1
	inventory := `
# ## Configure 'ip' variable to bind kubernetes services on a
# ## different ip than the default iface
# ## We should set etcd_member_name for etcd cluster. The node that is not a etcd member do not need to set the value, or can set the empty string value.
[all]
`
	etcdNum := 1
	for _, server := range a.servers {
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
	for _, server := range a.servers {
		if server.Role == "master" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
# ## configure etcd
[etcd]
`
	for _, server := range a.servers {
		if server.Role == "master" {
			inventory += fmt.Sprintf("%s\n", server.ID)
		}
	}

	inventory += `
# ## configure nodes
[kube_node]
`
	for _, server := range a.servers {
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

	// write inventory.ini
	a.inventory = strings.Join([]string{a.cmdRunDir, "inventory.ini"}, "/")
	return os.WriteFile(a.inventory, []byte(inventory), 0644)
}

// Playbook represents an Ansible Playbook
type Playbook struct {
	Name  string `yaml:"name"`
	Hosts string `yaml:"hosts"`
	Tasks []Task `yaml:"tasks"`
}

// Task represents a task in the Ansible Playbook
type Task struct {
	Name          string             `yaml:"name"`
	Community     *Community         `yaml:"community,omitempty"`
	GetUrl        *GetUrl            `yaml:"get_url,omitempty"`
	User          *UserTask          `yaml:"user,omitempty"`
	Systemd       *SystemdTask       `yaml:"systemd,omitempty"`
	AptRepository *AptRepositoryTask `yaml:"apt_repository,omitempty"`
	AptKey        *AptKeyTask        `yaml:"apt_key,omitempty"`
	Apt           *AptTask           `yaml:"apt,omitempty"`
	Yum           *YumTask           `yaml:"yum,omitempty"`
	Synchronize   *Synchronize       `yaml:"synchronize,omitempty"`
	Copy          *Copy              `yaml:"copy,omitempty"`
	When          string             `yaml:"when,omitempty"`
	Shell         string             `yaml:"shell,omitempty"`
	Register      string             `yaml:"register,omitempty"`
	IgnoreErrors  string             `yaml:"ignore_errors,omitempty"`
}

// AptTask represents an apt task
type AptTask struct {
	Name        []string `yaml:"name,omitempty"`
	State       string   `yaml:"state,omitempty"`
	UpdateCache string   `yaml:"update_cache,omitempty"`
}

// YumTask represents a yum task
type YumTask struct {
	Name  string `yaml:"name,omitempty"`
	State string `yaml:"state,omitempty"`
}

// Synchronize represents a synchronize task
type Synchronize struct {
	Src    string `yaml:"src,omitempty"`
	Dest   string `yaml:"dest,omitempty"`
	Delete bool   `yaml:"delete,omitempty"`
}

type Copy struct {
	Src  string `yaml:"src,omitempty"`
	Dest string `yaml:"dest,omitempty"`
}

type AptKeyTask struct {
	Keyserver string `yaml:"keyserver,omitempty"`
	Id        string `yaml:"id,omitempty"`
	Url       string `yaml:"url,omitempty"`
	State     string `yaml:"state,omitempty"`
}

type AptRepositoryTask struct {
	Repo        string `yaml:"repo,omitempty"`
	State       string `yaml:"state,omitempty"`
	UpdateCache string `yaml:"update_cache,omitempty"`
}

type SystemdTask struct {
	Name         string `yaml:"name,omitempty"`
	State        string `yaml:"state,omitempty"`
	Enabled      string `yaml:"enabled,omitempty"`
	DaemonReload string `yaml:"daemon_reload,omitempty"`
}

type UserTask struct {
	Name     string `yaml:"name,omitempty"`
	Groups   string `yaml:"groups,omitempty"`
	Append   string `yaml:"append,omitempty"`
	Shell    string `yaml:"shell,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type GetUrl struct {
	Url  string `yaml:"url,omitempty"`
	Dest string `yaml:"dest,omitempty"`
	Mode string `yaml:"mode,omitempty"`
}

type Community struct {
	Docker Docker `yaml:"docker,omitempty"`
}

type Docker struct {
	DockerCompose DockerCompose `yaml:"docker_compose,omitempty"`
}

type DockerCompose struct {
	ProjectSrc string `yaml:"project_src,omitempty"`
	Definition string `yaml:"definition,omitempty"`
	State      string `yaml:"state,omitempty"`
}

func (p *Playbook) AddTask(task Task) {
	p.Tasks = append(p.Tasks, task)
}

func (p *Playbook) AddSynchronize(name, src, dest string) {
	p.AddTask(Task{
		Name: name,
		Synchronize: &Synchronize{
			Src:    src,
			Dest:   dest,
			Delete: true,
		},
	})
}

func SavePlaybook(dir string, playbook *Playbook) (path string, err error) {
	playbooks := []Playbook{*playbook}
	playbookData, err := yaml.Marshal(playbooks)
	if err != nil {
		return "", err
	}
	fileContent := fmt.Sprintf("---\n%s", string(playbookData))
	path = strings.Join([]string{dir, playbook.Name + ".yml"}, "/")
	err = os.WriteFile(path, []byte(fileContent), 0644)
	if err != nil {
		return "", err
	}
	return path, nil
}

func GetServerInitPlaybook() *Playbook {
	return &Playbook{
		Name:  "serverinit",
		Hosts: "all",
		Tasks: []Task{
			{
				Name:  "check 'root' user home directory",
				Shell: "echo 'Hi ocean' | tee -a /root/ocean.log",
			},
			{
				Name:     "cheack IPv4",
				Shell:    "cat /proc/sys/net/ipv4/ip_forward",
				Register: "ipv4_status",
			},
			{
				Name:  "IPv4",
				When:  "ipv4_status.stdout == \"0\"",
				Shell: "echo 'net.ipv4.ip_forward=1' | tee -a /etc/sysctl.conf",
			},
			{
				Name:     "cheack IPv6",
				Shell:    "cat /proc/sys/net/ipv6/conf/all/forwarding",
				Register: "ipv6_status",
			},
			{
				Name:  "IPv6",
				When:  "ipv6_status.stdout == \"0\"",
				Shell: "echo 'net.ipv6.conf.all.forwarding=1' | tee -a /etc/sysctl.conf",
			},
			{
				Name:         "Reload sysctl",
				Shell:        "sysctl -p",
				IgnoreErrors: "yes",
			},
			{
				Name:         "Check firewall status",
				Shell:        "systemctl is-active firewalld",
				Register:     "firewall_status",
				IgnoreErrors: "yes",
			},
			{
				Name:  "Close firewall",
				When:  "firewall_status.stdout == \"active\"",
				Shell: "systemctl stop firewalld && systemctl disable firewalld",
			},
		},
	}
}

func GetMigratePlaybook() *Playbook {
	playbook := &Playbook{
		Name:  "migrate",
		Hosts: "all",
		Tasks: []Task{
			{
				Name: "Update apt cache",
				Apt: &AptTask{
					UpdateCache: "yes",
				},
			},
			{
				Name: "Install prerequisite packages",
				Apt: &AptTask{
					Name: []string{
						"apt-transport-https",
						"ca-certificates",
						"curl",
						"software-properties-common",
						"rsync",
					},
					State: "present",
				},
			},
			{
				Name: "Add Docker's official GPG key",
				AptKey: &AptKeyTask{
					Url:   "https://download.docker.com/linux/ubuntu/gpg",
					State: "present",
				},
			},
			{
				Name: "Add Docker APT repository",
				AptRepository: &AptRepositoryTask{
					Repo:        "deb [arch=amd64] https://download.docker.com/linux/ubuntu {{ ansible_distribution_release }} stable",
					State:       "present",
					UpdateCache: "yes",
				},
			},
			{
				Name: "Install Docker",
				Apt: &AptTask{
					Name:  []string{"docker-ce"},
					State: "present",
				},
			},
			{
				Name: "Ensure Docker service is started and enabled",
				Systemd: &SystemdTask{
					Name:    "docker",
					State:   "started",
					Enabled: "yes",
				},
			},
			{
				Name: "Add current user to Docker group",
				User: &UserTask{
					Name:   "\"{{ ansible_user }}\"",
					Groups: "docker",
					Append: "yes",
				},
			},
			{
				Name: "Ensure Docker is installed",
				Apt: &AptTask{
					Name:        []string{"docker.io"},
					State:       "present",
					UpdateCache: "yes",
				},
			},
			{
				Name: "Ensure Docker Compose is installed",
				GetUrl: &GetUrl{
					Url:  "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-{{ ansible_system }}-{{ ansible_architecture }}",
					Dest: "/usr/local/bin/docker-compose",
					Mode: "0755",
				},
			},
			{
				Name: "Ensure Docker Compose file is present",
				Copy: &Copy{
					Src:  "./docker-compose.yml",
					Dest: "/opt/docker-compose.yml",
				},
			},
		},
	}
	dockerCompose := &DockerCompose{
		ProjectSrc: "/opt",
		Definition: "\"{{ lookup('file', '/opt/docker-compose.yml') }}\"",
		State:      "present",
	}
	docker := &Docker{
		DockerCompose: *dockerCompose,
	}
	community := &Community{
		Docker: *docker,
	}
	task := Task{
		Name:      "Change directory and run Docker Compose",
		Community: community,
	}
	playbook.AddTask(task)
	return playbook
}
