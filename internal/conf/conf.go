package conf

import (
	"os"
	"strconv"
)

type Bootstrap struct {
	Server Server `json:"server,omitempty"`
	Data   Data   `json:"data,omitempty"`
	Log    Log    `json:"log,omitempty"`
}

type Server struct {
	HTTP HTTP `json:"http,omitempty"`
	GRPC GRPC `json:"grpc,omitempty"`
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

type Data struct {
	Database   Database   `json:"database,omitempty"`
	Redis      Redis      `json:"redis,omitempty"`
	Semaphore  Semaphore  `json:"semaphore,omitempty"`
	Kubernetes Kubernetes `json:"kubernetes,omitempty"`
}

type Database struct {
	Driver   string `json:"driver,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
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

type Redis struct {
	Host     string `json:"host,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Password string `json:"password,omitempty"`
	Db       int32  `json:"db,omitempty"`
}

func (r Redis) GetHost() string {
	host := os.Getenv("REDIS_HOST")
	if host != "" {
		return host
	}
	return r.Host
}

func (r Redis) GetPort() int32 {
	port := os.Getenv("REDIS_PORT")
	if port != "" {
		portInt, _ := strconv.Atoi(port)
		return int32(portInt)
	}
	return r.Port
}

func (r Redis) GetPassword() string {
	password := os.Getenv("REDIS_PASSWORD")
	if password != "" {
		return password
	}
	return r.Password
}

func (r Redis) GetDb() int32 {
	db := os.Getenv("REDIS_DB")
	if db != "" {
		dbInt, _ := strconv.Atoi(db)
		return int32(dbInt)
	}
	return r.Db
}

type Semaphore struct {
	Admin         string `json:"admin,omitempty"`
	AdminPassword string `json:"admin_password,omitempty"`
	Host          string `json:"host,omitempty"`
	Port          int32  `json:"port,omitempty"`
}

func (s Semaphore) GetAdmin() string {
	admin := os.Getenv("SEMAPHORE_ADMIN")
	if admin != "" {
		return admin
	}
	return s.Admin
}

func (s Semaphore) GetAdminPassword() string {
	adminPassword := os.Getenv("SEMAPHORE_ADMIN_PASSWORD")
	if adminPassword != "" {
		return adminPassword
	}
	return s.AdminPassword
}

func (s Semaphore) GetHost() string {
	host := os.Getenv("SEMAPHORE_HOST")
	if host != "" {
		return host
	}
	return s.Host
}

func (s Semaphore) GetPort() int32 {
	port := os.Getenv("SEMAPHORE_PORT")
	if port != "" {
		portInt, _ := strconv.Atoi(port)
		return int32(portInt)
	}
	return s.Port
}

type Kubernetes struct {
	MasterUrl  string `json:"master_url,omitempty"`
	KubeConfig string `json:"kube_config,omitempty"`
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
