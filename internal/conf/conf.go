package conf

import (
	"os"
	"strconv"
)

type Bootstrap struct {
	Server Server `json:"server,omitempty"`
	Data   Data   `json:"data,omitempty"`
	Log    Log    `json:"log,omitempty"`
	Auth   Auth   `json:"auth,omitempty"`
}

type Server struct {
	Debug   bool   `json:"debug,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	HTTP    HTTP   `json:"http,omitempty"`
	GRPC    GRPC   `json:"grpc,omitempty"`
	Env     string `json:"env,omitempty"`
}

type HTTP struct {
	Network string `json:"network,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

type GRPC struct {
	Network string `json:"network,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

type Data struct {
	Driver   string `json:"driver,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
}

type Log struct {
	MaxSize    int32 `json:"max_size,omitempty"`
	MaxAge     int32 `json:"max_age,omitempty"`
	MaxBackups int32 `json:"max_backups,omitempty"`
}

type Auth struct {
	Exp int32  `json:"exp,omitempty"` // hours
	Key string `json:"key,omitempty"` // secret key
}

type Env string

const (
	EnvLocal       Env = "local"
	EnvBostionHost Env = "bostionhost"
	EnvCluster     Env = "cluster"
)

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

func (e Env) String() string {
	return string(e)
}

func (s Server) GetEnv() Env {
	env := os.Getenv("ENV")
	if env != "" {
		return Env(env)
	}
	return Env(s.Env)
}

func (d Data) GetDriver() string {
	driver := os.Getenv("DATABASE_DRIVER")
	if driver != "" {
		return driver
	}
	return d.Driver
}

func (d Data) GetUsername() string {
	username := os.Getenv("DATABASE_USERNAME")
	if username != "" {
		return username
	}
	return d.Username
}

func (d Data) GetPassword() string {
	password := os.Getenv("DATABASE_PASSWORD")
	if password != "" {
		return password
	}
	return d.Password
}

func (d Data) GetHost() string {
	host := os.Getenv("DATABASE_HOST")
	if host != "" {
		return host
	}
	return d.Host
}

func (d Data) GetPort() int32 {
	port := os.Getenv("DATABASE_PORT")
	if port != "" {
		portInt, _ := strconv.Atoi(port)
		return int32(portInt)
	}
	return d.Port
}

func (d Data) GetDatabase() string {
	database := os.Getenv("DATABASE_DATABASE")
	if database != "" {
		return database
	}
	return d.Database
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
