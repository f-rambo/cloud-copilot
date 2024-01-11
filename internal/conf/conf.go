package conf

import (
	"os"
	"strconv"
)

type Bootstrap struct {
	Server     Server     `json:"server,omitempty"`
	Data       Data       `json:"data,omitempty"`
	Log        Log        `json:"log,omitempty"`
	Auth       Auth       `json:"auth,omitempty"`
	Kubernetes Kubernetes `json:"kubernetes,omitempty"`
	Resource   Resource   `json:"resource,omitempty"`
}

type Resource struct {
	AppPath  string `json:"app_path,omitempty"`
	IconPath string `json:"icon_path,omitempty"`
	RepoPath string `json:"repo_path,omitempty"`
}

func (r Resource) GetAppPath() string {
	appPath := os.Getenv("APP_PATH")
	if appPath == "" {
		appPath = r.AppPath
	}
	return appPath
}

func (r Resource) GetIconPath() string {
	iconPath := os.Getenv("ICON_PATH")
	if iconPath == "" {
		iconPath = r.IconPath
	}
	return iconPath
}

func (r Resource) GetRepoPath() string {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		repoPath = r.RepoPath
	}
	return repoPath
}

type Server struct {
	HTTP   HTTP   `json:"http,omitempty"`
	GRPC   GRPC   `json:"grpc,omitempty"`
	STATIC STATIC `json:"static,omitempty"`
}

type HTTP struct {
	Network string `json:"network,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

func (h HTTP) GetNetwork() string {
	newWork := os.Getenv("HTTP_NETWORK")
	if newWork != "" {
		return newWork
	}
	return h.Network
}

func (h HTTP) GetAddr() string {
	addr := os.Getenv("HTTP_ADDR")
	if addr != "" {
		return addr
	}
	return h.Addr
}

type GRPC struct {
	Network string `json:"network,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

func (g GRPC) GetNetwork() string {
	netWork := os.Getenv("GRPC_NETWORK")
	if netWork != "" {
		return netWork
	}
	return g.Network
}

func (g GRPC) GetAddr() string {
	addr := os.Getenv("GRPC_ADDR")
	if addr != "" {
		return addr
	}
	return g.Addr
}

type STATIC struct {
	Network string `json:"network,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

func (s STATIC) GetNetwork() string {
	netWork := os.Getenv("STATIC_NETWORK")
	if netWork != "" {
		return netWork
	}
	return s.Network
}

func (s STATIC) GetAddr() string {
	addr := os.Getenv("STATIC_ADDR")
	if addr != "" {
		return addr
	}
	return s.Addr
}

type Data struct {
	Database Database `json:"database,omitempty"`
}

type Database struct {
	Driver     string `json:"driver,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int32  `json:"port,omitempty"`
	Database   string `json:"database,omitempty"`
	DBFilePath string `json:"dbfilepath,omitempty"`
}

func (d Database) GetDriver() string {
	driver := os.Getenv("DATABASE_DRIVER")
	if driver != "" {
		return driver
	}
	return d.Driver
}

func (d Database) GetUsername() string {
	username := os.Getenv("DATABASE_USERNAME")
	if username != "" {
		return username
	}
	return d.Username
}

func (d Database) GetPassword() string {
	password := os.Getenv("DATABASE_PASSWORD")
	if password != "" {
		return password
	}
	return d.Password
}

func (d Database) GetHost() string {
	host := os.Getenv("DATABASE_HOST")
	if host != "" {
		return host
	}
	return d.Host
}

func (d Database) GetPort() int32 {
	port := os.Getenv("DATABASE_PORT")
	if port != "" {
		portInt, _ := strconv.Atoi(port)
		return int32(portInt)
	}
	return d.Port
}

func (d Database) GetDatabase() string {
	database := os.Getenv("DATABASE_DATABASE")
	if database != "" {
		return database
	}
	return d.Database
}

func (d Database) GetDBFilePath() string {
	dbFilePath := os.Getenv("DATABASE_DBFILEPATH")
	if dbFilePath != "" {
		return dbFilePath
	}
	return d.DBFilePath
}

type Kubernetes struct {
	MasterUrl  string `json:"master_url,omitempty"`
	KubeConfig string `json:"kube_config,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
}

func (k Kubernetes) GetMasterUrl() string {
	masterUrl := os.Getenv("KUBERNETES_MASTER_URL")
	if masterUrl != "" {
		return masterUrl
	}
	return k.MasterUrl
}

func (k Kubernetes) GetKubeConfig() string {
	kubeConfig := os.Getenv("KUBERNETES_KUBE_CONFIG")
	if kubeConfig != "" {
		return kubeConfig
	}
	return k.KubeConfig
}

type Log struct {
	Path       string `json:"path,omitempty"`
	Filename   string `json:"filename,omitempty"`
	MaxSize    int32  `json:"max_size,omitempty"`
	MaxAge     int32  `json:"max_age,omitempty"`
	MaxBackups int32  `json:"max_backups,omitempty"`
	Compress   bool   `json:"compress,omitempty"`
	LocalTime  bool   `json:"local_time,omitempty"`
}

type Auth struct {
	Exp      int32  `json:"exp,omitempty"`
	Key      string `json:"key,omitempty"`
	Email    string `json:"email,omitempty"`
	PassWord string `json:"password,omitempty"`
}

func (a Auth) GetExp() int32 {
	exp := os.Getenv("AUTH_EXP")
	if exp != "" {
		expInt, _ := strconv.Atoi(exp)
		return int32(expInt)
	}
	return a.Exp
}

func (a Auth) GetKey() string {
	key := os.Getenv("AUTH_KEY")
	if key != "" {
		return key
	}
	return a.Key
}

func (a Auth) GetEmail() string {
	email := os.Getenv("AUTH_EMAIL")
	if email != "" {
		return email
	}
	return a.Email
}

func (a Auth) GetPassWord() string {
	password := os.Getenv("AUTH_PASSWORD")
	if password != "" {
		return password
	}
	return a.PassWord
}
