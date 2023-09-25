package app

import (
	"fmt"

	v1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

var (
	client         v1alpha1.AppServiceClient
	l              *log.Helper
	appDir         = "./app"
	appYamlFile    = fmt.Sprintf("%s/app.yaml", appDir)
	confYamlFile   = fmt.Sprintf("%s/config.yaml", appDir)
	secretYamlFile = fmt.Sprintf("%s/secret.yaml", appDir)
)

func NewAppommand(conn *grpc.ClientConn, logger log.Logger) *cobra.Command {
	client = v1alpha1.NewAppServiceClient(conn)
	l = log.NewHelper(logger)
	command := &cobra.Command{
		Use:   "app",
		Short: `Manage the helm application`,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			return nil
		},
	}
	command.AddCommand(apply(), list(), delete(), example())
	return command
}

func apply() *cobra.Command {
	var (
		appPath    string
		configPath string
		secretPath string
	)
	command := &cobra.Command{
		Use: "apply",
		Short: `
		ocean app apply -app app.yaml -config config.yaml -secret secret.yaml
		default app.yaml: ./app/app.yaml
		default config.yaml: ./app/config.yaml
		default secret.yaml: ./app/secret.yaml
		`,
		RunE: func(c *cobra.Command, args []string) error {
			// save & apply
			if !utils.CheckFileIsExist(appYamlFile) {
				return fmt.Errorf("app yaml file not exist")
			}
			if !utils.CheckFileIsExist(configPath) {
				return fmt.Errorf("app config yaml file not exist")
			}
			appContent, err := utils.ReadFile(appYamlFile)
			if err != nil {
				return err
			}
			configContent, err := utils.ReadFile(configPath)
			if err != nil {
				return err
			}
			secretContent, err := utils.ReadFile(secretPath)
			if err != nil {
				return err
			}
			app := &v1alpha1.AppV1Alpha1{}
			err = yaml.Unmarshal([]byte(appContent), &app)
			if err != nil {
				return err
			}
			app.Spec.Config = configContent
			app.Spec.Secret = secretContent
			// todo 返回appID
			appRes, err := client.Save(c.Context(), app.Spec)
			if err != nil {
				return err
			}
			l.Info("save app success", "appID", appRes.AppID)
			app.Spec.Id = appRes.AppID
			appYaml, err := yaml.Marshal(app)
			if err != nil {
				return err
			}
			err = utils.WriteFile(appYamlFile, string(appYaml))
			if err != nil {
				return err
			}
			l.Info("write app yaml success")
			_, err = client.Apply(c.Context(), &v1alpha1.AppID{AppID: app.Spec.Id})
			if err != nil {
				return err
			}
			l.Info("apply app success")
			return nil
		},
	}
	command.Flags().StringVar(&appPath, "app", appYamlFile, "app yaml flile")
	command.Flags().StringVar(&configPath, "config", confYamlFile, "app config yaml flile")
	command.Flags().StringVar(&secretPath, "secret", secretYamlFile, "app secret yaml flile")
	return command
}

func list() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Short: `
		ocean app list [cluster id]
		`,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			apps, err := client.GetApps(c.Context(), &v1alpha1.ClusterID{ClusterID: cast.ToInt32(args[0])})
			if err != nil {
				return err
			}
			for _, app := range apps.Apps {
				fmt.Printf("appID: %d  name: %s  namespace: %s chartname: %s  version: %s  repo: %s  repoUrl: %s  clusterID: %d\n",
					app.Id, app.Name, app.Namespace, app.ChartName, app.Version, app.RepoName, app.RepoUrl, app.ClusterID)
			}
			return nil
		},
	}
}

func delete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "ocean app delete [app id]",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			app, err := client.GetApp(c.Context(), &v1alpha1.AppID{AppID: cast.ToInt32(args[0])})
			if err != nil {
				return err
			}
			if app == nil || app.Id == 0 {
				return fmt.Errorf("app not exist")
			}
			_, err = client.Delete(c.Context(), &v1alpha1.AppID{AppID: app.Id})
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func example() *cobra.Command {
	return &cobra.Command{
		Use: "example",
		Short: `
		ocean app example
		generate app yaml file
		---------------------
		./app/app.yaml
		./app/config.yaml
		`,
		RunE: func(c *cobra.Command, args []string) error {
			metaData := &v1alpha1.MetaData{
				Name:      "redis",
				Namespace: "default",
			}
			app := &v1alpha1.AppV1Alpha1{
				MetaData: metaData,
				Kind:     "app",
				Spec: &v1alpha1.App{
					Name:      "redis",
					RepoName:  "bitnami",
					RepoUrl:   "https://charts.bitnami.com/bitnami",
					ChartName: "bitnami/redis",
					Version:   "18.0.0",
					ClusterID: 1,
					Namespace: "default",
				},
			}
			config := `
auth:
  ## @param auth.enabled Enable password authentication
  ##
  enabled: true
  ## @param auth.sentinel Enable password authentication on sentinels too
  ##
  sentinel: true
  ## @param auth.password Redis&reg; password
  ## Defaults to a random 10-character alphanumeric string if not set
  ##
  password: ""
  ## @param auth.existingSecret The name of an existing secret with Redis&reg; credentials
  ## NOTE: When it's set, the previous auth.password parameter is ignored
  ##
  existingSecret: ""
  ## @param auth.existingSecretPasswordKey Password key to be retrieved from existing secret
  ## NOTE: ignored unless auth.existingSecret parameter is set
  ##
  existingSecretPasswordKey: ""
  ## @param auth.usePasswordFiles Mount credentials as files instead of using an environment variable
  ##
  usePasswordFiles: false
			`
			secret := ``
			appyaml, err := yaml.Marshal(app)
			if err != nil {
				return err
			}
			if !utils.CheckFileIsExist(appDir) {
				err = utils.CreateDir(appDir)
				if err != nil {
					return err
				}
			}
			err = utils.WriteFile(appYamlFile, string(appyaml))
			if err != nil {
				return err
			}
			err = utils.WriteFile(confYamlFile, config)
			if err != nil {
				return err
			}
			err = utils.WriteFile(secretYamlFile, secret)
			if err != nil {
				return err
			}
			l.Info("successd")
			return nil
		},
	}
}
