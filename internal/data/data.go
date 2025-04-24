package data

import (
	"context"
	"fmt"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo, NewProjectRepo, NewWorkspaceRepo)

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

	dbFile := fmt.Sprintf("%s.db", c.Server.Name)
	if !utils.IsFileExist(dbFile) {
		err = utils.CreateFile(dbFile)
		if err != nil {
			return data, cleanup, errors.New("create db file failed")
		}
	}

	data.db, err = gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		Logger:         data,
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		return data, cleanup, err
	}

	err = data.db.AutoMigrate(
		&biz.AppType{},
		&biz.AppRepo{},
		&biz.App{},
		&biz.AppVersion{},
		&biz.AppRelease{},
		&biz.AppReleaseResource{},
		&biz.Cluster{},
		&biz.Node{},
		&biz.NodeGroup{},
		&biz.CloudResource{},
		&biz.Security{},
		&biz.Disk{},
		&biz.Project{},
		&biz.Service{},
		&biz.Port{},
		&biz.Workflow{},
		&biz.WorkflowStep{},
		&biz.WorkflowTask{},
		&biz.ContinuousIntegration{},
		&biz.ContinuousDeployment{},
		&biz.User{},
		&biz.Role{},
		&biz.UserRole{},
		&biz.Workspace{},
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
