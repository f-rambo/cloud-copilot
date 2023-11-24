package data

import (
	"context"
	"fmt"
	"time"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/argoworkflows"
	"github.com/f-rambo/ocean/pkg/operatorapp"
	"github.com/f-rambo/ocean/pkg/semaphore"

	bigcache "github.com/allegro/bigcache/v3"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo)

type Data struct {
	c                 *conf.Data
	logger            log.Logger
	k8sClient         *kubernetes.Clientset
	db                *gorm.DB
	redisClient       *redis.Client
	localCache        *bigcache.BigCache
	semaphore         *semaphore.Semaphore
	operatorappClient *operatorapp.AppV1Alpha1Client
	workflowClient    *argoworkflows.WorkflowV1Alpha1Client
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	var err error
	data := &Data{
		c:      c,
		logger: logger,
	}
	cleanup := func() {
		data.redisClient.Close()
		log.NewHelper(logger).Info("closing the data resources")
	}
	err = data.newKubernetes()
	if err != nil {
		log.NewHelper(logger).Info("kubernetes client error, check whether the cluster has been deployed. If the cluster is not deployed, ignore this error")
	}
	data.db, err = newDB(c.Database)
	if err != nil {
		return nil, cleanup, err
	}
	data.redisClient, err = newRedis(c.Redis)
	if err != nil {
		return nil, cleanup, err
	}
	data.localCache, err = newCache()
	if err != nil {
		return nil, cleanup, err
	}
	data.semaphore, err = semaphore.NewSemaphore(c.Semaphore)
	if err != nil {
		return nil, cleanup, err
	}
	return data, cleanup, nil
}

// 获取数据库连接客户端
func newDB(c conf.Database) (*gorm.DB, error) {
	var client *gorm.DB
	var err error
	if c.GetDriver() == "mysql" {
		dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
			c.GetUsername(), c.GetPassword(), c.GetHost(), c.GetPort(), c.GetDatabase())
		client, err = gorm.Open(mysql.Open(dns), &gorm.Config{})
		if err != nil {
			return client, err
		}
	}
	if c.GetDriver() == "postgres" {
		dns := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
			c.GetHost(), c.GetUsername(), c.GetPassword(), c.GetDatabase(), c.GetPort())
		client, err = gorm.Open(postgres.Open(dns), &gorm.Config{})
		if err != nil {
			return client, err
		}
	}
	// AutoMigrate
	err = client.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").
		AutoMigrate(&biz.Cluster{}, &biz.Node{}, &biz.App{}, &biz.Service{}, &biz.CI{})
	if err != nil {
		return client, err
	}
	return client, nil
}

// 获取k8s客户端 todo kubeconfig path masterurl
func (d *Data) newKubernetes() error {
	c := d.c.Kubernetes
	kubeconfig := c.GetKubeConfig()
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	cfg, err := clientcmd.BuildConfigFromFlags(c.GetMasterUrl(), kubeconfig)
	if err != nil {
		// 集群内连接
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	}
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	if k8sClient != nil {
		d.operatorappClient, err = operatorapp.NewForConfig(cfg)
		if err != nil {
			return err
		}
		d.workflowClient, err = argoworkflows.NewForConfig(cfg)
		if err != nil {
			return err
		}
	}
	d.k8sClient = k8sClient
	return nil
}

// 获取Redis客户端
func newRedis(c conf.Redis) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.GetHost(), c.GetPort()),
		Password: c.Password,
		DB:       int(c.GetDb()),
	})
	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}
	return client, nil
}

// 获取本地缓存
func newCache() (*bigcache.BigCache, error) {
	return bigcache.New(context.TODO(), bigcache.DefaultConfig(time.Hour))
}
