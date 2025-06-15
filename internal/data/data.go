package data

import (
	"context"
	"fmt"
	"sync"
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
	if c.Persistence == nil {
		return nil, func() {}, errors.New("data config is nil")
	}

	var err error
	data := &Data{
		conf:          c,
		log:           log.NewHelper(logger),
		dbLoggerLevel: gormlogger.Warn,
		runner:        make([]Datarunner, 0),
		runnerChan:    make(chan Datarunner, 1024),
		runnerErrChan: make(chan error, 1),
	}

	syncOnce := new(sync.Once)
	syncOnce.Do(func() {
		if err = data.newDatabase(); err != nil {
			return
		}
		if err = data.newKafka(); err != nil {
			return
		}
		if err = data.newPrometheus(ctx); err != nil {
			return
		}
		if err = data.newElasticSearch(ctx); err != nil {
			return
		}
	})
	if err != nil {
		return data, data.Clean, err
	}

	return data, data.Clean, nil
}

func (d *Data) Clean() {
	d.log.Info("closing the data resources")
	if d.db != nil {
		sqlDB, dbErr := d.db.DB()
		if dbErr != nil {
			d.log.Errorf("failed to close db: %v", dbErr)
		}
		sqlDB.Close()
	}
	if d.kafkaConsumer != nil {
		d.kafkaConsumer.Close()
	}
}

func (d *Data) newDatabase() error {

	c := d.conf

	defaultDSN := fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		c.Persistence.Database.Host, c.Persistence.Database.Username, c.Persistence.Database.Password, c.Persistence.Database.Port)
	tmpDB, err := gorm.Open(postgres.Open(defaultDSN+" dbname=postgres"), &gorm.Config{})
	if err != nil {
		return errors.Wrap(err, "connect to postgres default db failed")
	}
	var exists bool
	dbName := c.Persistence.Database.Database
	checkSQL := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", dbName)
	err = tmpDB.Raw(checkSQL).Scan(&exists).Error
	if err != nil {
		return errors.Wrap(err, "check database exists failed")
	}
	if !exists {
		createSQL := fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING 'UTF8'", dbName)
		if err = tmpDB.Exec(createSQL).Error; err != nil {
			return errors.Wrap(err, "create database failed")
		}
	}
	sqlDB, _ := tmpDB.DB()
	sqlDB.Close()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		c.Persistence.Database.Host, c.Persistence.Database.Username, c.Persistence.Database.Password, dbName, c.Persistence.Database.Port)
	d.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         d,
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		return errors.Wrap(err, "connect to postgres failed")
	}

	err = d.db.AutoMigrate(
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
		&biz.Volume{},
		&biz.Pod{},
		&biz.Workflow{},
		&biz.WorkflowStep{},
		&biz.WorkflowTask{},
		&biz.ContinuousIntegration{},
		&biz.ContinuousDeployment{},
		&biz.Trace{},
		&biz.User{},
		&biz.Role{},
		&biz.Permission{},
		&biz.Workspace{},
		&biz.WorkspaceRole{},
		&biz.WorkspaceClusterRelationship{},
	)
	if err != nil {
		return errors.Wrap(err, "auto migrate failed")
	}
	return nil
}

func (d *Data) newKafka() error {
	c := d.conf
	if c.Persistence.Kafka == nil || len(c.Persistence.Kafka.Brokers) == 0 || len(c.Persistence.Kafka.Topics) == 0 {
		return nil
	}
	if c.Persistence.Kafka.GroupId == "" {
		c.Persistence.Kafka.GroupId = c.Server.Name
	}
	kafkaConsumer, err := lib.NewKafkaConsumer(c.Persistence.Kafka.Brokers, c.Persistence.Kafka.Topics, c.Persistence.Kafka.GroupId)
	if err != nil {
		return errors.Wrap(err, "new kafka consumer failed")
	}
	d.kafkaConsumer = kafkaConsumer
	return nil
}

func (d *Data) newPrometheus(ctx context.Context) error {
	c := d.conf

	if c.Persistence.Prometheus == nil || c.Persistence.Prometheus.BaseUrl == "" {
		return nil
	}
	prometheusClient, err := lib.NewPrometheusClient(c.Persistence.Prometheus.BaseUrl)
	if err != nil {
		return errors.Wrap(err, "new prometheus client failed")
	}
	_, err = prometheusClient.QueryServerInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "query prometheus server info failed")
	}
	d.prometheusClient = prometheusClient
	return nil
}

func (d *Data) newElasticSearch(ctx context.Context) error {
	c := d.conf

	if c.Persistence.ElasticSearch == nil || len(c.Persistence.ElasticSearch.Hosts) == 0 || c.Persistence.ElasticSearch.Username == "" || c.Persistence.ElasticSearch.Password == "" {
		return nil
	}
	esConfig := lib.ESConfig{
		Addresses: c.Persistence.ElasticSearch.Hosts,
		Username:  c.Persistence.ElasticSearch.Username,
		Password:  c.Persistence.ElasticSearch.Password,
	}
	esClient, err := lib.NewESClient(esConfig)
	if err != nil {
		return errors.Wrap(err, "new elastic search client failed")
	}
	_, err = esClient.Info()
	if err != nil {
		return errors.Wrap(err, "query elastic search server info failed")
	}
	err = esClient.CreateIndex(ctx, PodLogIndexName, map[string]any{
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
		return errors.Wrap(err, "create elastic search index failed")
	}
	err = esClient.CreateIndex(ctx, ServiceLogIndexName, map[string]any{
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
		return errors.Wrap(err, "create elastic search index failed")
	}
	d.esClient = esClient
	return nil
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
