package data

import (
	"context"
	"fmt"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/lib"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	PodLogIndexName string = "pod-log"

	PodKeyWord       string = "pod"
	HostKeyWord      string = "host"
	NamespaceKeyWord string = "namespace"

	ServiceLogIndexName string = "service-log"

	LevelKeyWord       string = "level"
	TraceIdKeyWord     string = "trace_id"
	ServiceIdKeyWord   string = "service_id"
	ServiceNameKeyWord string = "service_name"
	ProjectKeyWord     string = "project"
	WorkspaceKeyWord   string = "workspace"

	TimestampDate string = "@timestamp"
	MessageText   string = "message"
)

type Datarunner interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo, NewProjectRepo, NewWorkspaceRepo)

type Data struct {
	conf          *conf.Bootstrap
	log           *log.Helper
	db            *gorm.DB
	dbLoggerLevel gormlogger.LogLevel

	kafkaConsumer    *lib.KafkaConsumer
	prometheusClient *lib.PrometheusClient
	esClient         *lib.ESClient

	runner        []Datarunner
	runnerChan    chan Datarunner
	runnerErrChan chan error
}

func NewData(ctx context.Context, c *conf.Bootstrap, logger log.Logger) (*Data, func(), error) {
	var err error
	if c.Persistence == nil {
		return nil, func() {}, errors.New("data config is nil")
	}
	data := &Data{
		conf:          c,
		log:           log.NewHelper(logger),
		dbLoggerLevel: gormlogger.Warn,
		runner:        make([]Datarunner, 0),
		runnerChan:    make(chan Datarunner, 1024),
		runnerErrChan: make(chan error, 1),
	}

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if data.db != nil {
			sqlDB, dbErr := data.db.DB()
			if dbErr != nil {
				data.log.Errorf("failed to close db: %v", dbErr)
			}
			sqlDB.Close()
		}
		if data.kafkaConsumer != nil {
			data.kafkaConsumer.Close()
		}
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		c.Persistence.Database.Host, c.Persistence.Database.Username, c.Persistence.Database.Password, c.Persistence.Database.Database, c.Persistence.Database.Port)
	data.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
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
		&biz.Trace{},
		&biz.User{},
		&biz.Role{},
		&biz.UserRole{},
		&biz.Workspace{},
	)
	if err != nil {
		return data, cleanup, err
	}

	if c.Persistence.Kafka != nil && len(c.Persistence.Kafka.Brokers) > 0 && len(c.Persistence.Kafka.Topics) > 0 {
		if c.Persistence.Kafka.GroupId == "" {
			c.Persistence.Kafka.GroupId = c.Server.Name
		}
		data.kafkaConsumer, err = lib.NewKafkaConsumer(c.Persistence.Kafka.Brokers, c.Persistence.Kafka.Topics, c.Persistence.Kafka.GroupId)
		if err != nil {
			return data, cleanup, err
		}
	}

	if c.Persistence.Prometheus != nil && c.Persistence.Prometheus.BaseUrl != "" {
		data.prometheusClient, err = lib.NewPrometheusClient(c.Persistence.Prometheus.BaseUrl)
		if err != nil {
			return data, cleanup, err
		}
		_, err = data.prometheusClient.QueryServerInfo(ctx)
		if err != nil {
			return data, cleanup, err
		}
	}

	if c.Persistence.ElasticSearch != nil && len(c.Persistence.ElasticSearch.Hosts) != 0 && (c.Persistence.ElasticSearch.Username != "" || c.Persistence.ElasticSearch.Password != "") {
		esConfig := lib.ESConfig{
			Addresses: c.Persistence.ElasticSearch.Hosts,
			Username:  c.Persistence.ElasticSearch.Username,
			Password:  c.Persistence.ElasticSearch.Password,
		}
		data.esClient, err = lib.NewESClient(esConfig)
		if err != nil {
			return data, cleanup, err
		}
		_, err = data.esClient.Info()
		if err != nil {
			return data, cleanup, err
		}
		err = data.esClient.CreateIndex(ctx, PodLogIndexName, map[string]any{
			"properties": map[string]any{
				HostKeyWord: map[string]any{
					"type": "keyword",
				},
				PodKeyWord: map[string]any{
					"type": "keyword",
				},
				NamespaceKeyWord: map[string]any{
					"type": "keyword",
				},
				TimestampDate: map[string]any{
					"type": "date",
				},
				MessageText: map[string]any{
					"type":  "text",
					"index": false,
				},
			},
		})
		if err != nil {
			return data, cleanup, err
		}
		err = data.esClient.CreateIndex(ctx, ServiceLogIndexName, map[string]any{
			"properties": map[string]any{
				LevelKeyWord: map[string]any{
					"type": "keyword",
				},
				TraceIdKeyWord: map[string]any{
					"type": "keyword",
				},
				ServiceIdKeyWord: map[string]any{
					"type": "keyword",
				},
				HostKeyWord: map[string]any{
					"type": "keyword",
				},
				TimestampDate: map[string]any{
					"type": "date",
				},
				MessageText: map[string]any{
					"type":  "text",
					"index": false,
				},
			},
		})
		if err != nil {
			return data, cleanup, err
		}
	}

	return data, cleanup, nil
}

func (d *Data) registerRunner(runner Datarunner) {
	select {
	case d.runnerChan <- runner:
	default:
	}
}

func (d *Data) addRunnerError(err error) {
	select {
	case d.runnerErrChan <- err:
	default:
	}
}

func (d *Data) Start(ctx context.Context) error {
	for {
		select {
		case runner, ok := <-d.runnerChan:
			if !ok {
				return nil
			}
			go func() {
				err := runner.Start(ctx)
				if err != nil {
					d.addRunnerError(err)
				}
			}()
			d.runner = append(d.runner, runner)
		case <-ctx.Done():
			return nil
		case err, ok := <-d.runnerErrChan:
			if !ok {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

func (d *Data) Stop(ctx context.Context) error {
	for _, runner := range d.runner {
		err := runner.Stop(ctx)
		if err != nil {
			return err
		}
	}
	close(d.runnerChan)
	close(d.runnerErrChan)
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
