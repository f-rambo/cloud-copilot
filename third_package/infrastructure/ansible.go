package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/execute/result"
	"github.com/apenella/go-ansible/v2/pkg/execute/result/transformer"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

const (
	ansibleCliPackageName = "ansible"
	ansibleCliUrl         = "https://github.com/ansible/ansible/archive/refs/tags/v2.17.3.tar.gz"
)

type GoAnsiblePkg struct {
	ansiblePath   string
	ansibleConfig string
	logPrefix     string
	cmdRunDir     string
	inventory     string
	playbooks     []string
	env           map[string]string
	servers       []Server
	mateData      map[string]string
	w             io.Writer
	output        string
}

type Server struct {
	ID       string
	Ip       string
	Username string
	Role     string
}

func NewGoAnsiblePkg(w io.Writer) (*GoAnsiblePkg, error) {
	a := &GoAnsiblePkg{
		ansibleConfig: `
ansible:
  ssh_connection:
    pipelining: true
    ansible_ssh_args: '-o ControlMaster=auto -o ControlPersist=30m -o ConnectionAttempts=100 -o UserKnownHostsFile=/dev/null'
  defaults:
    timeout: 300
    ask_pass: false
    ask_become_pass: false
    force_valid_group_names: ignore
    host_key_checking: false
    gathering: smart
    fact_caching: jsonfile
    fact_caching_connection: /tmp
    fact_caching_timeout: 86400
    stdout_callback: default
    display_skipped_hosts: no
    library: './library'
    callbacks_enabled: 'profile_tasks'
    roles_path: 'roles:$VIRTUAL_ENV/usr/local/share/kubespray/roles:$VIRTUAL_ENV/usr/local/share/ansible/roles:/usr/share/kubespray/roles'
    deprecation_warnings: false
    inventory_ignore_extensions: '~, .orig, .bak, .ini, .cfg, .retry, .pyc, .pyo, .creds, .gpg'
  inventory:
    ignore_patterns: 'artifacts, credentials'`,
		w:      w,
		output: "",
	}
	err := a.autoInstallPython()
	if err != nil {
		return nil, err
	}
	err = a.autoInstallAnsibleCli()
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *GoAnsiblePkg) autoInstallPython() (err error) {
	// 检查文件是否存储
	pythonPath, err := utils.GetPackageStorePathByNames("python")
	if err != nil {
		return err
	}
	if utils.IsFileExist(pythonPath) {
		return nil
	}
	// 下载python
	err = utils.DownloadFile("https://www.python.org/ftp/python/3.9.7/Python-3.9.7.tgz", pythonPath)
	if err != nil {
		return err
	}
	// 解压python
	err = utils.Decompress(pythonPath, pythonPath)
	if err != nil {
		return err
	}
	return nil
}

func (a *GoAnsiblePkg) autoInstallAnsibleCli() (err error) {
	// 检查文件是否存储
	a.ansiblePath, err = utils.GetPackageStorePathByNames(ansibleCliPackageName)
	if err != nil {
		return err
	}
	if utils.IsFileExist(a.ansiblePath) {
		return nil
	}
	// 下载ansible
	err = utils.DownloadFile(ansibleCliUrl, a.ansiblePath)
	if err != nil {
		return err
	}
	// 解压ansible
	err = utils.Decompress(a.ansiblePath, a.ansiblePath)
	if err != nil {
		return err
	}
	return nil
}

func (a *GoAnsiblePkg) Print(ctx context.Context, reader io.Reader, writer io.Writer, options ...result.OptionsFunc) error {
	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if n > 0 {
			_, err = writer.Write(buf[:n])
			if err != nil {
				return err
			}
			a.output += string(buf[:n])
		}
	}
	return nil
}

func (a *GoAnsiblePkg) Enrich(err error) error {
	if err == nil {
		return nil
	}
	return errors.Wrap(err, "ansible error")
}

func (a *GoAnsiblePkg) SetLogPrefix(logPrefix string) *GoAnsiblePkg {
	a.logPrefix = logPrefix
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
	if env == nil {
		a.env = make(map[string]string)
		return a
	}
	for k, v := range env {
		a.SetEnv(k, v)
	}
	return a
}

func (a *GoAnsiblePkg) SetMatedata(key, val string) *GoAnsiblePkg {
	if a.mateData == nil {
		a.mateData = make(map[string]string)
	}
	a.mateData[key] = val
	return a
}

func (a *GoAnsiblePkg) SetMatedataMap(mateData map[string]string) *GoAnsiblePkg {
	for k, v := range mateData {
		a.SetMatedata(k, v)
	}
	return a
}

func (a *GoAnsiblePkg) ExecPlayBooks(ctx context.Context) (string, error) {
	if a.cmdRunDir == "" {
		return "", errors.New("cmdRunDir is required")
	}
	if len(a.playbooks) == 0 {
		return "", errors.New("playbooks is required")
	}
	if len(a.servers) == 0 {
		return "", errors.New("servers is required")
	}
	err := a.generateAnsibleCfg()
	if err != nil {
		return "", err
	}
	err = a.generateInventoryFile()
	if err != nil {
		return "", err
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
		playbook.WithBinary(playbook.DefaultAnsiblePlaybookBinary),
	)

	exec := execute.NewDefaultExecute(
		execute.WithCmd(playbookCmd),
		execute.WithCmdRunDir(a.cmdRunDir),
		execute.WithErrorEnrich(a),
		execute.WithWrite(a.w),
		execute.WithOutput(a),
		execute.WithWriteError(a.w),
		execute.WithEnvVars(a.env),
		execute.WithTransformers(
			transformer.Prepend(a.logPrefix),
		),
	)
	err = exec.Execute(ctx)
	if err != nil {
		return "", err
	}
	return a.output, nil
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
	ansibleCfgJson, err := json.Marshal(a.ansibleConfig)
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
	file, err := utils.NewFile(a.cmdRunDir, "ansible.cfg", true)
	if err != nil {
		return err
	}
	defer file.Close()
	err = file.ClearFileContent()
	if err != nil {
		return err
	}
	return file.Write([]byte(ansibleCfgContent))
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
	file, err := utils.NewFile(a.cmdRunDir, "inventory.ini", true)
	if err != nil {
		return err
	}
	defer file.Close()
	err = file.ClearFileContent()
	if err != nil {
		return err
	}
	err = file.Write([]byte(inventory))
	if err != nil {
		return err
	}
	a.inventory = file.GetFileName()
	return nil
}

// Playbook represents an Ansible Playbook
type Playbook struct {
	Name        string `yaml:"name"`
	Hosts       string `yaml:"hosts"`
	GatherFacts string `yaml:"gather_facts,omitempty"`
	Tasks       []Task `yaml:"tasks"`
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
	Command       string             `yaml:"command,omitempty"`
	Debug         *Debug             `yaml:"debug,omitempty"`
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

type Debug struct {
	Msg string `yaml:"msg,omitempty"`
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

func savePlaybook(dir string, playbook *Playbook) (path string, err error) {
	playbooks := []Playbook{*playbook}
	playbookData, err := yaml.Marshal(playbooks)
	if err != nil {
		return "", err
	}
	fileContent := fmt.Sprintf("---\n%s", string(playbookData))
	file, err := utils.NewFile(dir, playbook.Name+".yml", true)
	if err != nil {
		return "", err
	}
	defer file.Close()
	err = file.ClearFileContent()
	if err != nil {
		return "", err
	}
	err = file.Write([]byte(fileContent))
	if err != nil {
		return "", err
	}
	return file.GetFileName(), nil
}

func getServerInitPlaybook() *Playbook {
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

func getMigratePlaybook() *Playbook {
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

type OutputKey string

func (o OutputKey) String() string {
	return string(o)
}

const (
	StartOutputKey OutputKey = "oceanstart"
	EndOutputKey   OutputKey = "oceanend"
)

func getSystemInformation() *Playbook {
	return &Playbook{
		Name:        "systeminfo",
		Hosts:       "all",
		GatherFacts: "no",
		Tasks: []Task{
			{
				Name: "Install pciutils",
				Apt: &AptTask{
					Name:  []string{"pciutils"},
					State: "present",
				},
			},
			{
				Name:         "Get GPU number",
				Shell:        "lspci | grep -i 'NVIDIA' | wc -l", // 英伟达提供了kuberentes插件
				Register:     "gpu_number",
				IgnoreErrors: "yes",
			},
			{
				Name:         "Get GPU specification",
				Shell:        "lspci | grep -i 'NVIDIA' | awk -F 'VGA compatible controller:' '{print $2;exit}'", // 只保留一行数据
				Register:     "gpu_spec",
				IgnoreErrors: "yes",
			},
			{
				Name:     "Get CPU number",
				Shell:    "lscpu | grep '^CPU(s):' | awk '{print $2;exit}'",
				Register: "cpu_number",
			},
			{
				Name:     "Get memory",
				Shell:    `free -m | awk '/^Mem:/ {printf "%.0f\n", $2 / 1024}'`,
				Register: "memory",
			},
			{
				Name:     "Get disk",
				Shell:    `df -h | awk '/^\/dev/ {print $2}' | awk 'function convert(size) {n = substr(size, 1, length(size)-1); unit = substr(size, length(size), 1); if (unit == "G") {return n} else if (unit == "M") {return n / 1024} else if (unit == "K") {return n / (1024*1024)} else if (unit == "T") {return n * 1024} return 0} {sum += convert($1)} END {print int(sum)}'`,
				Register: "disk",
			},
			{
				Name:     "Get network information",
				Shell:    `default_ip=$(ip addr show $(ip route | grep default | awk '{print $5}') | grep 'inet ' | awk '{print $2}' | cut -d/ -f1) && echo $default_ip`,
				Register: "ip",
			},
			{
				Name:     "Get OS information",
				Shell:    `cat /etc/os-release | grep '^PRETTY_NAME=' | cut -d '=' -f 2 | tr -d '"'`,
				Register: "os_info",
			},
			{
				Name:     "Get kernel information",
				Shell:    "uname -a",
				Register: "kernel_info",
			},
			{
				Name:         "Get Container runtime version",
				Shell:        "containerd --version",
				Register:     "container_version",
				IgnoreErrors: "yes",
			},
			{
				Name: "Print system information",
				Debug: &Debug{
					Msg: fmt.Sprintf("%s%s%s",
						StartOutputKey.String(),
						`{"node_id":"{{ inventory_hostname }}","gpu_number": "{{ gpu_number.stdout }}", "gpu_spec": "{{ gpu_spec.stdout }}", "cpu_number": "{{ cpu_number.stdout }}", "memory": "{{ memory.stdout }}", "disk": "{{ disk.stdout }}", "ip": "{{ ip.stdout }}", "os_info": "{{ os_info.stdout }}", "kernel_info": "{{ kernel_info.stdout }}", "container_version": "{{ container_version.stdout }}"}`,
						EndOutputKey.String()),
				},
			},
		},
	}
}
