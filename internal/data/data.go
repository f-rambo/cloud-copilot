package data

import (
	"fmt"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo, NewProjectRepo)

type Data struct {
	c      *conf.Data
	logger log.Logger
	db     *gorm.DB
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	var err error
	data := &Data{
		c:      c,
		logger: logger,
	}
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	data.db, err = newDB(c.Database)
	if err != nil {
		log.NewHelper(logger).Info("database client error, check whether the database has been deployed. If the database is not deployed, ignore this error")
	}
	return data, cleanup, nil
}

func newDB(c conf.Database) (*gorm.DB, error) {
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
	if c.GetDriver() != "mysql" && c.GetDriver() != "postgres" {
		dbFilePath := c.GetDBFilePath()
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
