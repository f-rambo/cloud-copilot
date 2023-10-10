## Description

Cluster deployment, application CI/CD, helm chart deployment management

```sh
ocean client command

Usage:
  ocean [flags]
  ocean [command]

Available Commands:
  app         Manage the helm application
  cluster     Manage the k8s cluster
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  service     Manage services

Flags:
      --cluster-grpc-addr string   deployed cluster grpc address
  -h, --help                       help for ocean

Use "ocean [command] --help" for more information about a command.
```

## Getting Started
Must have a docker already running

### Running on the cluster

1. clone code & build:

```sh
git clone https://github.com/f-rambo/ocean.git && cd ocean && make build
```

2. Deploy to docker and install client:

```sh
mv bin/client bin/ocean && go install bin/ocean && docker compose up
```

3. Installing a cluster:

```sh
# Get the cluster sample yaml file and modify it to your own server content
ocean cluster example
ocean cluster apply 
# You can view the deployment progress at http://127.0.0.1:3000
```

4. Deploy ocean in the cluster:

```sh
kubectl apply -f install.yaml
ocean cluster sync cluster-grpc-addr="Your cluster address"
```

5. Deploy app:

```sh
ocean app example
ocean app apply
```

5. Deploy service:

```sh
ocean service example
ocean service apply
```