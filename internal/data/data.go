package data

import (
	"context"
	"fmt"
	"ocean/internal/conf"
	"time"

	bigcache "github.com/allegro/bigcache/v3"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewInfraRepo, NewClusterRepo, NewAppRepo)

type Data struct {
	c           *conf.Data
	logger      log.Logger
	k8sClient   *kubernetes.Clientset
	db          *gorm.DB
	redisClient *redis.Client
	localCache  *bigcache.BigCache
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	data := &Data{
		c:      c,
		logger: logger,
	}
	cleanup := func() {
		data.redisClient.Close()
		log.NewHelper(logger).Info("closing the data resources")
	}
	data.getK8sClient()
	data.getTiDBClient()
	data.getLocalCache()
	return data, cleanup, nil
}

// 获取数据库连接客户端
func (d *Data) getTiDBClient() {
	// 配置数据库客户端
	dns := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.c.Database.Username, d.c.Database.Password, d.c.Database.Addr, d.c.Database.Database)
	// 创建 GORM 实例
	d.db, _ = gorm.Open(mysql.Open(dns), &gorm.Config{})
}

// 获取k8s客户端
func (d *Data) getK8sClient() {
	cfg, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		// 集群内连接
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return
		}
	}
	// 连接k8s
	d.k8sClient, _ = kubernetes.NewForConfig(cfg)
}

// 获取Redis客户端
func (d *Data) getRedisClient() {
	d.redisClient = redis.NewClient(&redis.Options{
		Addr:     d.c.Redis.Addr,
		Password: d.c.Redis.Password,
	})
}

// 获取本地缓存
func (d *Data) getLocalCache() {
	d.localCache, _ = bigcache.New(context.TODO(), bigcache.DefaultConfig(10*time.Hour))
}
