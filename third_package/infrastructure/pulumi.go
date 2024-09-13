package infrastructure

import (
	"context"
	"encoding/json"
	"io"

	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	PulumiAlicloud        = "alicloud"
	PulumiAlicloudVersion = "3.56.0"

	PulumiAws        = "aws"
	PulumiAwsVersion = "6.38.0"

	PulumiGoogle        = "google"
	PulumiGoogleVersion = "4.12.0"

	PulumiKubernetes        = "kubernetes"
	PulumiKubernetesVersion = "4.12.0"
)

const (
	PulumiPackageName = "pulumi"
)

type PulumiAPI struct {
	projectName   string
	stackName     string
	plugins       []PulumiPlugin
	config        map[string]string
	deployFunc    func(ctx *pulumi.Context) error
	stack         auto.Stack
	pulumiCommand auto.PulumiCommand
	env           map[string]string
	w             io.Writer
}

type PulumiPlugin struct {
	Kind    string
	Version string
}

type PulumiFunc func(ctx *pulumi.Context) error

var CleanFunc PulumiFunc = func(ctx *pulumi.Context) error {
	return nil
}

func NewPulumiAPI(ctx context.Context, w io.Writer) *PulumiAPI {
	p := &PulumiAPI{
		w: w,
		env: map[string]string{
			"PULUMI_CONFIG_PASSPHRASE": "a8F3zQpL9wXyT6kBn4UvJhR2Vc0M1sCd",
		},
	}
	w.Write([]byte("Initializing Pulumi API \n"))
	p.autoInstallPulumiCli(ctx)
	return p
}

func (p *PulumiAPI) autoInstallPulumiCli(ctx context.Context) (err error) {
	p.w.Write([]byte("Installing Pulumi CLI \n"))
	p.pulumiCommand, err = auto.InstallPulumiCommand(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to install pulumi command")
	}
	p.w.Write([]byte("Pulumi CLI installed successfully \n"))
	return nil
}

func (p *PulumiAPI) ProjectName(projectName string) *PulumiAPI {
	p.projectName = projectName
	return p
}

func (p *PulumiAPI) StackName(stackName string) *PulumiAPI {
	p.stackName = stackName
	return p
}

func (p *PulumiAPI) Plugin(plugins ...PulumiPlugin) *PulumiAPI {
	p.plugins = plugins
	return p
}

func (p *PulumiAPI) RegisterDeployFunc(deployFunc func(ctx *pulumi.Context) error) *PulumiAPI {
	p.deployFunc = deployFunc
	return p
}

func (p *PulumiAPI) Config(config map[string]string) *PulumiAPI {
	if p.config == nil {
		p.config = make(map[string]string)
	}
	for k, v := range config {
		p.config[k] = v
	}
	return p
}

func (p *PulumiAPI) Env(env map[string]string) *PulumiAPI {
	if p.env == nil {
		p.env = make(map[string]string)
	}
	for k, v := range env {
		p.env[k] = v
	}
	return p
}

func (p *PulumiAPI) buildPulumiResources(ctx context.Context) (err error) {
	if p.projectName == "" || p.stackName == "" || p.deployFunc == nil {
		return errors.New("projectName, stackName, plugin, pluginVersion and deployFunc must be set")
	}
	if len(p.plugins) == 0 {
		return errors.New("plugin and pluginVersion must be set")
	}
	pulumiStorePath, err := utils.GetPackageStorePathByNames(PulumiPackageName)
	if err != nil {
		return err
	}
	stackConfigDir, err := utils.GetPackageStorePathByNames(PulumiPackageName, "stacks", p.stackName)
	if err != nil {
		return err
	}
	backendFilePath := "file://" + pulumiStorePath
	p.stack, err = auto.UpsertStackInlineSource(ctx, p.stackName, p.projectName, p.deployFunc,
		auto.Pulumi(p.pulumiCommand), auto.EnvVars(p.env),
		auto.Project(workspace.Project{
			Name:           tokens.PackageName(p.projectName),
			Runtime:        workspace.NewProjectRuntimeInfo("go", nil),
			StackConfigDir: stackConfigDir,
			Backend: &workspace.ProjectBackend{
				URL: backendFilePath,
			},
		}))
	if err != nil {
		return errors.Errorf("Failed to create or update stack %s: %v", p.stackName, err)
	}

	p.w.Write([]byte("Setting stack config \n"))

	if p.config != nil {
		for k, v := range p.config {
			p.stack.SetConfig(ctx, k, auto.ConfigValue{Value: v})
		}
	}

	workspace := p.stack.Workspace()
	if workspace == nil {
		return errors.New("Failed to get workspace")
	}

	p.w.Write([]byte("Installing plugins \n"))
	for _, plugin := range p.plugins {
		workspace.InstallPlugin(ctx, plugin.Kind, plugin.Version)
	}

	_, err = p.stack.Refresh(ctx)
	if err != nil {
		p.stack.Cancel(ctx)
		return errors.Errorf("Failed to refresh stack %s: %v", p.stackName, err)
	}

	return nil
}

func (p *PulumiAPI) Up(ctx context.Context) (outPut string, err error) {
	err = p.buildPulumiResources(ctx)
	if err != nil {
		return "", err
	}

	stdoutStreamer := optup.ProgressStreams(p.w)

	res, err := p.stack.Up(ctx, stdoutStreamer)
	if err != nil {
		return "", errors.Errorf("Failed to update stack %s: %v", p.stackName, err)
	}
	output, err := json.Marshal(res.Outputs)
	if err != nil {
		return "", errors.Errorf("Failed to marshal stack %s outputs: %v", p.stackName, err)
	}
	return string(output), nil
}

func (p *PulumiAPI) Destroy(ctx context.Context) (outPut string, err error) {
	err = p.buildPulumiResources(ctx)
	if err != nil {
		return "", err
	}

	stdoutStreamer := optdestroy.ProgressStreams(p.w)

	res, err := p.stack.Destroy(ctx, stdoutStreamer)
	if err != nil {
		return "", errors.Errorf("Failed to destroy stack %s: %v", p.stackName, err)
	}

	err = p.stack.Workspace().RemoveStack(ctx, p.stackName)
	if err != nil {
		return "", errors.Errorf("Failed to remove stack %s: %v", p.stackName, err)
	}

	return res.StdOut, nil
}

func (p *PulumiAPI) Preview(ctx context.Context) (outPut string, err error) {
	err = p.buildPulumiResources(ctx)
	if err != nil {
		return "", err
	}

	stdoutStreamer := optpreview.ProgressStreams(p.w)

	res, err := p.stack.Preview(ctx, stdoutStreamer)
	if err != nil {
		return "", errors.Errorf("Failed to preview stack %s: %v", p.stackName, err)
	}

	return res.StdOut, nil
}
