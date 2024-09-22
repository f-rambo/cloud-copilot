package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo, NewProjectRepo)

type DBDriver string

const (
	DBDriverMySQL    DBDriver = "mysql"
	DBDriverPostgres DBDriver = "postgres"
	DBDriverSQLite   DBDriver = "sqlite"
)

func (d DBDriver) String() string {
	return string(d)
}

const (
	DatabaseName = "ocean.db"
)

type Data struct {
	conf          *conf.Bootstrap
	log           *log.Helper
	dbLoggerLevel gormlogger.LogLevel
	db            *gorm.DB
	etcd          *clientv3.Client
	kvStore       *utils.KVStore
}

func NewData(c *conf.Bootstrap, logger log.Logger) (*Data, func(), error) {
	var err error
	data := &Data{
		conf:          c,
		log:           log.NewHelper(logger),
		dbLoggerLevel: gormlogger.Warn,
	}

	err = data.newDB(data.conf.Data)
	if err != nil {
		return nil, nil, err
	}
	err = data.newEtcd(data.conf.ETCD)
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

func (d *Data) newEtcd(c conf.ETCD) (err error) {
	endpoints := make([]string, 0)
	for _, endpoint := range c.GetETCDEndpoints() {
		if endpoint == "" {
			continue
		}
		endpoints = append(endpoints, endpoint)
	}
	if len(endpoints) == 0 {
		return errors.New("etcd endpoints is empty")
	}
	d.etcd, err = clientv3.New(clientv3.Config{
		Username:    c.GetUsername(),
		Password:    c.GetPassword(),
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *Data) newDB(c conf.Data) (err error) {
	var gormDialector gorm.Dialector
	switch DBDriver(c.GetDriver()) {
	case DBDriverMySQL:
		dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
			c.GetUsername(), c.GetPassword(), c.GetHost(), c.GetPort(), c.GetDatabase())
		gormDialector = mysql.Open(dns)
	case DBDriverPostgres:
		dns := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
			c.GetHost(), c.GetUsername(), c.GetPassword(), c.GetDatabase(), c.GetPort())
		gormDialector = postgres.Open(dns)
	default:
		dbFilePath, err := utils.GetPackageStorePathByNames(DBDriverSQLite.String(), DatabaseName)
		if err != nil {
			return err
		}
		if dbFilePath != "" && !utils.IsFileExist(dbFilePath) {
			dir, _ := filepath.Split(dbFilePath)
			os.MkdirAll(dir, 0755)
			file, err := os.Create(dbFilePath)
			if err != nil {
				return err
			}
			file.Close()
		}
		gormDialector = sqlite.Open(dbFilePath)
	}
	tablePrefix := fmt.Sprintf("%s_", c.GetDatabase())
	d.db, err = gorm.Open(gormDialector, &gorm.Config{
		Logger: d,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix,
			SingularTable: true,
		},
	})
	if err != nil {
		return err
	}
	// AutoMigrate
	err = d.db.AutoMigrate(
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
		return err
	}
	return nil
}

func (d *Data) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	d.dbLoggerLevel = level
	return d
}

func (d *Data) Info(ctx context.Context, msg string, args ...interface{}) {
	if d.dbLoggerLevel >= gormlogger.Info {
		d.log.WithContext(ctx).Infof(msg, args...)
	}
}

func (d *Data) Warn(ctx context.Context, msg string, args ...interface{}) {
	if d.dbLoggerLevel >= gormlogger.Warn {
		d.log.WithContext(ctx).Warnf(msg, args...)
	}
}

func (d *Data) Error(ctx context.Context, msg string, args ...interface{}) {
	if d.dbLoggerLevel >= gormlogger.Error {
		d.log.WithContext(ctx).Errorf(msg, args...)
	}
}

func (d *Data) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if d.dbLoggerLevel >= gormlogger.Info {
		sql, rows := fc()
		d.log.WithContext(ctx).Infof("begin: %s, sql: %s, rows: %d, err: %v", begin.Format("2006-01-02 15:04:05"), sql, rows, err)
	}
}
