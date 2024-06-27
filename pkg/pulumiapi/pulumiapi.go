package pulumiapi

import (
	"context"
	"fmt"
	"os"

	"github.com/blang/semver"
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
	configPassphrase = "a8F3zQpL9wXyT6kBn4UvJhR2Vc0M1sCd"
	pulumiOrg        = "ocean"
	pulumiCliVersion = "3.117.0"
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
	outPut        chan string
}

type PulumiPlugin struct {
	Kind    string
	Version string
}

type PulumiFunc func(ctx *pulumi.Context) error

func NewPulumiAPI(ctx context.Context, output chan string) *PulumiAPI {
	if output == nil {
		return nil
	}
	p := &PulumiAPI{
		outPut: output,
		env: map[string]string{
			"PULUMI_CONFIG_PASSPHRASE": configPassphrase,
		},
	}
	p.outPut <- "Initializing Pulumi API \n"
	p.autoInstallPulumiCli(ctx)
	return p
}

func (p *PulumiAPI) autoInstallPulumiCli(ctx context.Context) (err error) {
	p.outPut <- "Installing Pulumi CLI \n"
	tempDir, err := os.MkdirTemp("", "pulumi_cli_installation")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	p.pulumiCommand, err = auto.InstallPulumiCommand(ctx, &auto.PulumiCommandOptions{
		Version: semver.MustParse(pulumiCliVersion),
		Root:    tempDir,
	})
	if err != nil {
		return errors.Wrap(err, "failed to install pulumi command")
	}
	p.outPut <- "Pulumi CLI installed successfully \n"
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

	p.stack, err = auto.UpsertStackInlineSource(ctx, p.stackName, p.projectName, p.deployFunc,
		auto.Pulumi(p.pulumiCommand), auto.EnvVars(p.env),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(pulumiOrg),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			Backend: &workspace.ProjectBackend{
				URL: "file://~/.pulumi",
			},
		}))
	if err != nil {
		return errors.Errorf("Failed to create or update stack %s: %v", p.stackName, err)
	}

	p.outPut <- fmt.Sprintf("Stack %s created or updated successfully\n", p.stackName)

	if p.config != nil {
		for k, v := range p.config {
			p.stack.SetConfig(ctx, k, auto.ConfigValue{Value: v})
		}
	}

	workspace := p.stack.Workspace()
	if workspace == nil {
		return errors.New("Failed to get workspace")
	}

	p.outPut <- "Installing plugins \n"
	for _, plugin := range p.plugins {
		workspace.InstallPlugin(ctx, plugin.Kind, plugin.Version)
	}

	p.outPut <- "Plugin installed successfully \n"

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

	stdoutStreamer := optup.ProgressStreams(os.Stdout, p)

	res, err := p.stack.Up(ctx, stdoutStreamer)
	if err != nil {
		return "", errors.Errorf("Failed to update stack %s: %v", p.stackName, err)
	}

	return res.StdOut, nil
}

func (p *PulumiAPI) Destroy(ctx context.Context) (outPut string, err error) {
	err = p.buildPulumiResources(ctx)
	if err != nil {
		return "", err
	}

	stdoutStreamer := optdestroy.ProgressStreams(os.Stdout, p)

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

	stdoutStreamer := optpreview.ProgressStreams(os.Stdout, p)

	res, err := p.stack.Preview(ctx, stdoutStreamer)
	if err != nil {
		return "", errors.Errorf("Failed to preview stack %s: %v", p.stackName, err)
	}

	return res.StdOut, nil
}

func (p *PulumiAPI) Write(content []byte) (int, error) {
	p.outPut <- string(content)
	return len(content), nil
}
