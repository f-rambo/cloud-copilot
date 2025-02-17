package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/postgres"
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
	gormDialector := postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		c.Data.GetHost(), c.Data.GetUsername(), c.Data.GetPassword(), c.Data.GetDatabase(), c.Data.GetPort()))
	tablePrefix := fmt.Sprintf("%s_", strings.ReplaceAll(c.Data.GetDatabase(), "-", ""))
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
		&biz.IngressControllerRule{},
		&biz.Project{},
		&biz.Workflow{},
		&biz.WorkflowStep{},
		&biz.WorkflowTask{},
		&biz.ContinuousIntegration{},
		&biz.ContinuousDeployment{},
		&biz.Port{},
		&biz.Service{},
		&biz.User{},
		&biz.Role{},
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
