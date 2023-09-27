package service

import (
	"fmt"

	"github.com/f-rambo/ocean/api/service/v1alpha1"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

var (
	client            v1alpha1.ServiceServiceClient
	l                 *log.Helper
	serviceDir        = "./service"
	serviceYamlFile   = fmt.Sprintf("%s/service.yaml", serviceDir)
	serviceCiYamlFile = fmt.Sprintf("%s/ci.yaml", serviceDir)
	confYamlFile      = fmt.Sprintf("%s/config.yaml", serviceDir)
	secretYamlFile    = fmt.Sprintf("%s/secret.yaml", serviceDir)
	workflowYamlFile  = fmt.Sprintf("%s/workflow.yaml", serviceDir)
)

func NewServiceCommand(conn *grpc.ClientConn, logger log.Logger) *cobra.Command {
	client = v1alpha1.NewServiceServiceClient(conn)
	l = log.NewHelper(logger)
	command := &cobra.Command{
		Use:   "service",
		Short: "Manage services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	command.AddCommand(example(), apply(), get(), del())
	return command
}

func apply() *cobra.Command {
	var (
		servicePath  string
		ciPath       string
		configPath   string
		secretPath   string
		workflowPath string
	)
	command := &cobra.Command{
		Use:   "apply",
		Short: "apply service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 检查文件是否存在
			if !utils.CheckFileIsExist(servicePath) {
				return fmt.Errorf("service yaml file not exist")
			}
			if !utils.CheckFileIsExist(ciPath) {
				return fmt.Errorf("ci yaml file not exist")
			}
			if !utils.CheckFileIsExist(configPath) {
				return fmt.Errorf("confgi file not exist")
			}
			if !utils.CheckFileIsExist(secretPath) {
				return fmt.Errorf("secret file not exist")
			}
			if !utils.CheckFileIsExist(workflowPath) {
				return fmt.Errorf("workflow file not exist")
			}
			// save service & if cluster id == 0 update cluster id
			serviceContent, err := utils.ReadFile(servicePath)
			if err != nil {
				return err
			}
			wfContent, err := utils.ReadFile(workflowPath)
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
			service := &v1alpha1.ServiceV1Alpha1{}
			err = yaml.Unmarshal([]byte(serviceContent), service)
			if err != nil {
				return err
			}
			service.Spec.Workflow = wfContent
			service.Spec.Config = configContent
			service.Spec.Secret = secretContent
			serviceRes, err := client.SaveService(cmd.Context(), service.Spec)
			if err != nil {
				return err
			}
			if service.Spec.Id == 0 {
				service.Spec.Id = serviceRes.GetId()
				service.Spec.Workflow = ""
				service.Spec.Config = ""
				service.Spec.Secret = ""
				serviceYamlData, err := yaml.Marshal(service)
				if err != nil {
					return err
				}
				err = utils.WriteFile(serviceYamlFile, string(serviceYamlData))
				if err != nil {
					return err
				}
			}
			l.Info("service save successd")
			// save ci & if ci id == 0 update ci id
			ciContent, err := utils.ReadFile(ciPath)
			if err != nil {
				return err
			}
			ci := &v1alpha1.CIV1Alpha1{}
			err = yaml.Unmarshal([]byte(ciContent), ci)
			if err != nil {
				return err
			}
			ciRes, err := client.SaveCI(cmd.Context(), ci.Spec)
			if err != nil {
				return err
			}
			if ci.Spec.Id == 0 {
				ci.Spec.Id = ciRes.GetId()
				ci.Spec.ServiceId = service.Spec.Id
				ciYamlData, err := yaml.Marshal(ci)
				if err != nil {
					return err
				}
				err = utils.WriteFile(ciPath, string(ciYamlData))
				if err != nil {
					return err
				}
			}
			l.Info("ci save successd")
			// deploy
			_, err = client.Deploy(cmd.Context(), &v1alpha1.CIID{Id: ci.Spec.Id})
			if err != nil {
				return err
			}
			l.Info("deploy successd")
			return nil
		},
	}
	command.Flags().StringVar(&servicePath, "service", serviceYamlFile, "service yaml flile")
	command.Flags().StringVar(&ciPath, "ci", serviceCiYamlFile, "ci yaml flile")
	command.Flags().StringVar(&configPath, "config", confYamlFile, "config yaml flile")
	command.Flags().StringVar(&secretPath, "secret", secretYamlFile, "secret yaml flile")
	command.Flags().StringVar(&workflowPath, "workflow", workflowYamlFile, "workflow yaml flile")
	return command
}

func get() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: `get service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 无参数获取列表，有参数获取一个
			if len(args) == 0 {
				services, err := client.GetServices(cmd.Context(), &emptypb.Empty{})
				if err != nil {
					return err
				}
				if len(services.Services) == 0 {
					fmt.Println("no service")
				}
				for _, service := range services.Services {
					fmt.Printf("service id: %d, name: %s\n", service.GetId(), service.GetName())
				}
				return nil
			}
			serviceID := cast.ToInt32(args[0])
			if serviceID == 0 {
				return fmt.Errorf("service id is empty, mast be an int number")
			}
			service, err := client.GetService(cmd.Context(), &v1alpha1.ServiceID{Id: serviceID})
			if err != nil {
				return err
			}
			if service == nil || service.Id == 0 {
				fmt.Println("not found service : ", args[0])
				return nil
			}
			fmt.Printf("service id: %d, name: %s\n", service.GetId(), service.GetName())
			for _, ci := range service.Cis {
				fmt.Printf("id: %d, version: %s, branch: %s, tag: %s\n", ci.GetId(), ci.GetVersion(), ci.GetBranch(), ci.GetTag())
			}
			return nil
		},
	}
	return command
}

func del() *cobra.Command {
	command := &cobra.Command{
		Use:   "delete",
		Short: `delete service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			serviceID := cast.ToInt32(args[0])
			if serviceID == 0 {
				return fmt.Errorf("service id is empty, mast be an int number")
			}
			service, err := client.GetService(cmd.Context(), &v1alpha1.ServiceID{Id: serviceID})
			if err != nil {
				return err
			}
			if service.Id == 0 {
				return fmt.Errorf("service not exist")
			}
			// undeploy
			_, err = client.UnDeploy(cmd.Context(), &v1alpha1.ServiceID{Id: serviceID})
			if err != nil {
				return err
			}
			l.Info("undeploy successd")
			// ci
			// todo 批量删除
			for _, ci := range service.Cis {
				_, err = client.DeleteCI(cmd.Context(), &v1alpha1.CIID{Id: ci.GetId()})
				if err != nil {
					return err
				}
			}
			l.Info("delete ci successd")
			// service
			_, err = client.DeleteService(cmd.Context(), &v1alpha1.ServiceID{Id: serviceID})
			if err != nil {
				return err
			}
			l.Info("delete service successd")
			return nil
		},
	}
	return command
}

func example() *cobra.Command {
	return &cobra.Command{
		Use: "example",
		Short: `
		ocean service example
		generate example
		-------------
		./service/service.yaml
		./service/ci.yaml
		./service/config.yaml
		./service/secret.yaml
		./service/workflow.yaml
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			oceanService, err := client.GetOceanService(cmd.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			if !utils.CheckFileIsExist(serviceDir) {
				utils.CreateDir(serviceDir)
			}
			metaData := &v1alpha1.MetaData{
				Name:      "ocean",
				Namespace: "default",
			}
			service := &v1alpha1.ServiceV1Alpha1{
				MetaData: metaData,
				Kind:     "service",
				Spec:     oceanService,
			}
			ciYaml, err := yaml.Marshal(oceanService.Cis[0])
			if err != nil {
				return err
			}
			err = utils.WriteFile(serviceCiYamlFile, string(ciYaml))
			if err != nil {
				return err
			}
			err = utils.WriteFile(confYamlFile, oceanService.Config)
			if err != nil {
				return err
			}
			err = utils.WriteFile(secretYamlFile, oceanService.Secret)
			if err != nil {
				return err
			}
			err = utils.WriteFile(workflowYamlFile, oceanService.Workflow)
			if err != nil {
				return err
			}
			service.Spec.Cis = nil
			service.Spec.Secret = ""
			service.Spec.Workflow = ""
			service.Spec.Config = ""
			serviceYaml, err := yaml.Marshal(service)
			if err != nil {
				return err
			}
			err = utils.WriteFile(serviceYamlFile, string(serviceYaml))
			if err != nil {
				return err
			}
			l.Info("generate example successd")
			return nil
		},
	}
}
