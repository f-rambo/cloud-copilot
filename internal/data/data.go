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
	db            *gorm.DB
	dbLoggerLevel gormlogger.LogLevel
}

func NewData(c *conf.Bootstrap, logger log.Logger) (*Data, func(), error) {
	var err error
	data := &Data{
		conf:          c,
		log:           log.NewHelper(logger),
		dbLoggerLevel: gormlogger.Warn,
	}
	cleanup := func() {
		log.Info("closing the data resources")
	}
	var gormDialector gorm.Dialector
	if DBDriver(c.Data.GetDriver()) == DBDriverPostgres {
		dns := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
			c.Data.GetHost(), c.Data.GetUsername(), c.Data.GetPassword(), c.Data.GetDatabase(), c.Data.GetPort())
		gormDialector = postgres.Open(dns)
	}
	if DBDriver(c.Data.GetDriver()) == DBDriverSQLite {
		dbFilePath, err := utils.GetPackageStorePathByNames(DBDriverSQLite.String(), DatabaseName)
		if err != nil {
			return data, cleanup, err
		}
		if dbFilePath != "" && !utils.IsFileExist(dbFilePath) {
			dir, _ := filepath.Split(dbFilePath)
			os.MkdirAll(dir, 0755)
			file, err := os.Create(dbFilePath)
			if err != nil {
				return data, cleanup, err
			}
			file.Close()
		}
		gormDialector = sqlite.Open(dbFilePath)
	}
	if gormDialector == nil {
		return data, cleanup, errors.New("db driver is not supported")
	}
	tablePrefix := fmt.Sprintf("%s_", c.Data.GetDatabase())
	data.db, err = gorm.Open(gormDialector, &gorm.Config{
		Logger: data,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix,
			SingularTable: true,
		},
	})
	if err != nil {
		return data, cleanup, err
	}
	// AutoMigrate
	err = data.db.AutoMigrate(
		&biz.Cluster{},
		&biz.NodeGroup{},
		&biz.BostionHost{},
		&biz.Node{},
		&biz.Project{},
		&biz.App{},
		&biz.AppType{},
		&biz.AppVersion{},
		&biz.AppRelease{},
		&biz.Service{},
		&biz.CI{},
		&biz.User{},
		&biz.AppRepo{},
	)
	if err != nil {
		return data, cleanup, err
	}

	return data, cleanup, nil
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
