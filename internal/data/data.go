package data

import (
	"context"
	"fmt"
	"time"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo, NewProjectRepo)

type Data struct {
	databaseConf *conf.Data
	etcdConf     *conf.ETCD
	log          *log.Helper
	db           *gorm.DB
	etcd         *clientv3.Client
	kvStore      *utils.KVStore
}

func NewData(c *conf.Bootstrap, logger log.Logger) (*Data, func(), error) {
	var err error
	cdata := c.Data
	etcd := c.ETCD
	data := &Data{
		databaseConf: &cdata,
		etcdConf:     &etcd,
		log:          log.NewHelper(logger),
	}

	data.db, err = newDB(cdata)
	if err != nil {
		return nil, nil, err
	}
	data.etcd, err = newEtcd(etcd)
	if err != nil {
		data.kvStore = utils.NewKVStore()
	}
	cleanup := func() {
		if data.etcd != nil {
			err = data.etcd.Close()
			if err != nil {
				log.Error("closing the etcd resources", err)
			}
		}
		if data.kvStore != nil {
			data.kvStore.Close()
		}
		log.Info("closing the data resources")
	}
	return data, cleanup, nil
}

type QueueKey string

func (k QueueKey) String() string {
	return string(k)
}

func (d *Data) Put(ctx context.Context, key, val string) error {
	if d.etcd == nil {
		return d.kvStore.Put(ctx, key, val)
	}
	_, err := d.etcd.Put(ctx, key, val)
	if err != nil {
		return err
	}
	return nil
}

func (d *Data) Get(ctx context.Context, key string) (string, error) {
	if d.etcd == nil {
		return d.kvStore.Get(ctx, key)
	}
	resp, err := d.etcd.Get(ctx, key)
	if err != nil {
		return "", err
	}
	for _, v := range resp.Kvs {
		return string(v.Value), nil
	}
	return "", nil
}

func (d *Data) Watch(ctx context.Context, key string) (string, error) {
	if d.etcd == nil {
		watchChan, err := d.kvStore.Watch(ctx, key)
		if err != nil {
			return "", err
		}
		lenth := len(watchChan)
		var i int = 0
		for {
			i++
			select {
			case val, exists := <-watchChan:
				if !exists {
					return "", nil
				}
				if i >= lenth {
					return val, nil
				}
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}
	watchChan := d.etcd.Watch(ctx, key)
	if watchChan == nil {
		return "", errors.New("watch chan is nil")
	}
	for {
		select {
		case wresp, ok := <-watchChan:
			if !ok {
				return "", nil
			}
			for _, ev := range wresp.Events {
				return string(ev.Kv.Value), nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func (d *Data) Delete(ctx context.Context, key string) error {
	if d.etcd == nil {
		return d.kvStore.Delete(ctx, key)
	}
	_, err := d.etcd.Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

func newEtcd(c conf.ETCD) (*clientv3.Client, error) {
	endpoints := make([]string, 0)
	for _, endpoint := range c.GetETCDEndpoints() {
		if endpoint == "" {
			continue
		}
		endpoints = append(endpoints, endpoint)
	}
	if len(endpoints) == 0 {
		return nil, errors.New("etcd endpoints is empty")
	}
	cli, err := clientv3.New(clientv3.Config{
		Username:    c.GetUsername(),
		Password:    c.GetPassword(),
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func newDB(c conf.Data) (*gorm.DB, error) {
	var client *gorm.DB
	var err error
	if c.GetDriver() == "mysql" {
		dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
			c.GetUsername(), c.GetPassword(), c.GetHost(), c.GetPort(), c.GetDatabase())
		client, err = gorm.Open(mysql.Open(dns), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		client = client.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8")
	}
	if c.GetDriver() == "postgres" {
		dns := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
			c.GetHost(), c.GetUsername(), c.GetPassword(), c.GetDatabase(), c.GetPort())
		client, err = gorm.Open(postgres.Open(dns), &gorm.Config{})
		if err != nil {
			return nil, err
		}
	}
	if (c.GetDriver() != "mysql" && c.GetDriver() != "postgres") || c.GetDriver() == "sqlite" {
		dbFilePath, err := utils.GetPackageStorePathByNames("database", "ocean.db")
		if dbFilePath != "" && !utils.IsFileExist(dbFilePath) {
			path, filename := utils.GetFilePathAndName(dbFilePath)
			file, err := utils.NewFile(path, filename, true)
			if err != nil {
				return nil, err
			}
			file.Close()
		}
		if dbFilePath == "" {
			dbFilePath = "file::memory:?cache=shared"
		}
		client, err = gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
		if err != nil {
			return nil, err
		}
	}
	// AutoMigrate
	err = client.AutoMigrate(
		&biz.Cluster{},
		&biz.NodeGroup{},
		&biz.BostionHost{},
		&biz.Node{},
		&biz.Project{},
		&biz.App{},
		&biz.AppType{},
		&biz.AppVersion{},
		&biz.DeployApp{},
		&biz.Service{},
		&biz.CI{},
		&biz.User{},
		&biz.AppHelmRepo{},
	)
	if err != nil {
		return client, err
	}
	return client, nil
}
