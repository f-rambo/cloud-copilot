package data

import (
	"fmt"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/argoworkflows"
	"github.com/f-rambo/ocean/pkg/operatorapp"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewClusterRepo, NewAppRepo, NewServicesRepo, NewUserRepo, NewProjectRepo)

type Data struct {
	c                 *conf.Data
	kubeConfig        *conf.Kubernetes
	logger            log.Logger
	kubeClient        *kubernetes.Clientset
	operatorappClient *operatorapp.AppV1Alpha1Client
	workflowClient    *argoworkflows.WorkflowV1Alpha1Client
	db                *gorm.DB
}

func NewData(c *conf.Data, kube *conf.Kubernetes, logger log.Logger) (*Data, func(), error) {
	var err error
	data := &Data{
		c:          c,
		kubeConfig: kube,
		logger:     logger,
	}
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	err = data.newKubernetes()
	if err != nil {
		log.NewHelper(logger).Info("kubernetes client error, check whether the cluster has been deployed. If the cluster is not deployed, ignore this error")
	}
	data.db, err = newDB(c.Database)
	if err != nil {
		log.NewHelper(logger).Info("database client error, check whether the database has been deployed. If the database is not deployed, ignore this error")
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
			dbFilePath = "file::memory:?cache=shared" // 使用内存
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
		// &biz.Role{},
		// &biz.UserRole{},
		&biz.AppHelmRepo{},
	)
	if err != nil {
		return client, err
	}
	return client, nil
}

// 获取k8s客户端
func (d *Data) newKubernetes() error {
	c := d.kubeConfig
	kubeconfig := c.GetKubeConfig()
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	cfg, err := clientcmd.BuildConfigFromFlags(c.GetMasterUrl(), kubeconfig)
	if err != nil {
		// 尝试集群内连接
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
		// service operator-app
		d.operatorappClient, err = operatorapp.NewForConfig(cfg)
		if err != nil {
			return err
		}
		// argo workflow
		d.workflowClient, err = argoworkflows.NewForConfig(cfg)
		if err != nil {
			return err
		}
	}
	d.kubeClient = k8sClient
	return nil
}
